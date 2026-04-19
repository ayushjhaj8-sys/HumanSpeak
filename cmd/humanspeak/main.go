package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"humanspeak/internal/interpreter"
)

const version = "18.0.0-beta"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		repl()
		return
	}

	switch strings.ToLower(args[0]) {
	case "help", "--help", "-h":
		printHelp()
	case "version", "--version", "-v":
		fmt.Printf("HumanSpeak v%s\n", version)
	case "run":
		if len(args) < 2 {
			fatal("usage: humanspeak run <file.hs>")
		}
		runFile(args[1])
	case "new":
		if len(args) < 2 {
			fatal("usage: humanspeak new <project-name>")
		}
		newProject(args[1])
	case "check":
		target := "."
		if len(args) >= 2 {
			target = args[1]
		}
		checkProject(target)
	case "test":
		target := "."
		if len(args) >= 2 {
			target = args[1]
		}
		runTests(target)
	case "build":
		target := "."
		if len(args) >= 2 {
			target = args[1]
		}
		buildProject(target)
	case "fmt":
		target := "."
		if len(args) >= 2 {
			target = args[1]
		}
		formatProject(target)
	case "lint":
		target := "."
		if len(args) >= 2 {
			target = args[1]
		}
		lintProject(target)
	case "package":
		handlePackage(args[1:])
	default:
		if strings.HasSuffix(strings.ToLower(args[0]), ".hs") {
			runFile(args[0])
			return
		}
		code := strings.Join(args, " ")
		engine := interpreter.New(os.Stdin, os.Stdout)
		if err := engine.Run(code); err != nil {
			fatal(err.Error())
		}
	}
}

func repl() {
	fmt.Printf("HumanSpeak v%s\n", version)
	fmt.Println("Type HumanSpeak code. Type quit to exit.")

	engine := interpreter.New(os.Stdin, os.Stdout)
	scanner := bufio.NewScanner(os.Stdin)
	var buffer []string
	depth := 0

	for {
		if depth == 0 {
			fmt.Print(">>> ")
		} else {
			fmt.Print("... ")
		}

		if !scanner.Scan() {
			fmt.Println("\nGoodbye!")
			return
		}

		line := scanner.Text()
		lower := strings.ToLower(strings.TrimSpace(line))
		if lower == "quit" || lower == "exit" || lower == "bye" {
			fmt.Println("Goodbye!")
			return
		}

		if interpreter.StartsBlock(lower) {
			depth++
		}
		if interpreter.EndsBlock(lower) && depth > 0 {
			depth--
		}

		buffer = append(buffer, line)
		if depth == 0 {
			source := strings.Join(buffer, "\n")
			buffer = nil
			if err := engine.Run(source); err != nil {
				fmt.Println("Error:", err)
			}
		}
	}
}

func runFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fatal(err.Error())
	}

	engine := interpreter.New(os.Stdin, os.Stdout)
	engine.SetBaseDir(filepath.Dir(path))
	if err := engine.Run(string(data)); err != nil {
		fatal(err.Error())
	}
}

func newProject(name string) {
	root := filepath.Join(".", name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		fatal(err.Error())
	}

	mainFile := filepath.Join(root, "main.hs")
	configFile := filepath.Join(root, "humanspeak.json")

	mainSource := strings.Join([]string{
		`say "Welcome to native HumanSpeak"`,
		`remember appName as "Future Builder"`,
		`say "Project: " + appName`,
		``,
		`task greet using name`,
		`    say "Hello " + name`,
		`end task`,
		``,
		`do greet using "world"`,
	}, "\n")

	configSource := strings.Join([]string{
		`{`,
		`  "name": "` + name + `",`,
		`  "language": "HumanSpeak",`,
		`  "entry": "main.hs",`,
		`  "version": "` + version + `"`,
		`}`,
	}, "\n")

	if err := os.WriteFile(mainFile, []byte(mainSource), 0o644); err != nil {
		fatal(err.Error())
	}
	if err := os.WriteFile(configFile, []byte(configSource), 0o644); err != nil {
		fatal(err.Error())
	}

	fmt.Printf("Created HumanSpeak project in %s\n", root)
}

