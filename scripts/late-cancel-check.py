"""Windows: interrupt an explicitly selected running probe after large spills.

The operator must verify PID, output and temp workspace belong to their benchmark.
This preserves interrupted files; it does not recursively delete anything.
"""
import argparse
import ctypes
from ctypes import wintypes
import importlib.util
import json
from pathlib import Path
import subprocess
import time
from profile_artifacts import profile_files

spec = importlib.util.spec_from_file_location("transport", Path(__file__).with_name("transport-check.py"))
transport = importlib.util.module_from_spec(spec)
spec.loader.exec_module(transport)


class RunningProbe:
    def __init__(self, pid):
        self.pid, self.returncode = pid, None
        self.k = ctypes.WinDLL("kernel32", use_last_error=True)
        self.k.OpenProcess.argtypes = [wintypes.DWORD, wintypes.BOOL, wintypes.DWORD]
        self.k.OpenProcess.restype = wintypes.HANDLE
        self.k.GetExitCodeProcess.argtypes = [wintypes.HANDLE, ctypes.POINTER(wintypes.DWORD)]
        self.k.WaitForSingleObject.argtypes = [wintypes.HANDLE, wintypes.DWORD]
        self.k.TerminateProcess.argtypes = [wintypes.HANDLE, wintypes.UINT]
        self.k.CloseHandle.argtypes = [wintypes.HANDLE]
        self._handle = self.k.OpenProcess(0x100000 | 0x0400 | 0x0010 | 0x0001, False, pid)
        if not self._handle:
            raise ctypes.WinError(ctypes.get_last_error())

    def poll(self):
        code = wintypes.DWORD()
        if not self.k.GetExitCodeProcess(self._handle, ctypes.byref(code)):
            raise ctypes.WinError(ctypes.get_last_error())
        self.returncode = None if code.value == 259 else code.value
        return self.returncode

    def wait(self, timeout=None):
        status = self.k.WaitForSingleObject(self._handle, 0xFFFFFFFF if timeout is None else max(0, int(timeout*1000)))
        if status == 258:
            raise subprocess.TimeoutExpired("selected running probe", timeout)
        if status != 0:
            raise ctypes.WinError(ctypes.get_last_error())
        return self.poll()

    def kill(self):
        if not self.k.TerminateProcess(self._handle, 2):
            raise ctypes.WinError(ctypes.get_last_error())

    def close(self):
        self.k.CloseHandle(self._handle)


def live_size(path):
    # NTFS directory-entry lengths can lag while a writer still has the file open.
    try:
        with path.open("rb") as stream:
            return stream.seek(0, 2)
    except FileNotFoundError:
        return 0


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--pid", type=int, required=True)
    p.add_argument("--output", type=Path, required=True)
    p.add_argument("--workspace", type=Path, required=True)
    p.add_argument("--report", type=Path, required=True)
    p.add_argument("--minimum-events-bytes", type=int, default=6_800_000_000)
    p.add_argument("--minimum-spill-bytes", type=int, default=10_000_000_000)
    p.add_argument("--timeout", type=int, default=3600)
    p.add_argument("--cpu-profile", type=Path)
    p.add_argument("--minimum-profile-segments", type=int, default=0)
    a = p.parse_args()
    if a.minimum_profile_segments < 0 or (a.minimum_profile_segments and not a.cpu_profile):
        raise RuntimeError('A positive segment threshold requires --cpu-profile')
    if a.report.exists() or not a.workspace.is_dir() or not a.workspace.name.startswith("stackguard-trace-"):
        raise RuntimeError("Choose an existing trace workspace and a fresh report path")
    process = RunningProbe(a.pid)
    report = {"pid": a.pid, "operation": "trace", "output": str(a.output), "workspace": str(a.workspace), "complete": False}
    try:
        deadline = time.monotonic()+a.timeout
        while True:
            if process.poll() is not None:
                raise RuntimeError("Probe exited before the requested interruption phase")
            events = live_size(a.output/"events.jsonl")
            spills = sum(live_size(f) for f in a.workspace.rglob("*") if f.is_file())
            segments = 0
            if a.cpu_profile:
                index = Path(str(a.cpu_profile)+'.segments.jsonl')
                if index.exists():
                    segments = index.read_bytes().count(b'\n')
            if events >= a.minimum_events_bytes and spills >= a.minimum_spill_bytes and segments >= a.minimum_profile_segments:
                report.update(events_bytes_before_interrupt=events, spill_bytes_before_interrupt=spills,
                              finalized_cpu_segments_before_interrupt=segments)
                break
            if time.monotonic() >= deadline:
                raise RuntimeError("Requested large-spill phase was not reached before timeout")
            time.sleep(1)
        report["shutdown"] = transport.stop(process)
        manifest = json.loads((a.output/"manifest.json").read_text())
        if manifest["complete"] or not any(str(a.workspace).replace("\\", "/") in n.replace("\\", "/") for n in manifest.get("notes", [])):
            raise RuntimeError("Incomplete manifest did not record retained workspace")
        if not a.workspace.is_dir():
            raise RuntimeError("Interrupted workspace was unexpectedly deleted")
        counts = {}
        for artifact in a.output.glob("*.jsonl"):
            count = 0
            with artifact.open(encoding="utf-8") as source:
                for line in source:
                    json.loads(line)
                    count += 1
            counts[artifact.name] = count
        if counts.get("events.jsonl") != manifest["records"]:
            raise RuntimeError("Manifest record count differs from complete partial events")
        if a.cpu_profile:
            report['cpu_profile_files'] = [p.name for p in profile_files(a.cpu_profile)]
        report.update(complete=True, valid_partial_records=counts, manifest_records=manifest["records"],
                      spill_bytes_retained=sum(live_size(f) for f in a.workspace.rglob("*") if f.is_file()))
    except Exception as exc:
        report["failure"] = str(exc)
        raise
    finally:
        process.close()
        a.report.parent.mkdir(parents=True, exist_ok=True)
        a.report.write_text(json.dumps(report, indent=2)+"\n")
        print(json.dumps(report), flush=True)


if __name__ == "__main__":
    main()
