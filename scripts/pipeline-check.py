"""Windows scan/trace worker matrix on supplied real inputs; no generated evidence."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import threading
import time
from bench_budget import check_memory
from profile_artifacts import profile_files

spec = importlib.util.spec_from_file_location("transport", Path(__file__).with_name("transport-check.py"))
transport = importlib.util.module_from_spec(spec)
spec.loader.exec_module(transport)


def main():
    p = argparse.ArgumentParser()
    p.add_argument("operation", choices=["scan", "trace"])
    p.add_argument("--input", required=True, help="real repository root or adapter=original capture")
    p.add_argument("--out", type=Path, required=True)
    p.add_argument("--temp", required=True)
    p.add_argument("--workers", default="1,4,8,16")
    p.add_argument("--probe", type=Path, default=Path("bin/nhi-probe.exe"))
    a = p.parse_args()
    a.out.mkdir(parents=True, exist_ok=False)
    reports = []
    expected = None
    with a.probe.open('rb') as executable:
        probe_sha256 = hashlib.file_digest(executable, 'sha256').hexdigest()
    for workers in map(int, a.workers.split(",")):
        dest = a.out / f"w{workers}"
        command = [str(a.probe.resolve()), a.operation,
                   "--root" if a.operation == "scan" else "--input", a.input,
                   "--out", str(dest), "--temp", a.temp, "--workers", str(workers),
                   "--cpu-profile", str(a.out / f"cpu-w{workers}.pprof"),
                   "--heap-profile", str(a.out / f"heap-w{workers}.pprof")]
        if a.operation == "trace":
            command.append("--lineage")
        began = time.monotonic()
        process = transport.start(command)
        done = threading.Event()
        memory = {}
        monitor = threading.Thread(target=transport.measure, args=(process, done, memory))
        monitor.start()
        try:
            code = process.wait()
        finally:
            done.set()
            monitor.join()
        elapsed = time.monotonic() - began
        manifest_path = dest / 'manifest.json'
        manifest = json.loads(manifest_path.read_text()) if manifest_path.exists() else {}
        if code != 0 or not manifest.get("complete"):
            failure = dict(workers=workers, seconds=elapsed, exit_code=code,
                           failure='incomplete process or manifest', manifest=manifest, **memory)
            (a.out / f"failure-w{workers}.json").write_text(json.dumps(failure, indent=2)+"\n")
            raise RuntimeError(f"worker {workers} incomplete, exit {code}")
        names = ["inventory.jsonl"] if a.operation == "scan" else ["events.jsonl", "edges.jsonl", "findings.jsonl", "lineage.txt"]
        digests = {}
        for name in names:
            with (dest / name).open("rb") as source:
                digests[name] = hashlib.file_digest(source, "sha256").hexdigest()
        if expected is not None and expected != digests:
            raise RuntimeError("worker output is nondeterministic")
        expected = digests
        report = dict(workers=workers, seconds=elapsed, bytes=manifest.get("bytes", 0),
                      records=manifest["records"], digests=digests, probe_sha256=probe_sha256, **memory)
        try:
            check_memory(memory)
            report['cpu_profile_files'] = [p.name for p in profile_files(a.out / f'cpu-w{workers}.pprof')]
        except (RuntimeError, ValueError, OSError) as exc:
            (a.out / f'failure-w{workers}.json').write_text(json.dumps(dict(report, failure=str(exc)), indent=2)+'\n')
            raise
        reports.append(report)
        (a.out / "summary.json").write_text(json.dumps(reports, indent=2)+"\n")
        print(json.dumps(report), flush=True)


if __name__ == "__main__":
    main()
