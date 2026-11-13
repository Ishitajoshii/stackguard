"""Malformed benchmark-metadata tests; no simulated agent activity."""
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


class CollectorRefusalTests(unittest.TestCase):
    def check_refusal(self, wrong_binary):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root/'source'
            (source/'w4').mkdir(parents=True)
            probe = root/'not-an-executable.bin'
            probe.write_bytes(b'metadata-test-only; never executed')
            row = dict(workers=4, records=100, bytes=100, digests={},
                       peak_working_set_bytes=1, peak_commit_bytes=1)
            if wrong_binary:
                row['probe_sha256'] = '0'*64
            (source/'summary.json').write_text(json.dumps([row]))
            (source/'w4/manifest.json').write_text(json.dumps(dict(
                complete=True, errors=0, operation='trace', records=100, bytes=100)))
            dest = root/'bundle'
            command = [sys.executable, str(Path(__file__).with_name('collect-pipeline-result.py')),
                       '--source', str(source), '--out', str(dest), '--probe', str(probe),
                       '--workers', '4', '--expected-records', '99', '--expected-bytes', '100']
            result = subprocess.run(command, capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(dest.exists())
            self.assertIn('provided executable differs' if wrong_binary else 'unexpected workload size', result.stderr)

    def test_rejects_different_executable(self):
        self.check_refusal(True)

    def test_rejects_wrong_workload_size(self):
        self.check_refusal(False)


if __name__ == '__main__':
    unittest.main()
