# HumanSpeak v18 Blueprint

## Goal

HumanSpeak v18 is meant to be a full standalone language that keeps plain-English readability while growing into a serious platform for:
- apps
- websites
- automations
- AI systems
- operating-system tooling
- large multi-file software

## Core language principles

1. Human-first syntax stays.
2. Short aliases are allowed for speed, but the readable form remains first-class.
3. Native runtime only. No Python dependency.
4. One language should cover scripting, applications, services, and AI workflows.
5. The language should scale from beginner code to professional systems.

## Syntax direction

Readable forms:
- `say "hello"`
- `remember score as 10`
- `make a map called app with "name" as "Nova"`
- `task greet using name`
- `for i in range 1 to 10 step by 1`

Short forms:
- `p "hello"`
- `let score be 10`
- `task greet using name`
- `call greet using "Ayush"`

## Runtime pillars

### 1. Native CLI
- `humanspeak run main.hs`
- `humanspeak new myapp`
- `humanspeak test`
- `humanspeak build`
- `humanspeak fmt`
- `humanspeak package add web`

### 2. Module system
- local imports
- package imports
- namespace-safe modules
- lazy load support

### 3. Data model
- numbers
- text
- yes/no
- lists
- maps
- records
- objects
- binary data
- streams

### 4. App model
- terminal apps
- web apps
- desktop apps
- services
- agents

### 5. AI model
- model connectors
- tool calling
- prompt templates
- memory
- workflows
- embeddings and search

## Planned language layers

### Layer A: Solid native scripting
- variables
- loops
- tasks
- files
- json
- maps
- imports

### Layer B: Application development
- packages
- testing
- formatting
- project manifests
- logging
- better errors

### Layer C: Systems and performance
- concurrency
- channels / tasks
- type hints
- compiler backend
- standalone builds

### Layer D: Web and AI
- HTTP server
- routing
- HTML/UI tools
- AI clients
- agents and workflows

## What is already implemented in this workspace

- native Go runtime
- compiled executable
- HumanSpeak-style interpreter
- tasks
- maps
- lists
- range loops
- imports
- files/data/text/math/system built-ins

## Next high-value implementation steps

1. Add package manifests and `package` / `export`
2. Add structured objects and methods
3. Add HTTP server/runtime
4. Add `test` command
5. Add formatter
6. Add typed function parameters and typed maps
7. Add concurrency primitives

This blueprint is the contract for growing HumanSpeak from a native interpreter into a giant language platform.
