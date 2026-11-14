"""Verify the selected full-size evidence bundles; never rerun or modify workloads."""
import argparse
import hashlib
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
from bench_budget import check_memory, MEMORY_LIMIT_BYTES


def read(path):
    return json.loads(path.read_text(encoding='utf-8'))


def sha(path):
    with path.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def main():
    if not __debug__:
        raise RuntimeError('Run this verifier without -O or PYTHONOPTIMIZE; assertion checks are required')
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", type=Path, required=True, help="fresh report destination")
    args = parser.parse_args()
    if args.out.exists():
        raise RuntimeError("Refusing to replace an existing verification")
    results = []
    for operation, prefix, records, size in [
        ('scan', 'scan-corpus-full', 137, 5835736856),
        ('trace', 'trace-10m-throughput', 10000000, 5984000000),
    ]:
        expected_hashes = None
        expected_binary = None
        for workers in (1, 4, 8, 16):
            bundle = ROOT / 'results/bench' / f'{prefix}-w{workers}'
            rows = read(bundle / 'summary.json')
            assert len(rows) == 1
            row = rows[0]
            manifest = read(bundle / f'w{workers}/manifest.json')
            assert row['workers'] == workers
            assert row['records'] == manifest['records'] == records
            assert row['bytes'] == manifest['bytes'] == size
            assert manifest['operation'] == operation
            assert manifest['complete'] and manifest['errors'] == 0
            check_memory(row)
            if expected_hashes is None:
                expected_hashes = row['digests']
            assert row['digests'] == expected_hashes
            provenance_path = bundle / 'provenance.json'
            provenance = {}
            if provenance_path.exists():
                provenance = read(provenance_path)
                if expected_binary is None:
                    expected_binary = provenance['probe_sha256']
                assert provenance['probe_sha256'] == expected_binary
                for name, digest in provenance['copied_sha256'].items():
                    assert sha(bundle / name) == digest, (bundle, name)
            else:
                assert operation == 'scan' and workers in (8, 16)
            for name, digest in row['digests'].items():
                artifact = bundle / f'w{workers}' / name
                if artifact.exists():
                    assert sha(artifact) == digest
            if operation == 'trace':
                baseline = read(ROOT / 'results/bench/trace-10m-bounded-profiles-w4/summary.json')[0]
                assert row['digests'] == baseline['digests']
                assert row['profiles'] == 'none'
            results.append(dict(operation=operation, workers=workers,
                                source=str(bundle.relative_to(ROOT)).replace('\\', '/'),
                                records=records, bytes=size, binary_hash_recorded=provenance_path.exists(),
                                binary_hash_provenance=("runner_recorded" if provenance.get("runner_recorded_binary_sha256") else "collected_after_run" if provenance_path.exists() else "not_recorded"),
                                peak_working_set_bytes=row['peak_working_set_bytes'],
                                peak_commit_bytes=row['peak_commit_bytes']))
    for workers, suffix in ((1, 'w1-final'), (4, 'w4-r5'), (8, 'w8-final'), (16, 'w16-final')):
        bundle = ROOT / 'results/bench' / f'transport-{suffix}'
        row = read(bundle / 'summary.json')
        assert row['workers'] == workers
        check_memory(row['proxied_memory'])
        for mode in ('baseline', 'proxied'):
            load = row[mode]
            assert load['complete'] and load['failures'] == 0
            assert load['requests'] == 2000 and load['server_peak_inflight'] >= 1000
        assert row['call_count'] == row['completion_count'] == 2000
        assert row['proxy_shutdown']['exit_code'] == 17
        assert row['proxy_shutdown']['elapsed_seconds'] < 2
        results.append(dict(operation='proxy', workers=workers,
                            source=str(bundle.relative_to(ROOT)).replace('\\', '/'),
                            successful_requests=2000, server_peak_inflight=row['proxied']['server_peak_inflight'],
                            **row['proxied_memory']))
    report = dict(passed=True, memory_limit_bytes=MEMORY_LIMIT_BYTES, rows=results,
                  scope='All twelve selected full-size matrix rows. Both probe OS memory peaks are below the decimal limit. Canonical scan/trace hashes agree across workers; available canonical files and provenance-listed bundle artifacts were rehashed. Earlier failures remain failures. Binary hashes agree within scan/trace rows where recorded; legacy scan 8/16 bundles lack an independent binary-hash record. Paused/shared-resource wall times are not isolated speedups. File replays are not credential-bearing native graph evidence. Scan 1/4 hashes were collected after the run; only the trace runner recorded executable hashes at launch. Scan 8/16 have no independent executable-hash record.')
    dest = args.out
    if dest.exists():
        raise RuntimeError('Refusing to replace an existing final verification')
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
    print(json.dumps(report))


if __name__ == '__main__':
    main()
