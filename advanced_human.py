#!/usr/bin/env python3
"""
Advanced HumanSpeak prototype.

This keeps the plain-English flavor of HumanSpeak while adding:
- shorter aliases for frequent commands
- richer built-in modules
- better project scaffolding
- more serious runtime structure in one file

It is still a Python bootstrap prototype, not yet a native standalone compiler.
"""

from __future__ import annotations

import datetime as _datetime
import json
import math
import os
import platform
import random
import re
import subprocess
import sys
import time
import urllib.parse
import urllib.request
import webbrowser
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable

VERSION = "10.0.0-prototype"

BANNER = f"""
HumanSpeak Advanced v{VERSION}
Plain English programming with room to grow.
"""

HELP = """
Commands:
  python advanced_human.py run <file.hs>
  python advanced_human.py <file.hs>
  python advanced_human.py new <project-name>
  python advanced_human.py version
  python advanced_human.py help
  python advanced_human.py

Examples:
  say "hello world"
  p "fast alias"
  remember score as 10
  let name be "Ayush"
  list heroes with "Jinwoo", "Hae-in"
"""


class HSError(Exception):
    pass


class ReturnSignal(Exception):
    def __init__(self, value: Any):
        self.value = value


class BreakSignal(Exception):
    pass


class ContinueSignal(Exception):
    pass


class ExitSignal(Exception):
    def __init__(self, code: int = 0):
        self.code = code


class HSList(list):
    def first(self):
        return self[0] if self else None

    def last(self):
        return self[-1] if self else None

    def size(self):
        return len(self)


class Scope:
    def __init__(self, parent: "Scope | None" = None):
        self.values: dict[str, Any] = {}
        self.parent = parent

    def get(self, name: str) -> Any:
        if name in self.values:
            return self.values[name]
        if self.parent:
            return self.parent.get(name)
        return None

    def set(self, name: str, value: Any) -> None:
        self.values[name] = value

    def update(self, name: str, value: Any) -> None:
        if name in self.values:
            self.values[name] = value
            return
        if self.parent and self.parent.has(name):
            self.parent.update(name, value)
            return
        self.values[name] = value

    def has(self, name: str) -> bool:
        return name in self.values or (self.parent.has(name) if self.parent else False)

    def flatten(self) -> dict[str, Any]:
        merged = self.parent.flatten() if self.parent else {}
        merged.update(self.values)
        return merged


class MathModule:
    @staticmethod
    def sqrt(x):
        return math.sqrt(float(x))

    @staticmethod
    def power(x, y):
        return float(x) ** float(y)

    @staticmethod
    def random_number(low=0, high=100):
        return random.randint(int(low), int(high))

    @staticmethod
    def average_of(items: Iterable[Any]):
        nums = [float(x) for x in items]
        return sum(nums) / len(nums) if nums else 0


class TextModule:
    @staticmethod
    def uppercase(value):
        return str(value).upper()

    @staticmethod
    def lowercase(value):
        return str(value).lower()

    @staticmethod
    def split_by(value, sep=" "):
        return HSList(str(value).split(str(sep)))

    @staticmethod
    def join_with(items, sep=" "):
        return str(sep).join(str(x) for x in items)

    @staticmethod
    def contains_text(value, part):
        return str(part).lower() in str(value).lower()


class FilesModule:
    @staticmethod
    def read(path):
        return Path(str(path)).read_text(encoding="utf-8")

    @staticmethod
    def write(path, content):
        Path(str(path)).write_text(str(content), encoding="utf-8")
        return True

    @staticmethod
    def append(path, content):
        with Path(str(path)).open("a", encoding="utf-8") as handle:
            handle.write(str(content))
            handle.write("\n")
        return True

    @staticmethod
    def exists(path):
        return Path(str(path)).exists()

    @staticmethod
    def list_folder(path="."):
        return HSList(item.name for item in Path(str(path)).iterdir())

    @staticmethod
    def make_folder(path):
        Path(str(path)).mkdir(parents=True, exist_ok=True)
        return True

    @staticmethod
    def read_json(path):
        return json.loads(FilesModule.read(path))

    @staticmethod
    def write_json(path, data):
        FilesModule.write(path, json.dumps(data, indent=2))
        return True


class TimeModule:
    @staticmethod
    def now():
        return _datetime.datetime.now().strftime("%I:%M %p")

    @staticmethod
    def today():
        return _datetime.datetime.now().strftime("%B %d, %Y")

    @staticmethod
    def timestamp():
        return time.time()


