"""Measure real repository reads and Windows process shutdown; no provider tokens."""
import argparse
import ctypes
from ctypes import wintypes
import json
from pathlib import Path
import socket
import subprocess
import threading
import time
from bench_budget import check_memory


class Counters(ctypes.Structure):
    _fields_ = [("cb", wintypes.DWORD), ("PageFaultCount", wintypes.DWORD)] + [
        (name, ctypes.c_size_t) for name in (
            "PeakWorkingSetSize", "WorkingSetSize", "QuotaPeakPagedPoolUsage",
            "QuotaPagedPoolUsage", "QuotaPeakNonPagedPoolUsage", "QuotaNonPagedPoolUsage",
            "PagefileUsage", "PeakPagefileUsage", "PrivateUsage")]


def start(args):
    startup = subprocess.STARTUPINFO()
    startup.dwFlags |= subprocess.STARTF_USESHOWWINDOW
    startup.wShowWindow = 0
    return subprocess.Popen(args, stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                            stderr=subprocess.DEVNULL, startupinfo=startup,
                            creationflags=subprocess.CREATE_NEW_CONSOLE)


def stop(process):
    if process.poll() is not None:
        return {"exit_code": process.returncode, "elapsed_seconds": 0}
    kernel = ctypes.WinDLL("kernel32", use_last_error=True)
    began = time.monotonic()
    kernel.FreeConsole()
    if not kernel.AttachConsole(process.pid):
        process.kill()
        process.wait()
        raise RuntimeError("could not attach to hidden child console for Ctrl-C")
    kernel.SetConsoleCtrlHandler(None, True)
    sent = kernel.GenerateConsoleCtrlEvent(0, 0)
    time.sleep(0.05)
    kernel.FreeConsole()
    kernel.SetConsoleCtrlHandler(None, False)
    if not sent:
        process.kill()
        process.wait()
        raise RuntimeError("could not send Ctrl-C")
    try:
        process.wait(timeout=2)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait()
        raise RuntimeError("Ctrl-C shutdown exceeded two seconds")
    elapsed = time.monotonic() - began
    if process.returncode != 17 or elapsed > 2:
        raise RuntimeError("Ctrl-C did not return 17 within two seconds")
    return {"exit_code": process.returncode, "elapsed_seconds": elapsed}


def wait_port(port, process):
    deadline = time.monotonic() + 20
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError("service exited before listening")
        try:
            with socket.create_connection(("127.0.0.1", port), timeout=0.2):
                return
        except OSError:
            time.sleep(0.05)
    raise RuntimeError("service did not listen")


def measure(process, done, memory):
    psapi = ctypes.WinDLL("psapi", use_last_error=True)
    fn = psapi.GetProcessMemoryInfo
    fn.argtypes = [wintypes.HANDLE, ctypes.POINTER(Counters), wintypes.DWORD]
    fn.restype = wintypes.BOOL
    while not done.is_set():
        counters = Counters()
        counters.cb = ctypes.sizeof(counters)
        if fn(wintypes.HANDLE(int(process._handle)), ctypes.byref(counters), counters.cb):
            memory["peak_working_set_bytes"] = max(memory.get("peak_working_set_bytes", 0), counters.PeakWorkingSetSize)
            memory["peak_commit_bytes"] = max(memory.get("peak_commit_bytes", 0), counters.PeakPagefileUsage)
        done.wait(0.02)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", required=True, type=Path)
    parser.add_argument("--out", required=True, type=Path)
    parser.add_argument("--workers", type=int, default=4)
    parser.add_argument("--concurrency", type=int, default=1000)
    parser.add_argument("--bin-dir", type=Path, default=Path(__file__).resolve().parents[1] / "bin")
    args = parser.parse_args()
    workspace = Path(__file__).resolve().parents[1]
    args.out.mkdir(parents=True, exist_ok=False)
    scout = args.bin_dir.resolve() / "oss-scout.exe"
    probe = args.bin_dir.resolve() / "nhi-probe.exe"
    driver = args.bin_dir.resolve() / "sg-bench.exe"
    summary = {"workers": args.workers, "repository": str(args.repo),
               "memory_method": "Windows GetProcessMemoryInfo OS peak working set and peak commit; sampled every 20 ms; probe separate from server/driver",
               "correctness_scope": "real repository-file MCP transport, not native Codex/cloud correctness"}
    for mode in ("baseline", "proxied"):
        server = start([str(scout), "serve", "--files", str(args.repo), "--barrier", str(args.concurrency), "--listen", "127.0.0.1:18766"])
        proxy = None
        monitor = None
        done = threading.Event()
        memory = {}
        try:
            wait_port(18766, server)
            port = 18766
            if mode == "proxied":
                port = 18765
                proxy = start([str(probe), "proxy", "--mode", "http", "--listen", "127.0.0.1:18765", "--upstream", "http://127.0.0.1:18766/mcp", "--agent", "file-workload-client", "--client", "load-driver", "--workers", str(args.workers), "--out", str(args.out / "capture")])
                monitor = threading.Thread(target=measure, args=(proxy, done, memory))
                monitor.start()
                wait_port(port, proxy)
            completed = subprocess.run([str(driver), "--url", f"http://127.0.0.1:{port}/mcp", "--path", "README.md", "--concurrency", str(args.concurrency), "--requests", str(args.concurrency * 2), "--out", str(args.out / mode)], stdin=subprocess.DEVNULL, capture_output=True, timeout=180)
            if completed.returncode != 0:
                diagnostics = workspace / ".work" / "bench-attempts" / args.out.name
                diagnostics.mkdir(parents=True, exist_ok=True)
                (diagnostics / f"{mode}-stderr.txt").write_bytes(completed.stderr)
                raise RuntimeError(f"{mode} load failed with exit {completed.returncode}; inspect load.json if present and private diagnostics at {diagnostics}")
            row = json.loads((args.out / mode / "load.json").read_text())
            if row["server_peak_inflight"] < args.concurrency or row["failures"]:
                raise RuntimeError("required actual server concurrency was not achieved")
            summary[mode] = row
        finally:
            try:
                if proxy:
                    summary["proxy_shutdown"] = stop(proxy)
            finally:
                done.set()
                if monitor:
                    monitor.join()
                summary[mode + "_memory"] = memory
                stop(server)
    for percentile in ("p50_ns", "p99_ns"):
        summary["added_" + percentile] = summary["proxied"][percentile] - summary["baseline"][percentile]
    events = args.out / "capture" / "proxy.jsonl"
    calls = completions = 0
    with events.open() as source:
        for line in source:
            event = json.loads(line)
            calls += event["kind"] == "tool_call"
            completions += event["kind"] == "tool_completion"
    if calls != args.concurrency * 2 or completions != calls:
        raise RuntimeError("proxy call/completion counts differ from actual requests")
    manifest = json.loads((args.out / "capture" / "manifest.json").read_text())
    if manifest["records"] != calls + completions or manifest["complete"]:
        raise RuntimeError("interrupted manifest does not describe valid partial evidence")
    summary["call_count"] = calls
    summary["completion_count"] = completions
    check_memory(summary["proxied_memory"])
    (args.out / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    print(json.dumps(summary))


if __name__ == "__main__":
    main()
