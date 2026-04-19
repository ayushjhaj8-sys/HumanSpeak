from __future__ import annotations

from pathlib import Path

from reportlab.lib import colors
from reportlab.lib.enums import TA_CENTER
from reportlab.lib.pagesizes import LETTER
from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
from reportlab.lib.units import inch
from reportlab.platypus import (
    PageBreak,
    Paragraph,
    Preformatted,
    SimpleDocTemplate,
    Spacer,
    Table,
    TableStyle,
)


ROOT = Path(__file__).resolve().parents[1]
OUTPUT = ROOT / "HumanSpeak_Beginner_to_Advanced_Guide.pdf"


def code(text: str) -> Preformatted:
    return Preformatted(
        text.strip("\n"),
        ParagraphStyle(
            "CodeBlock",
            fontName="Courier",
            fontSize=9,
            leading=11,
            textColor=colors.HexColor("#1f2937"),
            backColor=colors.HexColor("#f3f4f6"),
            borderPadding=8,
            leftIndent=0,
            spaceAfter=8,
        ),
    )


def paragraph(text: str, style: ParagraphStyle) -> Paragraph:
    return Paragraph(text, style)


def build_pdf() -> None:
    styles = getSampleStyleSheet()
    title = ParagraphStyle(
        "Title",
        parent=styles["Title"],
        fontName="Helvetica-Bold",
        fontSize=26,
        leading=30,
        alignment=TA_CENTER,
        textColor=colors.HexColor("#0f172a"),
        spaceAfter=12,
    )
    subtitle = ParagraphStyle(
        "Subtitle",
        parent=styles["BodyText"],
        fontName="Helvetica",
        fontSize=12,
        leading=16,
        alignment=TA_CENTER,
        textColor=colors.HexColor("#475569"),
        spaceAfter=18,
    )
    h1 = ParagraphStyle(
        "H1",
        parent=styles["Heading1"],
        fontName="Helvetica-Bold",
        fontSize=18,
        leading=22,
        textColor=colors.HexColor("#111827"),
        spaceBefore=12,
        spaceAfter=8,
    )
    h2 = ParagraphStyle(
        "H2",
        parent=styles["Heading2"],
        fontName="Helvetica-Bold",
        fontSize=13,
        leading=16,
        textColor=colors.HexColor("#1f2937"),
        spaceBefore=8,
        spaceAfter=4,
    )
    body = ParagraphStyle(
        "Body",
        parent=styles["BodyText"],
        fontName="Helvetica",
        fontSize=10.5,
        leading=14,
        textColor=colors.HexColor("#111827"),
        spaceAfter=8,
    )
    small = ParagraphStyle(
        "Small",
        parent=styles["BodyText"],
        fontName="Helvetica",
        fontSize=9,
        leading=12,
        textColor=colors.HexColor("#374151"),
        spaceAfter=6,
    )

    doc = SimpleDocTemplate(
        str(OUTPUT),
        pagesize=LETTER,
        rightMargin=0.75 * inch,
        leftMargin=0.75 * inch,
        topMargin=0.7 * inch,
        bottomMargin=0.7 * inch,
    )

    story = []

    story.append(Spacer(1, 1.1 * inch))
    story.append(paragraph("HumanSpeak", title))
    story.append(paragraph("Beginner to Advanced Guide", subtitle))
    story.append(paragraph(
        "A practical handbook for the native HumanSpeak runtime. "
        "This guide starts from the simplest commands and works up to modules, classes, errors, async jobs, HTTP services, and toolchain workflows.",
        body,
    ))
    story.append(Spacer(1, 0.2 * inch))
    story.append(
        Table(
            [
                ["What you will learn", "Simple speech-like syntax, real projects, and advanced language features"],
                ["Best for", "Beginners, builders, and anyone prototyping apps in a readable language"],
                ["Runtime", "Native HumanSpeak runtime, no Python dependency for execution"],
            ],
            colWidths=[1.7 * inch, 4.8 * inch],
        )
    )
    story[-1].setStyle(
        TableStyle(
            [
                ("BACKGROUND", (0, 0), (-1, -1), colors.HexColor("#f8fafc")),
                ("TEXTCOLOR", (0, 0), (-1, -1), colors.HexColor("#111827")),
                ("FONTNAME", (0, 0), (-1, -1), "Helvetica"),
                ("FONTSIZE", (0, 0), (-1, -1), 9.5),
                ("LEADING", (0, 0), (-1, -1), 12),
                ("GRID", (0, 0), (-1, -1), 0.4, colors.HexColor("#cbd5e1")),
                ("VALIGN", (0, 0), (-1, -1), "TOP"),
                ("PADDING", (0, 0), (-1, -1), 8),
            ]
        )
    )
    story.append(PageBreak())

    sections = [
        (
            "1. What HumanSpeak Is",
            [
                "HumanSpeak is a readable programming language designed to feel close to spoken English while still being precise enough for real software.",
                "The native runtime supports variables, data structures, functions, objects, modules, errors, async jobs, HTTP services, and toolchain commands.",
            ],
            code(
                """
say "Hello from HumanSpeak"
remember name as "Ayush"
say "Welcome, " + name
                """
            ),
        ),
        (
            "2. Getting Started",
            [
                "Save code in a `.hs` file and run it with the native `humanspeak.exe` binary.",
                "The same runtime can also open a REPL for quick experiments.",
            ],
            code(
                """
humanspeak run main.hs
humanspeak check .
humanspeak test .
humanspeak fmt .
humanspeak lint .
                """
            ),
        ),
        (
            "3. Variables and Values",
            [
                "Use `remember` or `let` for assignment. The language keeps both the original long form and faster aliases.",
                "Typed assignments help catch mistakes while staying easy to read.",
            ],
            code(
                """
remember age as number 21
let name be text "Ayush"
change age to 22
say name
say age
                """
            ),
        ),
        (
            "4. Decisions and Loops",
            [
                "Use `if / otherwise / end` for branching.",
                "Use `repeat`, `for each`, and `for ... in range ...` for loops.",
            ],
            code(
                """
if age is more than 18 then
    say "adult"
otherwise
    say "minor"
end

for each hero in heroes
    say hero
end
                """
            ),
        ),
        (
            "5. Tasks, Functions, and Returns",
            [
                "Tasks are HumanSpeak's function-style building blocks.",
                "Use `give back` or `return` to return values.",
            ],
            code(
                """
task greet using name
    say "Hello " + name
    give back "done"
end task

do greet using "World"
                """
            ),
        ),
        (
            "6. Lists, Maps, and Data",
            [
                "Lists store ordered data, maps store key/value data, and both can be updated in place.",
                "These structures are the base for real applications, settings, caches, and API data.",
            ],
            code(
                """
list heroes with "Jinwoo", "Hae-in"
make a map called app with "name" as "Nova", "version" as 18
set "status" in app to "growing"
add "Cha Hae-in" to heroes
                """
            ),
        ),
        (
            "7. Classes and Objects",
            [
                "Classes give HumanSpeak a real object model for more advanced software design.",
                "Objects can store fields and run methods using `this`.",
            ],
            code(
                """
make a class called Person
    field species as "human"

    method greet using other
        say "Hello " + other + ", I am " + this.name
    end method
end class
                """
            ),
        ),
        (
            "8. Modules and Packages",
            [
                "You can split code into files and import the pieces you need.",
                "The native toolchain also supports package workflows for larger projects.",
            ],
            code(
                """
export task square using number
    give back number * number
end task

from "math_tools.hs" import square
say do square using 9
                """
            ),
        ),
        (
            "9. Errors, Async, and Web",
            [
                "Use `try / catch / finally` for structured error handling.",
                "Use `spawn` and `wait for` for background jobs, and `serve on port ... using ...` for web apps.",
            ],
            code(
                """
try
    spawn do heavy work and remember it as job
catch error
    say error
finally
    say "cleanup"
end

serve on port 8090 using handle
                """
            ),
        ),
        (
            "10. Toolchain and Quality",
            [
                "A serious language needs formatting, linting, tests, and a build flow.",
                "HumanSpeak now includes those commands so projects can grow without becoming messy.",
            ],
            code(
                """
humanspeak fmt .
humanspeak lint .
humanspeak check .
humanspeak test .
humanspeak build .
                """
            ),
        ),
        (
            "11. Advanced Project Pattern",
            [
                "The strongest way to use HumanSpeak is as a small project with a clear entry file, modules, tests, and a repeatable workflow.",
                "That is how the language starts to feel like a real platform instead of a script runner.",
            ],
            code(
                """
use "math_tools.hs" as math
make a class called App
    method start
        say "App is starting"
    end method
end class
                """
            ),
        ),
        (
            "12. Quick Reference",
            [
                "Core verbs: `say`, `remember`, `change`, `add`, `subtract`, `remove`, `ask`, `do`, `give back`.",
                "Core structures: `if`, `repeat`, `for each`, `task`, `class`, `map`, `list`, `try`, `catch`, `finally`.",
                "Tooling: `run`, `check`, `test`, `build`, `fmt`, `lint`, `package`.",
            ],
            None,
        ),
    ]

    for title_text, bullets, code_block in sections:
        story.append(paragraph(title_text, h1))
        for bullet in bullets:
            story.append(paragraph(f"• {bullet}", body))
        if code_block is not None:
            story.append(code_block)

    story.append(paragraph("Learning path", h1))
    story.append(paragraph("Beginner: `say`, `remember`, `if`, `repeat`, `tasks`.", body))
    story.append(paragraph("Intermediate: lists, maps, files, modules, and packages.", body))
    story.append(paragraph("Advanced: classes, async jobs, web servers, errors, and toolchain workflows.", body))
    story.append(paragraph(
        "HumanSpeak is now positioned as a practical native language runtime with a readable syntax and room to grow into bigger systems.",
        small,
    ))

    doc.build(story)


if __name__ == "__main__":
    build_pdf()