class SystemModule:
    @staticmethod
    def run_command(command):
        result = subprocess.run(str(command), shell=True, capture_output=True, text=True)
        return (result.stdout or "") + (result.stderr or "")

    @staticmethod
    def get_os():
        return platform.system()

    @staticmethod
    def clear_screen():
        os.system("cls" if os.name == "nt" else "clear")
        return True

    @staticmethod
    def open_browser(url):
        webbrowser.open(str(url))
        return True

    @staticmethod
    def search_web(query):
        url = "https://www.google.com/search?q=" + urllib.parse.quote(str(query))
        webbrowser.open(url)
        return url


class WebModule:
    @staticmethod
    def get(url):
        with urllib.request.urlopen(str(url)) as response:
            return response.read().decode("utf-8", errors="replace")

    @staticmethod
    def download(url, save_as):
        with urllib.request.urlopen(str(url)) as response:
            Path(str(save_as)).write_bytes(response.read())
        return True


class DataModule:
    @staticmethod
    def json_parse(text):
        return json.loads(str(text))

    @staticmethod
    def json_text(data):
        return json.dumps(data, indent=2)


class AIModule:
    @staticmethod
    def draft(prompt):
        return f"[AI draft placeholder] {prompt}"

    @staticmethod
    def summarize(text):
        words = str(text).split()
        return " ".join(words[: min(40, len(words))])


BUILTINS = {
    "math": MathModule,
    "text": TextModule,
    "files": FilesModule,
    "time": TimeModule,
    "system": SystemModule,
    "web": WebModule,
    "data": DataModule,
    "ai": AIModule,
}


@dataclass
class TaskDef:
    params: list[str]
    body: list[str]
    closure: Scope


