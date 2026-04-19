# HumanSpeak Native

This is a real standalone HumanSpeak runtime written in Go.

Goals:
- no Python dependency
- keep HumanSpeak readable
- support the original syntax and shorter aliases
- grow toward apps, AI systems, websites, and larger software

Current native features:
- `say`, `print`, `show`, `p`
- `remember x as y`, `let x be y`
- typed assignments like `remember age as number 21` and `let name be text "Ayush"`
- `export remember ...`, `export task ...`
- `export make a class called ...`
- `change x to y`
- `add x to y`
- `subtract x from y`
- `remove x from list`
- `ask ... and remember it as ...`
- local `use "module.hs"` imports
- `use "module.hs" as tools`
- `from "module.hs" import square, cube`
- `make a map called ... with "key" as value`
- `set "key" in map to value`
- `make a class called ...`
- `make an object called ... as ClassName with ...`
- `method greet using ...`
- `list heroes with ...`
- `repeat`
- `keep doing ... until ...`
- `for i in range 1 to 10 step by 1`
- `for each`
- `if / otherwise / end`
- `task / do / give back`
- `try / catch / finally`
- `throw` / `raise`
- `spawn` / `wait for`
- `serve on port ... using ...`
- `wait`
- `open file ... and remember it as ...`
- `save ... to file ...`
- `run command`
- built-in modules: `math`, `text`, `files`, `time`, `data`, `system`
- network helpers: `web get`, `web download`
- HTTP server runtime
- REPL
- project scaffolding
- project commands: `check`, `test`, `build`
- toolchain commands: `fmt`, `lint`
- package commands: `package init`, `package add`, `package list`

Real project example:
- [real_language_app/main.hs](C:\Users\ayush\Documents\Codex\2026-04-19-files-mentioned-by-the-user-humanspeak\real_language_app\main.hs)
- [real_language_app/math_tools.hs](C:\Users\ayush\Documents\Codex\2026-04-19-files-mentioned-by-the-user-humanspeak\real_language_app\math_tools.hs)
- [real_language_app/tests/math_test.hs](C:\Users\ayush\Documents\Codex\2026-04-19-files-mentioned-by-the-user-humanspeak\real_language_app\tests\math_test.hs)

Teaching guide:
- [HumanSpeak_Beginner_to_Advanced_Guide.pdf](C:\Users\ayush\Documents\Codex\2026-04-19-files-mentioned-by-the-user-humanspeak\HumanSpeak_Beginner_to_Advanced_Guide.pdf)

Install on Windows:

1. Open PowerShell.
2. Run:

```powershell
powershell -ExecutionPolicy Bypass -File .\install.ps1
```

3. Restart PowerShell or Windows Terminal.
4. Run:

```powershell
humanspeak.exe version
```

Install on macOS or Linux:

1. Open Terminal.
2. Run:

```bash
curl -fsSL https://raw.githubusercontent.com/ayushjhaj8-sys/HumanSpeak/main/install.sh | bash
```

3. Restart Terminal.
4. Run:

```bash
humanspeak version
```

Install from source on any machine with Git and Go:

1. Clone the repo.
2. Build the binary.
3. Run the resulting `humanspeak` executable.

```powershell
git clone https://github.com/ayushjhaj8-sys/HumanSpeak.git
cd HumanSpeak
go build -o humanspeak.exe .\cmd\humanspeak
.\humanspeak.exe version
```

Release packaging:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\package_release.ps1
```

Build:

```powershell
& "C:\Program Files\Go\bin\go.exe" build -o humanspeak.exe .\cmd\humanspeak
```

Run:

```powershell
.\humanspeak.exe run .\examples\native_demo.hs
```

This is now a stronger beta-level native runtime. It is still not the final ceiling of HumanSpeak, but it is beyond a thin alpha shell and is ready to grow into packages, web runtimes, AI connectors, app tooling, and a richer type system.

Tooling examples:

```powershell
.\humanspeak.exe fmt .\real_language_app
.\humanspeak.exe lint .\real_language_app
```
