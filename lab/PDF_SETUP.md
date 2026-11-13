# Local PDF extraction

Use Python with pypdf in the scout/lab layer. The current machine has Python 3.14 and pypdf 6.21.3. Pin the tested package with:

```powershell
python -m pip install -r lab/requirements-pdf.txt
```

The helper will preserve one-based page numbers and extract text locally. No PDF or extracted text is uploaded. Review the resulting skills summary before sending it to a cloud agent.

Text extraction cannot reliably read image-only scans and is not a layout renderer. Encrypted, corrupt, empty and scanned inputs must return an actionable fallback; OCR is outside the initial scope. See the [pypdf extraction documentation](https://pypdf.readthedocs.io/en/stable/user/extract-text.html).

This is a lab dependency, not an additional Go library. Its memory usage is separate from the probe's benchmark measurements. Extracted private text must never be included in the public example data or submission captures.
