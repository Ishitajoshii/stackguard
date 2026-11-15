"""Render the repository research source into a six-page review PDF.

Authoring helper only: reportlab is not a probe/runtime dependency.
"""
import html
from pathlib import Path
import re

from reportlab.lib import colors
from reportlab.lib.enums import TA_LEFT
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle
from reportlab.lib.units import mm
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, Table, TableStyle, PageBreak

ROOT = Path(__file__).resolve().parents[1]
DEST = ROOT / "output/pdf/stackguard-dossier-draft.pdf"
REPO = "https://github.com/Ishitajoshii/stackguard/blob/main/"
INK = colors.HexColor("#172D42")
MUTED = colors.HexColor("#526573")
ACCENT = colors.HexColor("#176B70")


def inline(text):
    text = text.replace("\u2013", "-").replace("\u2014", " - ").replace("\u2011", "-")
    text = html.escape(text)
    def link(match):
        label, target = match.groups()
        if not target.startswith("https://"):
            target = REPO + target
        return f'<link href="{target}" color="#176B70"><u>{label}</u></link>'
    text = re.sub(r"\[([^\]]+)\]\(([^)]+)\)", link, text)
    text = re.sub(r"\*\*(.+?)\*\*", r"<b>\1</b>", text)
    return re.sub(r"`([^`]+)`", r'<font name="Courier" size="8">\1</font>', text)


def render():
    DEST.parent.mkdir(parents=True, exist_ok=True)
    body = ParagraphStyle("Body", fontName="Helvetica", fontSize=9.2, leading=13,
                          textColor=INK, spaceAfter=9, alignment=TA_LEFT)
    title = ParagraphStyle("Title", parent=body, fontSize=24, leading=28, spaceAfter=13)
    heading = ParagraphStyle("Heading", parent=body, fontSize=16, leading=20, spaceAfter=14)
    cell = ParagraphStyle("Cell", parent=body, fontSize=8, leading=10.5, spaceAfter=0)
    story = []
    lines = (ROOT / "DOSSIER.md").read_text(encoding="utf-8").splitlines()
    i, section = 0, 0
    while i < len(lines):
        line = lines[i].strip()
        if not line:
            i += 1
            continue
        if line.startswith("# "):
            story.append(Paragraph(inline(line[2:]), title))
        elif line.startswith("## "):
            section += 1
            if section > 1:
                story.append(PageBreak())
            story.append(Paragraph(inline(line[3:]), heading))
        elif line.startswith("|"):
            rows = []
            while i < len(lines) and lines[i].strip().startswith("|"):
                columns = [part.strip() for part in lines[i].strip().strip("|").split("|")]
                if not all(re.fullmatch(r":?-+:?", part) for part in columns):
                    rows.append([Paragraph(inline(part), cell) for part in columns])
                i += 1
            widths = {3: [39*mm, 67*mm, 68*mm], 4: [22*mm, 53*mm, 48*mm, 51*mm]}
            table = Table(rows, colWidths=widths[len(rows[0])], repeatRows=1, hAlign="LEFT")
            table.setStyle(TableStyle([
                ("BACKGROUND", (0,0),(-1,0),colors.HexColor("#E2EEF0")),
                ("ROWBACKGROUNDS", (0,1),(-1,-1),[colors.white, colors.HexColor("#F3F6F8")]),
                ("VALIGN", (0,0),(-1,-1),"TOP"),
                ("LEFTPADDING",(0,0),(-1,-1),6), ("RIGHTPADDING",(0,0),(-1,-1),6),
                ("TOPPADDING",(0,0),(-1,-1),7), ("BOTTOMPADDING",(0,0),(-1,-1),7),
                ("LINEBELOW",(0,0),(-1,0),0.6,ACCENT),
            ]))
            story.extend([table, Spacer(1, 11)])
            continue
        else:
            paragraph = [line]
            while i+1 < len(lines) and lines[i+1].strip() and not lines[i+1].startswith(("#", "|")):
                i += 1
                paragraph.append(lines[i].strip())
            story.append(Paragraph(inline(" ".join(paragraph)), body))
        i += 1

    def frame(canvas, doc):
        canvas.saveState()
        canvas.setStrokeColor(ACCENT)
        canvas.line(18*mm, A4[1]-15*mm, A4[0]-18*mm, A4[1]-15*mm)
        canvas.setFillColor(MUTED)
        canvas.setFont("Helvetica", 8)
        canvas.drawString(18*mm, A4[1]-12*mm, "STACKGUARD / RESEARCH DRAFT")
        canvas.drawString(18*mm, 12*mm, "See page 6 for observed evidence and remaining validation gates")
        canvas.drawRightString(A4[0]-18*mm, 12*mm, f"{doc.page}")
        canvas.restoreState()

    doc = SimpleDocTemplate(str(DEST), pagesize=A4, rightMargin=18*mm, leftMargin=18*mm,
                            topMargin=23*mm, bottomMargin=22*mm,
                            title="StackGuard credential research dossier - draft", author="StackGuard")
    doc.build(story, onFirstPage=frame, onLaterPages=frame)
    print(DEST)


if __name__ == "__main__":
    render()