func printHelp() {
	fmt.Println("HumanSpeak commands:")
	fmt.Println("  humanspeak run <file.hs>")
	fmt.Println("  humanspeak <file.hs>")
	fmt.Println("  humanspeak new <project-name>")
	fmt.Println("  humanspeak check [project-or-file]")
	fmt.Println("  humanspeak test [project-folder]")
	fmt.Println("  humanspeak build [project-folder]")
	fmt.Println("  humanspeak fmt [project-or-file]")
	fmt.Println("  humanspeak lint [project-or-file]")
	fmt.Println("  humanspeak package init <name>")
	fmt.Println("  humanspeak package add <name> [source-path]")
	fmt.Println("  humanspeak package list")
	fmt.Println("  humanspeak version")
	fmt.Println("  humanspeak help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println(`  say "hello world"`)
	fmt.Println(`  p "fast alias"`)
	fmt.Println(`  remember score as 10`)
	fmt.Println(`  let name be "Ayush"`)
	fmt.Println(`  list heroes with "Jinwoo", "Hae-in"`)
	fmt.Println(`  remember age as number 21`)
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

func checkProject(target string) {
	info, err := os.Stat(target)
	if err != nil {
		fatal(err.Error())
	}
	if info.IsDir() {
		target = filepath.Join(target, "main.hs")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		fatal(err.Error())
	}
	engine := interpreter.New(os.Stdin, io.Discard)
	engine.SetBaseDir(filepath.Dir(target))
	if err := engine.Run(string(data)); err != nil {
		fatal("check failed: " + err.Error())
	}
	fmt.Printf("check passed: %s\n", target)
}

func runTests(target string) {
	testsDir := filepath.Join(target, "tests")
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		fatal(err.Error())
	}
	passed := 0
	failed := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".hs") {
			continue
		}
		testFile := filepath.Join(testsDir, entry.Name())
		data, err := os.ReadFile(testFile)
		if err != nil {
			fmt.Printf("FAIL %s: %v\n", entry.Name(), err)
			failed++
			continue
		}
		engine := interpreter.New(os.Stdin, io.Discard)
		engine.SetBaseDir(filepath.Dir(testFile))
		if err := engine.Run(string(data)); err != nil {
			fmt.Printf("FAIL %s: %v\n", entry.Name(), err)
			failed++
			continue
		}
		fmt.Printf("PASS %s\n", entry.Name())
		passed++
	}
	fmt.Printf("tests finished: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func buildProject(target string) {
	projectDir := target
	info, err := os.Stat(projectDir)
	if err != nil {
		fatal(err.Error())
	}
	if !info.IsDir() {
		fatal("build expects a project directory")
	}
	manifestPath := filepath.Join(projectDir, "humanspeak.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		fatal(err.Error())
	}
	entry := "main.hs"
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err == nil {
		if rawEntry, ok := manifest["entry"].(string); ok && strings.TrimSpace(rawEntry) != "" {
			entry = rawEntry
		}
	}
	buildDir := filepath.Join(projectDir, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		fatal(err.Error())
	}
	sourceEntry := filepath.Join(projectDir, entry)
	targetEntry := filepath.Join(buildDir, entry)
	if err := copyFile(sourceEntry, targetEntry); err != nil {
		fatal(err.Error())
	}
	if err := copyFile(manifestPath, filepath.Join(buildDir, "humanspeak.json")); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("build ready in %s\n", buildDir)
}

func formatProject(target string) {
	files, err := collectHumanSpeakFiles(target)
	if err != nil {
		fatal(err.Error())
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			fatal(err.Error())
		}
		formatted := formatHumanSpeakSource(string(data))
		if err := os.WriteFile(path, []byte(formatted), 0o644); err != nil {
			fatal(err.Error())
		}
		fmt.Printf("formatted %s\n", path)
	}
}

