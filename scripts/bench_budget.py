"""Assignment memory ceiling: decimal MB, with missing measurements failing closed."""

MEMORY_LIMIT_BYTES = 512_000_000


def check_memory(memory):
    for key in ("peak_working_set_bytes", "peak_commit_bytes"):
        value = memory.get(key)
        if not isinstance(value, int) or isinstance(value, bool) or value <= 0:
            raise RuntimeError(f"missing or invalid {key}")
        if value >= MEMORY_LIMIT_BYTES:
            raise RuntimeError(f"{key} exceeded the 512 MB (512,000,000 byte) budget")