class Interpreter:
    def __init__(self):
        self.global_scope = Scope()
        self.tasks: dict[str, TaskDef] = {}
        self._setup_builtins()

    def _setup_builtins(self):
        g = self.global_scope
        g.set("yes", True)
        g.set("no", False)
        g.set("true", True)
        g.set("false", False)
        g.set("nothing", None)
        g.set("empty", None)
        g.set("PI", math.pi)
        for name, value in BUILTINS.items():
            g.set(name, value)

    def run(self, source: str):
        lines = source.splitlines()
        try:
            self._exec(lines, 0, len(lines), self.global_scope)
        except ExitSignal as signal:
            raise SystemExit(signal.code) from signal

    def _exec(self, lines: list[str], start: int, end: int, scope: Scope):
        i = start
        while i < end:
            raw = lines[i]
            line = raw.strip()
            if not line or line.startswith("#") or line.startswith("//"):
                i += 1
                continue
            i = self._exec_line(lines, i, end, scope)

    def _exec_line(self, lines: list[str], i: int, end: int, scope: Scope) -> int:
        line = lines[i].strip()
        lower = line.lower()

        m = re.match(r'^(?:say|print|show|p)\s+(.+)$', line, re.I)
        if m:
            print(self._display(self._eval(m.group(1), scope)))
            return i + 1

        if lower == "say the current time":
            print(TimeModule.now())
            return i + 1

        if lower == "say the current date":
            print(TimeModule.today())
            return i + 1

        m = re.match(r'^(?:remember|let|set)\s+(.+?)\s+(?:as|be)\s+(.+)$', line, re.I)
        if m:
            scope.set(m.group(1).strip(), self._eval(m.group(2).strip(), scope))
            return i + 1

        m = re.match(r'^change\s+(.+?)\s+to\s+(.+)$', line, re.I)
        if m:
            scope.update(m.group(1).strip(), self._eval(m.group(2).strip(), scope))
            return i + 1

        m = re.match(r'^add\s+(.+?)\s+to\s+(.+)$', line, re.I)
        if m:
            value = self._eval(m.group(1).strip(), scope)
            name = m.group(2).strip()
            current = scope.get(name)
            if isinstance(current, list):
                current.append(value)
            else:
                scope.update(name, (current or 0) + value)
            return i + 1

        m = re.match(r'^subtract\s+(.+?)\s+from\s+(.+)$', line, re.I)
        if m:
            value = self._eval(m.group(1).strip(), scope)
            name = m.group(2).strip()
            current = scope.get(name) or 0
            scope.update(name, float(current) - float(value))
            return i + 1

        m = re.match(r'^(?:ask\s+(.+?)\s+and remember it as|input\s+(.+?)\s+as)\s+(.+)$', line, re.I)
        if m:
            prompt_expr = m.group(1) or m.group(2)
            target = m.group(3)
            answer = input(str(self._eval(prompt_expr.strip(), scope)) + " ")
            scope.set(target.strip(), answer)
            return i + 1

        m = re.match(r'^save\s+(.+?)\s+to file\s+(.+)$', line, re.I)
        if m:
            FilesModule.write(self._eval(m.group(2), scope), self._eval(m.group(1), scope))
            return i + 1

        if lower.startswith("search the web for "):
            SystemModule.search_web(self._eval(line[19:].strip(), scope))
            return i + 1

        m = re.match(r'^(?:make a list called|list)\s+(.+?)(?:\s+with\s+(.+))?$', line, re.I)
        if m:
            name = m.group(1).strip()
            values = HSList()
            if m.group(2):
                values = HSList(self._eval(part.strip(), scope) for part in self._split_args(m.group(2)))
            scope.set(name, values)
            return i + 1

        m = re.match(r'^for each\s+(.+?)\s+in\s+(.+)$', line, re.I)
        if m:
            var_name = m.group(1).strip()
            iterable = self._eval(m.group(2).strip(), scope)
            body, end_i = self._collect_block(lines, i + 1, end, ("end",))
            for item in iterable or []:
                inner = Scope(scope)
                inner.set(var_name, item)
                try:
                    self._exec(body, 0, len(body), inner)
                except BreakSignal:
                    break
                except ContinueSignal:
                    continue
            return end_i + 1

        m = re.match(r'^repeat\s+(.+?)\s+times$', line, re.I)
        if m:
            count = int(self._eval(m.group(1).strip(), scope))
            body, end_i = self._collect_block(lines, i + 1, end, ("end",))
            for _ in range(count):
                try:
                    self._exec(body, 0, len(body), Scope(scope))
                except BreakSignal:
                    break
                except ContinueSignal:
                    continue
            return end_i + 1

        if lower == "keep doing":
            body, until_i = self._collect_block(lines, i + 1, end, ("until ",))
            condition = lines[until_i].strip()[6:].strip()
            safety = 0
            while safety < 100000:
                safety += 1
                try:
                    self._exec(body, 0, len(body), Scope(scope))
                except BreakSignal:
                    break
                except ContinueSignal:
                    pass
                if self._eval_condition(condition, scope):
                    break
            return until_i + 1

        m = re.match(r'^if\s+(.+?)\s+then$', line, re.I)
        if m:
            body, stop_i = self._collect_block(lines, i + 1, end, ("otherwise", "else", "end"))
            stop_text = lines[stop_i].strip().lower() if stop_i < end else "end"
            if self._eval_condition(m.group(1), scope):
                self._exec(body, 0, len(body), Scope(scope))
                if stop_text in ("otherwise", "else"):
                    _, stop_i = self._collect_block(lines, stop_i + 1, end, ("end",))
            elif stop_text in ("otherwise", "else"):
                alt_body, stop_i = self._collect_block(lines, stop_i + 1, end, ("end",))
                self._exec(alt_body, 0, len(alt_body), Scope(scope))
            return stop_i + 1

        if lower == "try":
            try_body, fail_i = self._collect_block(lines, i + 1, end, ("if it fails",))
            fail_body, end_i = self._collect_block(lines, fail_i + 1, end, ("end",))
            try:
                self._exec(try_body, 0, len(try_body), Scope(scope))
            except Exception as exc:
                fail_scope = Scope(scope)
                fail_scope.set("error", str(exc))
                self._exec(fail_body, 0, len(fail_body), fail_scope)
            return end_i + 1

        m = re.match(r'^(?:make a task called|task)\s+(.+?)(?:\s+using\s+(.+))?$', line, re.I)
        if m:
            name = m.group(1).strip()
            params = [part.strip() for part in self._split_args(m.group(2) or "")]
            body, end_i = self._collect_block(lines, i + 1, end, ("end task", "end"))
            self.tasks[name] = TaskDef(params=params, body=body, closure=scope)
            return end_i + 1

        m = re.match(r'^(?:remember\s+(.+?)\s+as\s+)?(?:do|call)\s+(.+?)(?:\s+using\s+(.+))?$', line, re.I)
        if m:
            store = m.group(1)
            task_name = m.group(2).strip()
            args = [self._eval(part.strip(), scope) for part in self._split_args(m.group(3) or "")]
            value = self._call_task(task_name, args, scope)
            if store:
                scope.set(store.strip(), value)
            return i + 1

        m = re.match(r'^(?:give back|return)\s+(.+)$', line, re.I)
        if m:
            raise ReturnSignal(self._eval(m.group(1).strip(), scope))

        m = re.match(r'^wait\s+(.+?)\s+seconds?$', line, re.I)
        if m:
            time.sleep(float(self._eval(m.group(1).strip(), scope)))
            return i + 1

        if lower in ("break", "stop loop"):
            raise BreakSignal()

        if lower in ("continue", "next iteration", "skip"):
            raise ContinueSignal()

        if lower in ("stop", "exit", "quit", "stop running"):
            raise ExitSignal(0)

        m = re.match(r'^(?:use|import)\s+(.+)$', line, re.I)
        if m:
            name = m.group(1).strip().lower()
            if name not in BUILTINS:
                raise HSError(f'Unknown module "{name}"')
            scope.set(name, BUILTINS[name])
            return i + 1

        m = re.match(r'^run command\s+(.+)$', line, re.I)
        if m:
            print(SystemModule.run_command(self._eval(m.group(1).strip(), scope)))
            return i + 1

        m = re.match(r'^(\w+)\s+(\w[\w ]*?)(?:\s+(.+))?$', line)
        if m:
            module_name = m.group(1).lower()
            method_name = m.group(2).strip().replace(" ", "_").lower()
            args = [self._eval(part.strip(), scope) for part in self._split_args(m.group(3) or "")]
            module = BUILTINS.get(module_name)
            if module and hasattr(module, method_name):
                result = getattr(module, method_name)(*args)
                if result is not None:
                    print(self._display(result))
                return i + 1

        raise HSError(f"Could not understand line: {line}")

    def _call_task(self, name: str, args: list[Any], scope: Scope):
        task = self.tasks.get(name)
        if not task:
            raise HSError(f'Task "{name}" is not defined')
        inner = Scope(task.closure)
        for param, value in zip(task.params, args):
            inner.set(param, value)
        try:
            self._exec(task.body, 0, len(task.body), inner)
        except ReturnSignal as signal:
            return signal.value
        return None

    def _eval(self, expr: str, scope: Scope):
        expr = expr.strip()
        if expr == "":
            return ""

        if re.match(r'^do\s+.+$', expr, re.I):
            match = re.match(r'^do\s+(.+?)(?:\s+using\s+(.+))?$', expr, re.I)
            if match:
                task_name = match.group(1).strip()
                args = [self._eval(part.strip(), scope) for part in self._split_args(match.group(2) or "")]
                return self._call_task(task_name, args, scope)

        if (expr.startswith('"') and expr.endswith('"')) or (expr.startswith("'") and expr.endswith("'")):
            return expr[1:-1]

        if expr.lower() in ("yes", "true"):
            return True
        if expr.lower() in ("no", "false"):
            return False
        if expr.lower() in ("nothing", "empty", "null", "void"):
            return None

        if expr.startswith("[") and expr.endswith("]"):
            inner = expr[1:-1].strip()
            if not inner:
                return HSList()
            return HSList(self._eval(part.strip(), scope) for part in self._split_args(inner))

        if re.fullmatch(r"-?\d+", expr):
            return int(expr)
        if re.fullmatch(r"-?\d+\.\d+", expr):
            return float(expr)

        env = scope.flatten()
        env.update(
            {
                "HSList": HSList,
                "len": len,
                "str": str,
                "int": int,
                "float": float,
                "bool": bool,
                "sum": sum,
                "min": min,
                "max": max,
                "abs": abs,
                "__builtins__": {},
            }
        )
        translated = self._translate_expression(expr)
        try:
            return eval(translated, env, env)
        except Exception:
            if scope.has(expr):
                return scope.get(expr)
            return expr

    def _translate_expression(self, expr: str) -> str:
        translated = expr
        translated = re.sub(r"\byes\b", "True", translated, flags=re.I)
        translated = re.sub(r"\bno\b", "False", translated, flags=re.I)
        translated = re.sub(r"\bnothing\b|\bempty\b|\bnull\b|\bvoid\b", "None", translated, flags=re.I)
        return translated

    def _eval_condition(self, expr: str, scope: Scope) -> bool:
        translated = expr.strip()
        replacements = {
            " is more than ": " > ",
            " is less than ": " < ",
            " is at least ": " >= ",
            " is at most ": " <= ",
            " is not ": " != ",
            " is ": " == ",
            " contains ": " in ",
            " and ": " and ",
            " or ": " or ",
        }
        for old, new in replacements.items():
            translated = re.sub(re.escape(old), new, translated, flags=re.I)
        result = self._eval(translated, scope)
        return bool(result)

    def _split_args(self, text: str) -> list[str]:
        if not text.strip():
            return []
        parts: list[str] = []
        current = []
        depth = 0
        quote = None
        for char in text:
            if quote:
                current.append(char)
                if char == quote:
                    quote = None
                continue
            if char in ('"', "'"):
                quote = char
                current.append(char)
                continue
            if char in "([":
                depth += 1
            elif char in ")]":
                depth -= 1
            if char == "," and depth == 0:
                parts.append("".join(current).strip())
                current = []
            else:
                current.append(char)
        if current:
            parts.append("".join(current).strip())
        return parts

    def _collect_block(self, lines: list[str], start: int, end: int, endings: tuple[str, ...]):
        block: list[str] = []
        depth = 0
        i = start
        openers = ("if ", "repeat ", "keep doing", "make a task called", "task ", "for each ", "try")
        while i < end:
            current = lines[i].strip()
            current_lower = current.lower()
            if any(current_lower.startswith(opener) for opener in openers):
                depth += 1
            if depth == 0 and any(current_lower == ending or current_lower.startswith(ending) for ending in endings):
                return block, i
            if current_lower in ("end", "end task") and depth > 0:
                depth -= 1
            block.append(lines[i])
            i += 1
        raise HSError("Block was not closed properly")

    def _display(self, value: Any) -> str:
        if isinstance(value, bool):
            return "yes" if value else "no"
        if value is None:
            return "nothing"
        return str(value)


