"""Process-level Windows Ctrl-C checks on real repository/capture inputs."""
import argparse
import importlib.util
import json
from pathlib import Path
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
    a = p.parse_args()
    a.out.mkdir(parents=True, exist_ok=False)
    records = []
    for name, args in [("scan", ["--root", a.repo]), ("trace", ["--input", "normalized="+a.events])]:
        dest = a.out/name
        process = transport.start([str(Path("bin/nhi-probe.exe").resolve()), name, *args,
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
    (a.out/"summary.json").write_text(json.dumps(records, indent=2)+"\n")
    print(json.dumps(records))


if __name__ == "__main__":
    main()