func lintProject(target string) {
	files, err := collectHumanSpeakFiles(target)
	if err != nil {
		fatal(err.Error())
	}
	issues := 0
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("FAIL %s: %v\n", path, err)
			issues++
			continue
		}
		fileIssues := lintHumanSpeakSource(string(data))
		if len(fileIssues) == 0 {
			fmt.Printf("OK   %s\n", path)
			continue
		}
		for _, issue := range fileIssues {
			fmt.Printf("WARN %s: %s\n", path, issue)
		}
		issues += len(fileIssues)
	}
	if issues == 0 {
		fmt.Println("lint passed")
		return
	}
	fmt.Printf("lint finished with %d issue(s)\n", issues)
	os.Exit(1)
}

func collectHumanSpeakFiles(target string) ([]string, error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if strings.HasSuffix(strings.ToLower(target), ".hs") {
			return []string{target}, nil
		}
		return nil, fmt.Errorf("expected a .hs file or directory")
	}
	files := []string{}
	err = filepath.WalkDir(target, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if name == "build" || name == "packages" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".hs") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func formatHumanSpeakSource(source string) string {
	lines := strings.Split(source, "\n")
	formatted := make([]string, 0, len(lines))
	blankRun := 0
	for _, line := range lines {
		trimmedRight := strings.TrimRight(line, " \t")
		if strings.TrimSpace(trimmedRight) == "" {
			blankRun++
			if blankRun > 2 {
				continue
			}
			formatted = append(formatted, "")
			continue
		}
		blankRun = 0
		formatted = append(formatted, trimmedRight)
	}
	return strings.TrimRight(strings.Join(formatted, "\n"), "\n") + "\n"
}

func lintHumanSpeakSource(source string) []string {
	lines := strings.Split(source, "\n")
	issues := []string{}
	depth := 0
	for idx, line := range lines {
		if strings.TrimRight(line, " \t") != line {
			issues = append(issues, fmt.Sprintf("line %d: trailing whitespace", idx+1))
		}
		trimmed := strings.ToLower(strings.TrimSpace(line))
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if interpreter.StartsBlock(trimmed) {
			depth++
		}
		if interpreter.EndsBlock(trimmed) {
			if depth == 0 {
				issues = append(issues, fmt.Sprintf("line %d: unexpected block end", idx+1))
			} else {
				depth--
			}
		}
	}
	if depth > 0 {
		issues = append(issues, "missing closing end for one or more blocks")
	}
	return issues
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func handlePackage(args []string) {
	if len(args) == 0 {
		fatal("usage: humanspeak package <init|add|list> ...")
	}

	switch strings.ToLower(args[0]) {
	case "init":
		if len(args) < 2 {
			fatal("usage: humanspeak package init <name>")
		}
		initPackage(args[1])
	case "add":
		if len(args) < 2 {
			fatal("usage: humanspeak package add <name> [source-path]")
		}
		source := args[1]
		if len(args) >= 3 {
			source = args[2]
		}
		addPackage(args[1], source)
	case "list":
		listPackages()
	default:
		fatal("usage: humanspeak package <init|add|list> ...")
	}
}

func initPackage(name string) {
	root := filepath.Join(".", name)
	if err := os.MkdirAll(filepath.Join(root, "packages"), 0o755); err != nil {
		fatal(err.Error())
	}
	manifest := map[string]any{
		"name":     name,
		"language": "HumanSpeak",
		"packages": []string{},
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "humanspeak.json"), data, 0o644); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("package project created: %s\n", root)
}

func addPackage(name, source string) {
	dest := filepath.Join(".", "packages", name)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		fatal(err.Error())
	}
	info, err := os.Stat(source)
	if err != nil {
		fatal(err.Error())
	}
	if info.IsDir() {
		fatal("package add currently expects a .hs file source path")
	}
	target := filepath.Join(dest, filepath.Base(source))
	if err := copyFile(source, target); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("package added: %s\n", name)
}

func listPackages() {
	dir := filepath.Join(".", "packages")
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("no packages installed")
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Println(entry.Name())
		}
	}
}
