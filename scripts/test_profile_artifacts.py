import gzip
import json
from pathlib import Path
import tempfile
import unittest

from profile_artifacts import profile_files


class ProfileArtifactsTests(unittest.TestCase):
    # Gzip protocol fixtures only; these do not represent measured profiles.
    def test_legacy_and_segmented_framing(self):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory)/'cpu.pprof'
            base.write_bytes(gzip.compress(b'protocol-test'))
            self.assertEqual(profile_files(base), [base])
            part = Path(str(base)+'.part-000001')
            part.write_bytes(gzip.compress(b'protocol-test'))
            index = Path(str(base)+'.segments.jsonl')
            index.write_text('\n'.join(json.dumps(dict(index=i, file=f.name))
                                      for i, f in enumerate([base, part]))+'\n')
            self.assertEqual(profile_files(base), [base, part])
            part.write_bytes(part.read_bytes()[:-3])
            with self.assertRaisesRegex(RuntimeError, 'incomplete'):
                profile_files(base)

    def test_rejects_unindexed_missing_and_unsafe_names(self):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory)/'cpu.pprof'
            base.write_bytes(gzip.compress(b'protocol-test'))
            index = Path(str(base)+'.segments.jsonl')
            for rows in [[], [dict(index=0, file='../private')],
                         [dict(index=0, file=base.name), dict(index=1, file=base.name+'.part-000001')]]:
                index.write_text(''.join(json.dumps(r)+'\n' for r in rows))
                with self.assertRaises(RuntimeError):
                    profile_files(base)
            index.write_text(json.dumps(dict(index=0, file=base.name))+'\n')
            Path(str(base)+'.part-000001').write_bytes(gzip.compress(b'extra'))
            with self.assertRaisesRegex(RuntimeError, 'unindexed'):
                profile_files(base)


if __name__ == '__main__':
    unittest.main()
