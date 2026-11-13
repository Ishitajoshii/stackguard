"""Collect a completed measured row; retain large event streams on the data drive.

CPU segments are merged by Go pprof. The bundle records hashes of every source
segment, the exact executable, and the copied small artifacts. This does not
turn an incomplete run or a memory-budget failure into a passing result.
"""
import argparse
import gzip
import hashlib
import json
from pathlib import Path
import shutil
import subprocess

from bench_budget import check_memory
from profile_artifacts import profile_files


def digest(path):
    with path.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def valid_gzip(path):
    with gzip.open(path, 'rb') as source:
        total = 0
        while chunk := source.read(64*1024):
            total += len(chunk)
    if total == 0:
        raise RuntimeError('empty profile')


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--source', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--probe', type=Path, required=True)
    parser.add_argument('--workers', type=int, required=True)
    parser.add_argument('--expected-records', type=int, required=True)
    parser.add_argument('--expected-bytes', type=int, required=True)
    a = parser.parse_args()
    source, dest = a.source.resolve(), a.out.resolve()
    if dest.exists() or dest.is_relative_to(source) or not 1 <= a.workers <= 64:
        raise RuntimeError('choose a fresh bundle outside the source run')
    rows = json.loads((source/'summary.json').read_text())
    selected = [row for row in rows if row['workers'] == a.workers]
    if len(selected) != 1:
        raise RuntimeError('completed worker row not found exactly once')
    row = selected[0]
    check_memory(row)
    probe_sha256 = digest(a.probe)
    if row.get('probe_sha256', probe_sha256) != probe_sha256:
        raise RuntimeError('provided executable differs from the measured binary')
    for other in rows:
        if other['digests'] != row['digests']:
            raise RuntimeError('worker output hashes differ')
    run = source/f'w{a.workers}'
    manifest = json.loads((run/'manifest.json').read_text())
    if not manifest.get('complete') or manifest.get('errors') != 0:
        raise RuntimeError('incomplete or erroneous run')
    for key, expected in [('records', a.expected_records), ('bytes', a.expected_bytes)]:
        if manifest[key] != expected or row[key] != expected:
            raise RuntimeError('unexpected workload size')
    if manifest['operation'] not in ('scan', 'trace'):
        raise RuntimeError('unsupported operation')
    cpu_base = source/f'cpu-w{a.workers}.pprof'
    mode = row.get('profiles', 'both')
    if mode not in ('both', 'none'):
        raise RuntimeError('unknown profiling mode')
    parts = profile_files(cpu_base) if mode == 'both' else []
    if 'cpu_profile_files' in row and row['cpu_profile_files'] != [p.name for p in parts]:
        raise RuntimeError('CPU profile list differs from measured summary')
    heap = source/f'heap-w{a.workers}.pprof'
    if mode == 'both':
        valid_gzip(heap)
    build = subprocess.check_output(['go', 'version', '-m', str(a.probe)], text=True)
    dest.mkdir(parents=True)
    small = dest/f'w{a.workers}'
    small.mkdir()

    copied = {}

    def copy_small(path, target):
        if path.stat().st_size > 32*1024*1024:
            raise RuntimeError('artifact exceeds the small-bundle limit; retain it on the data drive')
        shutil.copyfile(path, target)
        copied[str(target.relative_to(dest)).replace('\\', '/')] = digest(target)

    copy_small(run/'manifest.json', small/'manifest.json')
    if manifest['operation'] == 'scan':
        names = ['inventory.jsonl', 'observations.jsonl', 'coverage.jsonl']
    else:
        names = ['edges.jsonl', 'findings.jsonl', 'lineage.txt', 'quarantine.jsonl']
    for name in names:
        path = run/name
        if path.exists():
            copy_small(path, small/name)
            if name in row['digests'] and digest(small/name) != row['digests'][name]:
                raise RuntimeError('copied output differs from measured hash')
    merged = None
    if mode == 'both':
        copy_small(heap, dest/heap.name)
        merged = dest/f'cpu-w{a.workers}-merged.pprof'
        subprocess.run(['go', 'tool', 'pprof', '-proto', '-output', str(merged),
                        *map(str, parts)], check=True, capture_output=True)
        valid_gzip(merged)
        if merged.stat().st_size > 32*1024*1024:
            raise RuntimeError('merged profile exceeds bundle limit')
        copied[merged.name] = digest(merged)
    metadata = dict(source_run=str(source), probe_sha256=probe_sha256,
                    runner_recorded_binary_sha256='probe_sha256' in row,
                    build_info=build, profiles=mode,
                    merged_cpu_profile=merged.name if merged else None,
                    original_cpu_profiles=[dict(file=p.name, bytes=p.stat().st_size,
                                                sha256=digest(p)) for p in parts],
                    copied_sha256=copied,
                    scope='Large events.jsonl and any original CPU segments remain in source_run. Output digests were measured by pipeline-check; copied canonical files were rechecked. When profiles=both, gzip framing and Go pprof merging validate the profiles. Throughput replay is not native credential-lineage evidence.')
    (dest/'provenance.json').write_text(json.dumps(metadata, indent=2)+'\n')
    (dest/'summary.json').write_text(json.dumps([row], indent=2)+'\n')
    print(json.dumps(dict(bundle=str(dest), workers=a.workers,
                          records=row['records'], cpu_segments=len(parts))), flush=True)


if __name__ == '__main__':
    main()
