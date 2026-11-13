"""Conservative prefix audit: report object IDs, never suspected secret values.

This does not detect all formatless secrets or private prose. Review remains
necessary. No candidate is verified against a provider.
"""
import argparse
import json
from pathlib import Path
import re
import subprocess


def git(*args):
    return subprocess.check_output(["git", *args])


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    patterns = [
        rb"(?:gh[pousr]_[A-Za-z0-9]{20,255}|github_pat_[A-Za-z0-9_]{20,255})",
        rb"(?:ntn_|secret_)[A-Za-z0-9]{20,255}",
        rb"sk-ant-[A-Za-z0-9_-]{20,255}",
        rb"sk-(?:proj-|svcacct-)?[A-Za-z0-9_-]{20,255}",
        rb"xox[baprs]-[A-Za-z0-9-]{16,255}",
        rb"AIza[A-Za-z0-9_-]{35}",
        rb"(?:AKIA|ASIA)[A-Z0-9]{16}",
    ]
    pattern = re.compile(b"|".join(patterns))
    objects = sorted({line.split(b" ", 1)[0] for line in git("rev-list", "--objects", "--all").splitlines()})
    process = subprocess.Popen(["git", "cat-file", "--batch"], stdin=subprocess.PIPE, stdout=subprocess.PIPE)
    checked, hits = 0, []
    try:
        for oid in objects:
            process.stdin.write(oid+b"\n")
            process.stdin.flush()
            _, kind, length = process.stdout.readline().split()
            data = process.stdout.read(int(length))
            if len(data) != int(length) or process.stdout.read(1) != b"\n":
                raise RuntimeError("incomplete git object read")
            if kind == b"blob":
                checked += 1
                if pattern.search(data):
                    hits.append(oid.decode("ascii"))
    finally:
        process.stdin.close()
        process.stdout.close()
        if process.wait() != 0:
            raise RuntimeError("git object audit failed")
    tracked = git("ls-files", "-z").decode().split("\0")
    private = [p for p in tracked if p in ("PRD.md", "TODO.md", ".env") or p.startswith(("private/", "profiles/", ".work/", "lab/private/"))]
    ignored = all(subprocess.run(["git", "check-ignore", "-q", name]).returncode == 0 for name in ("PRD.md", "TODO.md"))
    records = 0
    for name in tracked:
        if name.endswith(".jsonl"):
            with Path(name).open(encoding="utf-8") as source:
                for line in source:
                    json.loads(line)
                    records += 1
    result = dict(history_blobs_checked=checked, pattern_match_object_ids=hits,
                  private_paths_tracked=private, planning_files_ignored=ignored,
                  valid_tracked_jsonl_records=records,
                  scope="Seven provider prefix families across reachable history blobs; private-path exclusions; tracked JSONL syntax. Formatless secrets and private prose require separate review. No provider verification.")
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(result, indent=2)+"\n", encoding="utf-8")
    print(json.dumps(result))
    if hits or private or not ignored:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