def repl():
    print(BANNER)
    print("Type HumanSpeak code. Type quit to exit.\n")
    interp = Interpreter()
    buffer: list[str] = []
    depth = 0
    openers = ("if ", "repeat ", "keep doing", "make a task called", "task ", "for each ", "try")
    closers = ("end", "end task", "until ")
    while True:
        try:
            prompt = "... " if depth else ">>> "
            line = input(prompt)
        except (EOFError, KeyboardInterrupt):
            print("\nGoodbye!")
            break

        lo = line.strip().lower()
        if lo in ("quit", "exit", "stop", "bye"):
            print("Goodbye!")
            break

        if any(lo.startswith(opener) for opener in openers):
            depth += 1
        if any(lo.startswith(closer) for closer in closers) and depth > 0:
            depth -= 1

        buffer.append(line)
        if depth == 0:
            source = "\n".join(buffer)
            buffer = []
            try:
                interp.run(source)
            except Exception as exc:
                print(f"Error: {exc}")


def run_file(path: str):
    file_path = Path(path)
    if not file_path.exists():
        raise SystemExit(f'File not found: "{path}"')
    interp = Interpreter()
    try:
        interp.run(file_path.read_text(encoding="utf-8"))
    except HSError as exc:
        raise SystemExit(f"HumanSpeak Error: {exc}") from exc


