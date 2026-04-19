# HumanSpeak

HumanSpeak is a native, readable programming language with no Python dependency at runtime.

It is built for people who want code that feels natural, but still works for real software:
- apps
- websites
- data tools
- AI workflows
- scripts and automation

## What it can do

- print text with `say`, `print`, `show`, or `p`
- store values with `remember` and `let`
- use typed values like `remember age as number 21`
- work with lists, maps, objects, and classes
- create tasks with `task` and `do`
- handle errors with `try`, `catch`, and `finally`
- run background work with `spawn` and `wait for`
- serve web apps with `serve on port ... using ...`
- read and write files
- import local HumanSpeak modules
- use project commands like `check`, `test`, `build`, `fmt`, and `lint`

## Install

### Windows

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

### macOS and Linux

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

### Build from source

If you have Git and Go installed:

```bash
git clone https://github.com/ayushjhaj8-sys/HumanSpeak.git
cd HumanSpeak
go build -o humanspeak ./cmd/humanspeak
```

## Guides

- Beginner to advanced PDF guide: [HumanSpeak_Beginner_to_Advanced_Guide.pdf](HumanSpeak_Beginner_to_Advanced_Guide.pdf)

## Release files

- Windows install: `install.ps1`
- macOS/Linux install: `install.sh`
- Release bundles: GitHub Releases

## Example

```hs
say "Hello from HumanSpeak"
remember name as "Ayush"
say "Welcome, " + name
```
