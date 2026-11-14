"""Process-level Windows Ctrl-C checks on real repository/capture inputs."""
import argparse
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import time

spec = importlib.util.spec_from_file_location("transport", Path(__file__).with_name("transport-check.py"))
transport = importlib.util.module_from_spec(spec)
spec.loader.exec_module(transport)


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--repo", required=True)
    p.add_argument("--events", required=True)
    p.add_argument("--temp", required=True)
    p.add_argument("--out", type=Path, required=True)
    p.add_argument("--probe", type=Path, default=Path("bin/nhi-probe.exe"))
    p.add_argument('--nested-child', action='store_true', help=argparse.SUPPRESS)
    a = p.parse_args()
    a.out.mkdir(parents=True, exist_ok=False)
    records = []
    for name, args in [("scan", ["--root", a.repo]), ("trace", ["--input", "normalized="+a.events])]:
        dest = a.out/name
        process = transport.start([str(a.probe.resolve()), name, *args,
                                   "--out", str(dest), "--temp", a.temp, "--workers", "4"])
        # Wait until the Go entry point installed its signal handler and opened
        # an output, rather than racing Windows process/console creation.
        deadline = time.monotonic() + 10
        while not dest.exists() and time.monotonic() < deadline:
            if process.poll() is not None:
                raise RuntimeError("probe exited before cancellation check")
            time.sleep(0.02)
        time.sleep(0.1)
        result = transport.stop(process)
        manifest = json.loads((dest/"manifest.json").read_text())
        if result["exit_code"] != 17 or manifest["complete"]:
            raise RuntimeError("interruption contract failed")
        count = 0
        for artifact in dest.glob("*.jsonl"):
            with artifact.open(encoding="utf-8") as source:
                for line in source:
                    json.loads(line)
                    count += 1
        records.append(dict(operation=name, valid_partial_jsonl_records=count, **result))
    if not a.nested_child:
        # The nested child must itself send Ctrl-C before printing its result.
        # Intentionally inherit ALL streams: redirecting even stdin causes
        # Python to set STARTF_USESTDHANDLES and masks the original Windows bug.
        # This is a real process-control check, not simulated agent activity.
        child_out = a.out/'inherited-child'
        child = subprocess.run([sys.executable, str(Path(__file__).resolve()),
                                '--repo', a.repo, '--events', a.events,
                                '--temp', a.temp, '--out', str(child_out),
                                '--probe', str(a.probe.resolve()), '--nested-child'], timeout=60)
        status = dict(exit_code=child.returncode,
                      child_report_present=(child_out/'summary.json').exists())
        (a.out/'inherited-child-status.json').write_text(json.dumps(status, indent=2)+'\n')
        if child.returncode != 0 or not status['child_report_present']:
            raise RuntimeError('nested Ctrl-C helper failed with inherited standard streams')
        for record in records:
            record['inherited_stream_child_exit_code'] = child.returncode
    (a.out/"summary.json").write_text(json.dumps(records, indent=2)+"\n")
    print(json.dumps(records))


if __name__ == "__main__":
    main()
