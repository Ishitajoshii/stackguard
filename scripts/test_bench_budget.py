import unittest

from bench_budget import MEMORY_LIMIT_BYTES, check_memory


class MemoryBudgetTests(unittest.TestCase):
    def test_boundary(self):
        check_memory(dict(peak_working_set_bytes=MEMORY_LIMIT_BYTES-1,
                          peak_commit_bytes=MEMORY_LIMIT_BYTES-1))
        for field in ("peak_working_set_bytes", "peak_commit_bytes"):
            for value in (MEMORY_LIMIT_BYTES, 512*1024*1024, None, 0, -1, True):
                with self.subTest(field=field, value=value):
                    memory = dict(peak_working_set_bytes=1, peak_commit_bytes=1)
                    memory[field] = value
                    with self.assertRaises(RuntimeError):
                        check_memory(memory)

    def test_missing_measurement(self):
        with self.assertRaises(RuntimeError):
            check_memory({})


if __name__ == "__main__":
    unittest.main()