def new_project(name: str):
    root = Path.cwd() / name
    root.mkdir(parents=True, exist_ok=True)
    main_file = root / "main.hs"
    config_file = root / "humanspeak.json"
    main_file.write_text(
        '\n'.join(
            [
                f'# {name}',
                'say "Welcome to advanced HumanSpeak"',
                'remember appName as "Future Builder"',
                'say "Project: " + appName',
                "",
                "task greet using name",
                '    say "Hello " + name',
                "end task",
                "",
                'do greet using "world"',
            ]
        ),
        encoding="utf-8",
    )
    config_file.write_text(
        json.dumps(
            {
                "name": name,
                "language": "HumanSpeak",
                "version": VERSION,
                "entry": "main.hs",
            },
            indent=2,
        ),
        encoding="utf-8",
    )
    print(f'Created project "{name}" at {root}')


def main():
    args = sys.argv[1:]
    if not args:
        repl()
        return

    cmd = args[0].lower()
    if cmd in ("help", "--help", "-h"):
        print(HELP)
        return
    if cmd in ("version", "--version", "-v"):
        print(f"HumanSpeak Advanced v{VERSION}")
        return
    if cmd == "run":
        if len(args) < 2:
            raise SystemExit("Usage: python advanced_human.py run <file.hs>")
        run_file(args[1])
        return
    if cmd == "new":
        if len(args) < 2:
            raise SystemExit("Usage: python advanced_human.py new <project-name>")
        new_project(args[1])
        return
    if cmd.endswith(".hs"):
        run_file(args[0])
        return

    Interpreter().run(" ".join(args))


if __name__ == "__main__":
    main()
