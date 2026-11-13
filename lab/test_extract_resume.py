"""PDF parser error fixtures are not agent evidence or benchmark captures."""

import tempfile
import unittest
from pathlib import Path

from pypdf import PdfWriter

from extract_resume import ResumeError, extract_pages


class PDFErrors(unittest.TestCase):
    def test_missing_file(self):
        with tempfile.TemporaryDirectory() as root:
            with self.assertRaisesRegex(ResumeError, "not found"):
                extract_pages(Path(root) / "missing.pdf")

    def test_malformed_does_not_echo_content(self):
        with tempfile.TemporaryDirectory() as root:
            source = Path(root) / "invalid file.pdf"
            source.write_bytes(b"sensitive-content-not-to-be-echoed")
            with self.assertRaises(ResumeError) as result:
                extract_pages(source)
            self.assertNotIn("sensitive-content", str(result.exception))

    def test_blank_pdf_requests_text_fallback(self):
        with tempfile.TemporaryDirectory() as root:
            source = Path(root) / "blank file.pdf"
            writer = PdfWriter()
            writer.add_blank_page(width=100, height=100)
            writer.write(source)
            with self.assertRaisesRegex(ResumeError, "No extractable text"):
                extract_pages(source)

    def test_encrypted_pdf_requests_unlocked_copy(self):
        with tempfile.TemporaryDirectory() as root:
            source = Path(root) / "encrypted.pdf"
            writer = PdfWriter()
            writer.add_blank_page(width=100, height=100)
            writer.encrypt("test-only-password")
            writer.write(source)
            with self.assertRaisesRegex(ResumeError, "Encrypted"):
                extract_pages(source)


if __name__ == "__main__":
    unittest.main()
