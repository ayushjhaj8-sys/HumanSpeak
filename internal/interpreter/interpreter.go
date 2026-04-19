package interpreter

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Value = any

type Task struct {
	Params []string
	Body   []string
}

type ClassDef struct {
	Name    string
	Fields  map[string]Value
	Methods map[string]Task
}

type HSObject struct {
	ClassName string
	Fields    map[string]Value
}

type Job struct {
	done chan jobResult
}

type jobResult struct {
	value Value
	err   error
}

type Scope struct {
	values map[string]Value
	types  map[string]string
	parent *Scope
}

func NewScope(parent *Scope) *Scope {
	return &Scope{values: map[string]Value{}, types: map[string]string{}, parent: parent}
}

func (s *Scope) Get(name string) (Value, bool) {
	if value, ok := s.values[name]; ok {
		return value, true
	}
	if s.parent != nil {
		return s.parent.Get(name)
	}
	return nil, false
}

func (s *Scope) Set(name string, value Value) {
	s.values[name] = value
}

func (s *Scope) SetType(name, typeName string) {
	s.types[name] = strings.ToLower(strings.TrimSpace(typeName))
}

func (s *Scope) GetType(name string) (string, bool) {
	if value, ok := s.types[name]; ok {
		return value, true
	}
	if s.parent != nil {
		return s.parent.GetType(name)
	}
	return "", false
}

func (s *Scope) Update(name string, value Value) {
	if _, ok := s.values[name]; ok {
		s.values[name] = value
		return
	}
	if s.parent != nil {
		if _, ok := s.parent.Get(name); ok {
			s.parent.Update(name, value)
			return
		}
	}
	s.values[name] = value
}

func (s *Scope) Resolve(name string) (*Scope, bool) {
	if _, ok := s.values[name]; ok {
		return s, true
	}
	if s.parent != nil {
		return s.parent.Resolve(name)
	}
	return nil, false
}

type Interpreter struct {
	in              *bufio.Reader
	out             io.Writer
	tasks           map[string]Task
	classes         map[string]ClassDef
	root            *Scope
	baseDir         string
	imported        map[string]bool
	moduleCache     map[string]ModuleSnapshot
	exportedValues  map[string]bool
	exportedTasks   map[string]bool
	exportedClasses map[string]bool
}

type ModuleSnapshot struct {
	Values  map[string]Value
	Tasks   map[string]Task
	Classes map[string]ClassDef
}

func New(in io.Reader, out io.Writer) *Interpreter {
	root := NewScope(nil)
	root.Set("yes", true)
	root.Set("no", false)
	root.Set("true", true)
	root.Set("false", false)
	root.Set("nothing", nil)
	root.Set("empty", nil)
	return &Interpreter{
		in:              bufio.NewReader(in),
		out:             out,
		tasks:           map[string]Task{},
		classes:         map[string]ClassDef{},
		root:            root,
		baseDir:         ".",
		imported:        map[string]bool{},
		moduleCache:     map[string]ModuleSnapshot{},
		exportedValues:  map[string]bool{},
		exportedTasks:   map[string]bool{},
		exportedClasses: map[string]bool{},
	}
}

func (i *Interpreter) SetBaseDir(path string) {
	if strings.TrimSpace(path) == "" {
		i.baseDir = "."
		return
	}
	i.baseDir = path
}

func StartsBlock(line string) bool {
	line = stripExportPrefix(strings.TrimSpace(line))
	return strings.HasPrefix(line, "if ") ||
		strings.HasPrefix(line, "try") ||
		strings.HasPrefix(line, "repeat ") ||
		strings.HasPrefix(line, "keep doing") ||
		strings.HasPrefix(line, "for ") ||
		strings.HasPrefix(line, "for each ") ||
		strings.HasPrefix(line, "method ") ||
		strings.HasPrefix(line, "task ") ||
		strings.HasPrefix(line, "make a task called ") ||
		strings.HasPrefix(line, "make a class called ")
}

func EndsBlock(line string) bool {
	line = stripExportPrefix(strings.TrimSpace(line))
	return line == "end" || line == "end task" || line == "end method" || line == "end class" || strings.HasPrefix(line, "until ")
}

func stripExportPrefix(line string) string {
	lower := strings.ToLower(strings.TrimSpace(line))
	if strings.HasPrefix(lower, "export ") {
		return strings.TrimSpace(line[len("export "):])
	}
	return line
}

func (i *Interpreter) Run(source string) error {
	lines := normalizeLines(source)
	_, err := i.execBlock(lines, 0, len(lines), i.root)
	return err
}

func normalizeLines(source string) []string {
	raw := strings.Split(source, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func (i *Interpreter) execBlock(lines []string, start, end int, scope *Scope) (int, error) {
	index := start
	for index < end {
		next, err := i.execLine(lines, index, end, scope)
		if err != nil {
			return index, err
		}
		index = next
	}
	return index, nil
}

func (i *Interpreter) execLine(lines []string, index, end int, scope *Scope) (int, error) {
	line := strings.TrimSpace(lines[index])
	lower := strings.ToLower(line)
	exportMode := false

	if strings.HasPrefix(lower, "export ") {
		exportMode = true
		line = strings.TrimSpace(line[len("export "):])
		lower = strings.ToLower(line)
	}

	if expr, ok := matchAnyPrefix(line, []string{"say ", "print ", "show ", "p "}); ok {
		value, err := i.eval(expr, scope)
		if err != nil {
			return index, err
		}
		fmt.Fprintln(i.out, display(value))
		return index + 1, nil
	}

	if lower == "say the current time" {
		fmt.Fprintln(i.out, time.Now().Format("03:04 PM"))
		return index + 1, nil
	}

	if lower == "say the current date" {
		fmt.Fprintln(i.out, time.Now().Format("January 02, 2006"))
		return index + 1, nil
	}

	if rest, ok := matchAnyPrefix(line, []string{"remember ", "let ", "set "}); ok {
		if strings.Contains(strings.ToLower(rest), " as ") {
			parts := splitTwo(rest, " as ")
			name := strings.TrimSpace(parts[0])
			typeName, expr := parseTypedAssignment(parts[1])
			value, err := i.eval(expr, scope)
			if err != nil {
				return index, err
			}
			if typeName != "" {
				if err := ensureAssignableType(typeName, value); err != nil {
					return index, fmt.Errorf("%s", err)
				}
				scope.SetType(name, typeName)
			} else if existingType, ok := scope.GetType(name); ok {
				if err := ensureAssignableType(existingType, value); err != nil {
					return index, fmt.Errorf("%s", err)
				}
			}
			scope.Set(name, value)
			if exportMode {
				i.exportedValues[name] = true
			}
			return index + 1, nil
		}
		if strings.Contains(strings.ToLower(rest), " be ") {
			parts := splitTwo(rest, " be ")
			name := strings.TrimSpace(parts[0])
			typeName, expr := parseTypedAssignment(parts[1])
			value, err := i.eval(expr, scope)
			if err != nil {
				return index, err
			}
			if typeName != "" {
				if err := ensureAssignableType(typeName, value); err != nil {
					return index, fmt.Errorf("%s", err)
				}
				scope.SetType(name, typeName)
			} else if existingType, ok := scope.GetType(name); ok {
				if err := ensureAssignableType(existingType, value); err != nil {
					return index, fmt.Errorf("%s", err)
				}
			}
			scope.Set(name, value)
			if exportMode {
				i.exportedValues[name] = true
			}
			return index + 1, nil
		}
	}

	if strings.HasPrefix(lower, "change ") && strings.Contains(lower, " to ") {
		parts := splitTwo(line[len("change "):], " to ")
		value, err := i.eval(parts[1], scope)
		if err != nil {
			return index, err
		}
		name := strings.TrimSpace(parts[0])
		if typeName, ok := scope.GetType(name); ok {
			if err := ensureAssignableType(typeName, value); err != nil {
				return index, fmt.Errorf("%s", err)
			}
		}
		scope.Update(name, value)
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "add ") && strings.Contains(lower, " to ") {
		parts := splitTwo(line[len("add "):], " to ")
		value, err := i.eval(parts[0], scope)
		if err != nil {
			return index, err
		}
		name := strings.TrimSpace(parts[1])
		current, _ := scope.Get(name)
		switch typed := current.(type) {
		case []Value:
			scope.Update(name, append(typed, value))
		case int:
			scope.Update(name, typed+toInt(value))
		case float64:
			scope.Update(name, typed+toFloat(value))
		case string:
			scope.Update(name, typed+fmt.Sprint(value))
		case nil:
			scope.Update(name, value)
		default:
			scope.Update(name, fmt.Sprint(current)+fmt.Sprint(value))
		}
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "subtract ") && strings.Contains(lower, " from ") {
		parts := splitTwo(line[len("subtract "):], " from ")
		value, err := i.eval(parts[0], scope)
		if err != nil {
			return index, err
		}
		name := strings.TrimSpace(parts[1])
		current, _ := scope.Get(name)
		scope.Update(name, toFloat(current)-toFloat(value))
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "remove ") && strings.Contains(lower, " from ") {
		parts := splitTwo(line[len("remove "):], " from ")
		value, err := i.eval(parts[0], scope)
		if err != nil {
			return index, err
		}
		name := strings.TrimSpace(parts[1])
		current, _ := scope.Get(name)
		items := toSlice(current)
		filtered := make([]Value, 0, len(items))
		removed := false
		for _, item := range items {
			if !removed && sameValue(item, value) {
				removed = true
				continue
			}
			filtered = append(filtered, item)
		}
		scope.Update(name, filtered)
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "ask ") && strings.Contains(lower, " and remember it as ") {
		parts := splitTwo(line[len("ask "):], " and remember it as ")
		promptValue, err := i.eval(parts[0], scope)
		if err != nil {
			return index, err
		}
		fmt.Fprint(i.out, display(promptValue)+" ")
		answer, readErr := i.in.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return index, readErr
		}
		scope.Set(strings.TrimSpace(parts[1]), strings.TrimSpace(answer))
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "spawn ") {
		rest := strings.TrimSpace(line[len("spawn "):])
		name := ""
		if idx := strings.Index(strings.ToLower(rest), " and remember it as "); idx != -1 {
			name = strings.TrimSpace(rest[idx+len(" and remember it as "):])
			rest = strings.TrimSpace(rest[:idx])
		}
		spawnLine := rest
		job := &Job{done: make(chan jobResult, 1)}
		go func() {
			value, err := i.eval(spawnLine, scope)
			job.done <- jobResult{value: value, err: err}
		}()
		if name != "" {
			scope.Set(name, job)
		}
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "input ") && strings.Contains(lower, " as ") {
		parts := splitTwo(line[len("input "):], " as ")
		promptValue, err := i.eval(parts[0], scope)
		if err != nil {
			return index, err
		}
		fmt.Fprint(i.out, display(promptValue)+" ")
		answer, readErr := i.in.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return index, readErr
		}
		scope.Set(strings.TrimSpace(parts[1]), strings.TrimSpace(answer))
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "wait for ") && strings.Contains(lower, " and remember it as ") {
		rest := strings.TrimSpace(line[len("wait for "):])
		parts := splitTwo(rest, " and remember it as ")
		jobValue, err := i.eval(parts[0], scope)
		if err != nil {
			return index, err
		}
		job, ok := jobValue.(*Job)
		if !ok {
			return index, fmt.Errorf("wait for expects a job")
		}
		result := <-job.done
		if result.err != nil {
			return index, result.err
		}
		scope.Set(strings.TrimSpace(parts[1]), result.value)
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "serve on port ") && strings.Contains(lower, " using ") {
		rest := strings.TrimSpace(line[len("serve on port "):])
		parts := splitTwo(rest, " using ")
		portValue, err := i.eval(parts[0], scope)
		if err != nil {
			return index, err
		}
		handlerName := strings.TrimSpace(parts[1])
		if err := i.serveHTTP(toInt(portValue), handlerName, scope); err != nil {
			return index, err
		}
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "from ") && strings.Contains(lower, " import ") {
		rest := strings.TrimSpace(line[len("from "):])
		parts := splitTwo(rest, " import ")
		modulePath, err := i.evalModulePath(parts[0], scope)
		if err != nil {
			return index, err
		}
		snapshot, err := i.loadModule(modulePath)
		if err != nil {
			return index, err
		}
		for _, name := range splitArgs(parts[1]) {
			clean := strings.TrimSpace(name)
			if value, ok := snapshot.Values[clean]; ok {
				scope.Set(clean, value)
				continue
			}
			if task, ok := snapshot.Tasks[clean]; ok {
				i.tasks[clean] = task
				continue
			}
			if classDef, ok := snapshot.Classes[clean]; ok {
				i.classes[clean] = classDef
				continue
			}
			return index, fmt.Errorf("module %s does not export %s", modulePath, clean)
		}
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "use ") || strings.HasPrefix(lower, "import ") {
		var rest string
		if strings.HasPrefix(lower, "use ") {
			rest = strings.TrimSpace(line[len("use "):])
		} else {
			rest = strings.TrimSpace(line[len("import "):])
		}
		namespace := ""
		if strings.Contains(strings.ToLower(rest), " as ") {
			parts := splitTwo(rest, " as ")
			rest = strings.TrimSpace(parts[0])
			namespace = strings.TrimSpace(parts[1])
		}
		modulePath, err := i.evalModulePath(rest, scope)
		if err != nil {
			return index, err
		}
		if namespace == "" {
			if err := i.runImportedFile(modulePath); err != nil {
				return index, err
			}
			return index + 1, nil
		}
		snapshot, err := i.loadModule(modulePath)
		if err != nil {
			return index, err
		}
		moduleMap := map[string]Value{}
		for key, value := range snapshot.Values {
			moduleMap[key] = value
		}
		for key, task := range snapshot.Tasks {
			i.tasks[namespace+"."+key] = task
			moduleMap[key] = "[task]"
		}
		for key, classDef := range snapshot.Classes {
			i.classes[namespace+"."+key] = classDef
			moduleMap[key] = "[class]"
		}
		scope.Set(namespace, moduleMap)
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "make a class called ") {
		name := strings.TrimSpace(line[len("make a class called "):])
		body, stop, err := collectUntil(lines, index+1, end, []string{"end class"})
		if err != nil {
			return index, err
		}
		classDef, err := i.parseClassBody(name, body, scope)
		if err != nil {
			return index, err
		}
		i.classes[name] = classDef
		if exportMode {
			i.exportedClasses[name] = true
		}
		return stop + 1, nil
	}

	if strings.HasPrefix(lower, "open file ") && strings.Contains(lower, " and remember it as ") {
		parts := splitTwo(line[len("open file "):], " and remember it as ")
		fileValue, err := i.eval(parts[0], scope)
		if err != nil {
			return index, err
		}
		data, readErr := os.ReadFile(fmt.Sprint(fileValue))
		if readErr != nil {
			return index, readErr
		}
		scope.Set(strings.TrimSpace(parts[1]), string(data))
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "save ") && strings.Contains(lower, " to file ") {
		parts := splitTwo(line[len("save "):], " to file ")
		content, err := i.eval(parts[0], scope)
		if err != nil {
			return index, err
		}
		pathValue, err := i.eval(parts[1], scope)
		if err != nil {
			return index, err
		}
		if writeErr := os.WriteFile(fmt.Sprint(pathValue), []byte(fmt.Sprint(content)), 0o644); writeErr != nil {
			return index, writeErr
		}
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "make a map called ") {
		rest := strings.TrimSpace(line[len("make a map called "):])
		name := rest
		values := map[string]Value{}
		if strings.Contains(strings.ToLower(rest), " with ") {
			parts := splitTwo(rest, " with ")
			name = strings.TrimSpace(parts[0])
			for _, entry := range splitArgs(parts[1]) {
				pair := splitTwo(entry, " as ")
				keyValue, err := i.eval(pair[0], scope)
				if err != nil {
					return index, err
				}
				value, err := i.eval(pair[1], scope)
				if err != nil {
					return index, err
				}
				values[fmt.Sprint(keyValue)] = value
			}
		}
		scope.Set(name, values)
		if exportMode {
			i.exportedValues[name] = true
		}
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "make an object called ") || strings.HasPrefix(lower, "make object ") {
		name, className, dataExpr, err := parseObjectHeader(line)
		if err != nil {
			return index, err
		}
		obj, err := i.createObject(className, dataExpr, scope)
		if err != nil {
			return index, err
		}
		scope.Set(name, obj)
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "list ") || strings.HasPrefix(lower, "make a list called ") {
		var rest string
		if strings.HasPrefix(lower, "list ") {
			rest = line[len("list "):]
		} else {
			rest = line[len("make a list called "):]
		}
		name := strings.TrimSpace(rest)
		items := []Value{}
		if strings.Contains(strings.ToLower(rest), " with ") {
			parts := splitTwo(rest, " with ")
			name = strings.TrimSpace(parts[0])
			for _, part := range splitArgs(parts[1]) {
				item, err := i.eval(part, scope)
				if err != nil {
					return index, err
				}
				items = append(items, item)
			}
		}
		scope.Set(name, items)
		if exportMode {
			i.exportedValues[name] = true
		}
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "for ") && strings.Contains(lower, " in range ") && strings.Contains(lower, " to ") {
		rest := line[len("for "):]
		rangeParts := splitTwo(rest, " in range ")
		name := strings.TrimSpace(rangeParts[0])
		step := 1
		rangeBody := rangeParts[1]
		if strings.Contains(strings.ToLower(rangeBody), " step by ") {
			rangeStepParts := splitTwo(rangeBody, " step by ")
			rangeBody = rangeStepParts[0]
			stepValue, err := i.eval(rangeStepParts[1], scope)
			if err != nil {
				return index, err
			}
			step = toInt(stepValue)
			if step == 0 {
				step = 1
			}
		}
		bounds := splitTwo(rangeBody, " to ")
		startValue, err := i.eval(bounds[0], scope)
		if err != nil {
			return index, err
		}
		endValue, err := i.eval(bounds[1], scope)
		if err != nil {
			return index, err
		}
		body, stop, err := collectUntil(lines, index+1, end, []string{"end"})
		if err != nil {
			return index, err
		}
		startNum := toInt(startValue)
		endNum := toInt(endValue)
		if step > 0 {
			for current := startNum; current <= endNum; current += step {
				inner := NewScope(scope)
				inner.Set(name, current)
				if _, err := i.execBlock(body, 0, len(body), inner); err != nil {
					return index, err
				}
			}
		} else {
			for current := startNum; current >= endNum; current += step {
				inner := NewScope(scope)
				inner.Set(name, current)
				if _, err := i.execBlock(body, 0, len(body), inner); err != nil {
					return index, err
				}
			}
		}
		return stop + 1, nil
	}

	if lower == "keep doing" {
		body, stop, err := collectUntilPrefix(lines, index+1, end, "until ")
		if err != nil {
			return index, err
		}
		untilLine := strings.TrimSpace(lines[stop])
		condition := strings.TrimSpace(untilLine[len("until "):])
		safety := 0
		for {
			safety++
			if safety > 100000 {
				return index, fmt.Errorf("loop safety stop: keep doing ran too many times")
			}
			if _, err := i.execBlock(body, 0, len(body), NewScope(scope)); err != nil {
				return index, err
			}
			ok, err := i.evalCondition(condition, scope)
			if err != nil {
				return index, err
			}
			if ok {
				break
			}
		}
		return stop + 1, nil
	}

	if strings.HasPrefix(lower, "repeat ") && strings.HasSuffix(lower, " times") {
		countExpr := strings.TrimSpace(line[len("repeat ") : len(line)-len(" times")])
		countValue, err := i.eval(countExpr, scope)
		if err != nil {
			return index, err
		}
		body, stop, err := collectUntil(lines, index+1, end, []string{"end"})
		if err != nil {
			return index, err
		}
		for range toInt(countValue) {
			if _, err := i.execBlock(body, 0, len(body), NewScope(scope)); err != nil {
				return index, err
			}
		}
		return stop + 1, nil
	}

	if strings.HasPrefix(lower, "for each ") && strings.Contains(lower, " in ") {
		rest := line[len("for each "):]
		parts := splitTwo(rest, " in ")
		itemName := strings.TrimSpace(parts[0])
		iterable, err := i.eval(parts[1], scope)
		if err != nil {
			return index, err
		}
		body, stop, err := collectUntil(lines, index+1, end, []string{"end"})
		if err != nil {
			return index, err
		}
		for _, item := range toSlice(iterable) {
			inner := NewScope(scope)
			inner.Set(itemName, item)
			if _, err := i.execBlock(body, 0, len(body), inner); err != nil {
				return index, err
			}
		}
		return stop + 1, nil
	}

	if strings.HasPrefix(lower, "if ") && strings.HasSuffix(lower, " then") {
		condExpr := strings.TrimSpace(line[len("if ") : len(line)-len(" then")])
		trueBody, stop, marker, err := collectIfBlocks(lines, index+1, end)
		if err != nil {
			return index, err
		}
		ok, err := i.evalCondition(condExpr, scope)
		if err != nil {
			return index, err
		}
		if ok {
			if _, err := i.execBlock(trueBody, 0, len(trueBody), NewScope(scope)); err != nil {
				return index, err
			}
		} else if marker == "otherwise" || marker == "else" {
			falseBody, stop2, err := collectUntil(lines, stop+1, end, []string{"end"})
			if err != nil {
				return index, err
			}
			if _, err := i.execBlock(falseBody, 0, len(falseBody), NewScope(scope)); err != nil {
				return index, err
			}
			return stop2 + 1, nil
		}
		if marker == "otherwise" || marker == "else" {
			_, stop2, err := collectUntil(lines, stop+1, end, []string{"end"})
			if err != nil {
				return index, err
			}
			return stop2 + 1, nil
		}
		return stop + 1, nil
	}

	if strings.HasPrefix(lower, "task ") || strings.HasPrefix(lower, "make a task called ") {
		var rest string
		if strings.HasPrefix(lower, "task ") {
			rest = line[len("task "):]
		} else {
			rest = line[len("make a task called "):]
		}
		name := strings.TrimSpace(rest)
		params := []string{}
		if strings.Contains(strings.ToLower(rest), " using ") {
			parts := splitTwo(rest, " using ")
			name = strings.TrimSpace(parts[0])
			for _, part := range splitArgs(parts[1]) {
				params = append(params, strings.TrimSpace(part))
			}
		}
		body, stop, err := collectUntil(lines, index+1, end, []string{"end task"})
		if err != nil {
			return index, err
		}
		i.tasks[name] = Task{Params: params, Body: body}
		if exportMode {
			i.exportedTasks[name] = true
		}
		return stop + 1, nil
	}

	if lower == "try" {
		tryBody, catchBody, finallyBody, catchName, stop, err := collectTrySections(lines, index+1, end)
		if err != nil {
			return index, err
		}

		var runErr error
		if _, runErr = i.execBlock(tryBody, 0, len(tryBody), NewScope(scope)); runErr != nil {
			var ret returnSignal
			if errors.As(runErr, &ret) {
				return index, runErr
			}
			var thrown throwSignal
			if errors.As(runErr, &thrown) || runErr != nil {
				if len(catchBody) > 0 {
					catchScope := NewScope(scope)
					if catchName == "" {
						catchName = "error"
					}
					if errors.As(runErr, &thrown) {
						catchScope.Set(catchName, thrown.value)
					} else {
						catchScope.Set(catchName, runErr.Error())
					}
					if _, catchErr := i.execBlock(catchBody, 0, len(catchBody), catchScope); catchErr != nil {
						return index, catchErr
					}
					runErr = nil
				}
			}
		}

		if len(finallyBody) > 0 {
			if _, finalErr := i.execBlock(finallyBody, 0, len(finallyBody), NewScope(scope)); finalErr != nil {
				return index, finalErr
			}
		}

		if runErr != nil {
			return index, runErr
		}
		return stop + 1, nil
	}

	if strings.HasPrefix(lower, "do ") || strings.HasPrefix(lower, "call ") || (strings.HasPrefix(lower, "remember ") && strings.Contains(lower, " as do ")) {
		var targetName string
		callLine := line
		if strings.HasPrefix(lower, "remember ") && strings.Contains(lower, " as do ") {
			parts := splitTwo(line[len("remember "):], " as ")
			targetName = strings.TrimSpace(parts[0])
			callLine = strings.TrimSpace(parts[1])
		}
		result, err := i.callTask(callLine, scope)
		if err != nil {
			return index, err
		}
		if targetName != "" {
			scope.Set(targetName, result)
		}
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "give back ") || strings.HasPrefix(lower, "return ") {
		expr := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "give back "), "return "))
		value, err := i.eval(expr, scope)
		if err != nil {
			return index, err
		}
		return index, returnSignal{value: value}
	}

	if strings.HasPrefix(lower, "wait ") && strings.HasSuffix(lower, " seconds") {
		secondsExpr := strings.TrimSpace(line[len("wait ") : len(line)-len(" seconds")])
		value, err := i.eval(secondsExpr, scope)
		if err != nil {
			return index, err
		}
		time.Sleep(time.Duration(toFloat(value) * float64(time.Second)))
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "run command ") {
		commandValue, err := i.eval(strings.TrimSpace(line[len("run command "):]), scope)
		if err != nil {
			return index, err
		}
		cmd := exec.Command("cmd", "/C", fmt.Sprint(commandValue))
		out, err := cmd.CombinedOutput()
		if err != nil && len(out) == 0 {
			return index, err
		}
		fmt.Fprintln(i.out, strings.TrimRight(string(out), "\r\n"))
		return index + 1, nil
	}

	if strings.HasPrefix(lower, "throw ") || strings.HasPrefix(lower, "raise ") {
		expr := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "throw "), "raise "))
		value, err := i.eval(expr, scope)
		if err != nil {
			return index, err
		}
		return index, throwSignal{value: value}
	}

	if strings.HasPrefix(lower, "set ") && strings.Contains(lower, " in ") && strings.Contains(lower, " to ") {
		rest := line[len("set "):]
		keyAndTarget := splitTwo(rest, " to ")
		left := splitTwo(keyAndTarget[0], " in ")
		keyValue, err := i.eval(left[0], scope)
		if err != nil {
			return index, err
		}
		targetName := strings.TrimSpace(left[1])
		newValue, err := i.eval(keyAndTarget[1], scope)
		if err != nil {
			return index, err
		}
		current, _ := scope.Get(targetName)
		record, ok := current.(map[string]Value)
		if ok {
			record[fmt.Sprint(keyValue)] = newValue
			scope.Update(targetName, record)
			return index + 1, nil
		}
		obj, ok := current.(HSObject)
		if !ok {
			record = map[string]Value{}
		} else {
			obj.Fields[fmt.Sprint(keyValue)] = newValue
			scope.Update(targetName, obj)
			return index + 1, nil
		}
		record[fmt.Sprint(keyValue)] = newValue
		scope.Update(targetName, record)
		return index + 1, nil
	}

	if result, handled, err := i.tryBuiltinCall(line, scope); handled {
		if err != nil {
			return index, err
		}
		if result != nil {
			fmt.Fprintln(i.out, display(result))
		}
		return index + 1, nil
	}

	return index, fmt.Errorf("could not understand line: %s", line)
}

func parseTypedAssignment(expr string) (string, string) {
	trimmed := strings.TrimSpace(expr)
	parts := strings.Fields(trimmed)
	if len(parts) < 2 {
		return "", trimmed
	}
	candidate := strings.ToLower(parts[0])
	switch candidate {
	case "number", "text", "string", "bool", "boolean", "list", "map", "object", "any":
		return candidate, strings.TrimSpace(trimmed[len(parts[0]):])
	default:
		return "", trimmed
	}
}

func ensureAssignableType(typeName string, value Value) error {
	if typeName == "" || strings.ToLower(strings.TrimSpace(typeName)) == "any" {
		return nil
	}
	actual := inferTypeName(value)
	switch strings.ToLower(strings.TrimSpace(typeName)) {
	case "number":
		if actual == "number" {
			return nil
		}
	case "text", "string":
		if actual == "text" {
			return nil
		}
	case "bool", "boolean":
		if actual == "bool" {
			return nil
		}
	case "list":
		if actual == "list" {
			return nil
		}
	case "map":
		if actual == "map" {
			return nil
		}
	case "object":
		if actual == "object" {
			return nil
		}
	default:
		return nil
	}
	return fmt.Errorf("type mismatch: expected %s, got %s", typeName, actual)
}

func inferTypeName(value Value) string {
	switch value.(type) {
	case nil:
		return "nil"
	case int, int8, int16, int32, int64, float32, float64:
		return "number"
	case string:
		return "text"
	case bool:
		return "bool"
	case []Value:
		return "list"
	case map[string]Value:
		return "map"
	case HSObject:
		return "object"
	case *Job:
		return "job"
	case Task:
		return "task"
	case ClassDef:
		return "class"
	default:
		return fmt.Sprintf("%T", value)
	}
}

type returnSignal struct {
	value Value
}

func (r returnSignal) Error() string {
	return "return"
}

type throwSignal struct {
	value Value
}

func (t throwSignal) Error() string {
	return fmt.Sprint(t.value)
}

func (i *Interpreter) callTask(line string, scope *Scope) (Value, error) {
	lower := strings.ToLower(line)
	if strings.HasPrefix(lower, "call ") {
		line = line[len("call "):]
	} else if strings.HasPrefix(lower, "do ") {
		line = line[len("do "):]
	}

	taskName := strings.TrimSpace(line)
	args := []Value{}
	if strings.Contains(strings.ToLower(line), " using ") {
		parts := splitTwo(line, " using ")
		taskName = strings.TrimSpace(parts[0])
		for _, part := range splitArgs(parts[1]) {
			value, err := i.eval(part, scope)
			if err != nil {
				return nil, err
			}
			args = append(args, value)
		}
	}

	return i.invokeTask(taskName, args, scope)
}

func (i *Interpreter) invokeTask(taskName string, args []Value, scope *Scope) (Value, error) {
	if strings.Contains(taskName, ".") {
		parts := strings.SplitN(taskName, ".", 2)
		objName := strings.TrimSpace(parts[0])
		methodName := strings.TrimSpace(parts[1])
		objValue, ok := scope.Get(objName)
		if !ok {
			return nil, fmt.Errorf("object not found: %s", objName)
		}
		obj, ok := objValue.(HSObject)
		if !ok {
			return nil, fmt.Errorf("%s is not an object", objName)
		}
		classDef, ok := i.classes[obj.ClassName]
		if !ok {
			return nil, fmt.Errorf("class not found: %s", obj.ClassName)
		}
		method, ok := classDef.Methods[methodName]
		if !ok {
			return nil, fmt.Errorf("method not found: %s", methodName)
		}
		inner := NewScope(scope)
		inner.Set("this", obj)
		for key, value := range obj.Fields {
			inner.Set(key, value)
		}
		for idx, param := range method.Params {
			if idx < len(args) {
				inner.Set(param, args[idx])
			} else {
				inner.Set(param, nil)
			}
		}
		return i.runTaskBody(method.Body, inner)
	}

	task, ok := i.tasks[taskName]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskName)
	}

	inner := NewScope(scope)
	for idx, param := range task.Params {
		if idx < len(args) {
			inner.Set(param, args[idx])
		} else {
			inner.Set(param, nil)
		}
	}

	return i.runTaskBody(task.Body, inner)
}

func (i *Interpreter) runTaskBody(body []string, scope *Scope) (Value, error) {
	_, err := i.execBlock(body, 0, len(body), scope)
	if err != nil {
		var ret returnSignal
		if errors.As(err, &ret) {
			return ret.value, nil
		}
		return nil, err
	}
	return nil, nil
}

func (i *Interpreter) serveHTTP(port int, handlerName string, scope *Scope) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := ioutil.ReadAll(r.Body)
		reqObj := HSObject{
			ClassName: "Request",
			Fields: map[string]Value{
				"method":  r.Method,
				"path":    r.URL.Path,
				"query":   r.URL.RawQuery,
				"body":    string(bodyBytes),
				"headers": map[string]Value{},
			},
		}
		headerMap := map[string]Value{}
		for key, values := range r.Header {
			items := make([]Value, 0, len(values))
			for _, value := range values {
				items = append(items, value)
			}
			headerMap[key] = items
		}
		reqObj.Fields["headers"] = headerMap

		inner := NewScope(scope)
		inner.Set("request", reqObj)
		result, err := i.invokeTask(handlerName, nil, inner)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		status := http.StatusOK
		headers := map[string]Value{}
		body := ""

		switch typed := result.(type) {
		case nil:
			body = ""
		case string:
			body = typed
		case HSObject:
			if v, ok := typed.Fields["status"]; ok {
				status = toInt(v)
			}
			if v, ok := typed.Fields["headers"]; ok {
				if m, ok := v.(map[string]Value); ok {
					headers = m
				}
			}
			if v, ok := typed.Fields["body"]; ok {
				body = fmt.Sprint(v)
			}
		case map[string]Value:
			if v, ok := typed["status"]; ok {
				status = toInt(v)
			}
			if v, ok := typed["headers"]; ok {
				if m, ok := v.(map[string]Value); ok {
					headers = m
				}
			}
			if v, ok := typed["body"]; ok {
				body = fmt.Sprint(v)
			}
		default:
			body = fmt.Sprint(typed)
		}

		for key, value := range headers {
			w.Header().Set(key, fmt.Sprint(value))
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Fprintf(i.out, "serving on %s with %s\n", addr, handlerName)
	return http.ListenAndServe(addr, mux)
}

func (i *Interpreter) eval(expr string, scope *Scope) (Value, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", nil
	}

	if expr == "yes" || expr == "true" {
		return true, nil
	}
	if expr == "no" || expr == "false" {
		return false, nil
	}
	if expr == "nothing" || expr == "empty" || expr == "null" {
		return nil, nil
	}

	if strings.HasPrefix(expr, "[") && strings.HasSuffix(expr, "]") {
		inner := strings.TrimSpace(expr[1 : len(expr)-1])
		if inner == "" {
			return []Value{}, nil
		}
		items := []Value{}
		for _, part := range splitArgs(inner) {
			value, err := i.eval(part, scope)
			if err != nil {
				return nil, err
			}
			items = append(items, value)
		}
		return items, nil
	}

	if strings.Contains(expr, ".") {
		parts := strings.SplitN(expr, ".", 2)
		left, ok := scope.Get(strings.TrimSpace(parts[0]))
		if ok {
			switch typed := left.(type) {
			case map[string]Value:
				if value, exists := typed[strings.TrimSpace(parts[1])]; exists {
					return value, nil
				}
			case HSObject:
				if value, exists := typed.Fields[strings.TrimSpace(parts[1])]; exists {
					return value, nil
				}
			}
		}
	}

	if value, err := strconv.Atoi(expr); err == nil {
		return value, nil
	}
	if value, err := strconv.ParseFloat(expr, 64); err == nil {
		return value, nil
	}

	if strings.HasPrefix(strings.ToLower(expr), "do ") {
		return i.callTask(expr, scope)
	}

	if parts := splitBinary(expr, " + "); len(parts) == 2 {
		left, err := i.eval(parts[0], scope)
		if err != nil {
			return nil, err
		}
		right, err := i.eval(parts[1], scope)
		if err != nil {
			return nil, err
		}
		if isNumber(left) && isNumber(right) {
			if isFloat(left) || isFloat(right) {
				return toFloat(left) + toFloat(right), nil
			}
			return toInt(left) + toInt(right), nil
		}
		return fmt.Sprint(left) + fmt.Sprint(right), nil
	}

	if parts := splitBinary(expr, " - "); len(parts) == 2 {
		left, err := i.eval(parts[0], scope)
		if err != nil {
			return nil, err
		}
		right, err := i.eval(parts[1], scope)
		if err != nil {
			return nil, err
		}
		return toFloat(left) - toFloat(right), nil
	}

	if parts := splitBinary(expr, " * "); len(parts) == 2 {
		left, err := i.eval(parts[0], scope)
		if err != nil {
			return nil, err
		}
		right, err := i.eval(parts[1], scope)
		if err != nil {
			return nil, err
		}
		return toFloat(left) * toFloat(right), nil
	}

	if parts := splitBinary(expr, " / "); len(parts) == 2 {
		left, err := i.eval(parts[0], scope)
		if err != nil {
			return nil, err
		}
		right, err := i.eval(parts[1], scope)
		if err != nil {
			return nil, err
		}
		denom := toFloat(right)
		if denom == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return toFloat(left) / denom, nil
	}

	if strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"") && strings.Count(expr, "\"") == 2 {
		return expr[1 : len(expr)-1], nil
	}

	if strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'") && strings.Count(expr, "'") == 2 {
		return expr[1 : len(expr)-1], nil
	}

	if result, handled, err := i.tryBuiltinCall(expr, scope); handled {
		return result, err
	}

	if value, ok := scope.Get(expr); ok {
		return value, nil
	}

	return expr, nil
}

func (i *Interpreter) evalCondition(expr string, scope *Scope) (bool, error) {
	if parts := splitBinary(expr, " and "); len(parts) == 2 {
		left, err := i.evalCondition(parts[0], scope)
		if err != nil {
			return false, err
		}
		right, err := i.evalCondition(parts[1], scope)
		if err != nil {
			return false, err
		}
		return left && right, nil
	}
	if parts := splitBinary(expr, " or "); len(parts) == 2 {
		left, err := i.evalCondition(parts[0], scope)
		if err != nil {
			return false, err
		}
		right, err := i.evalCondition(parts[1], scope)
		if err != nil {
			return false, err
		}
		return left || right, nil
	}

	operators := []string{
		" is more than ",
		" is less than ",
		" is at least ",
		" is at most ",
		" is not ",
		" is ",
		" contains ",
	}

	for _, operator := range operators {
		if parts := splitBinary(expr, operator); len(parts) == 2 {
			left, err := i.eval(parts[0], scope)
			if err != nil {
				return false, err
			}
			right, err := i.eval(parts[1], scope)
			if err != nil {
				return false, err
			}
			switch operator {
			case " is more than ":
				return toFloat(left) > toFloat(right), nil
			case " is less than ":
				return toFloat(left) < toFloat(right), nil
			case " is at least ":
				return toFloat(left) >= toFloat(right), nil
			case " is at most ":
				return toFloat(left) <= toFloat(right), nil
			case " is not ":
				return fmt.Sprint(left) != fmt.Sprint(right), nil
			case " is ":
				return fmt.Sprint(left) == fmt.Sprint(right), nil
			case " contains ":
				return strings.Contains(strings.ToLower(fmt.Sprint(left)), strings.ToLower(fmt.Sprint(right))), nil
			}
		}
	}

	value, err := i.eval(expr, scope)
	if err != nil {
		return false, err
	}
	return truthy(value), nil
}

func collectUntil(lines []string, start, end int, endings []string) ([]string, int, error) {
	body := []string{}
	depth := 0
	for idx := start; idx < end; idx++ {
		current := strings.ToLower(strings.TrimSpace(lines[idx]))
		if StartsBlock(current) {
			depth++
		}
		if depth == 0 {
			for _, ending := range endings {
				if current == ending {
					return body, idx, nil
				}
			}
		}
		if EndsBlock(current) && depth > 0 {
			depth--
		}
		body = append(body, lines[idx])
	}
	return nil, 0, fmt.Errorf("block was not closed")
}

func collectUntilPrefix(lines []string, start, end int, prefix string) ([]string, int, error) {
	body := []string{}
	depth := 0
	for idx := start; idx < end; idx++ {
		current := strings.ToLower(strings.TrimSpace(lines[idx]))
		if StartsBlock(current) {
			depth++
		}
		if depth == 0 && strings.HasPrefix(current, prefix) {
			return body, idx, nil
		}
		if EndsBlock(current) && depth > 0 {
			depth--
		}
		body = append(body, lines[idx])
	}
	return nil, 0, fmt.Errorf("block was not closed")
}

func collectTrySections(lines []string, start, end int) ([]string, []string, []string, string, int, error) {
	tryBody := []string{}
	var catchBody []string
	var finallyBody []string
	catchName := ""
	depth := 0
	idx := start
	var nextIdx int
	var err error

	for ; idx < end; idx++ {
		current := strings.ToLower(strings.TrimSpace(lines[idx]))
		if StartsBlock(current) {
			depth++
		}
		if depth == 0 && (current == "catch" || strings.HasPrefix(current, "catch ") || current == "if it fails" || current == "finally" || current == "end") {
			break
		}
		if EndsBlock(current) && depth > 0 {
			depth--
		}
		tryBody = append(tryBody, lines[idx])
	}

	if idx >= end {
		return nil, nil, nil, "", 0, fmt.Errorf("try block was not closed")
	}

	marker := strings.ToLower(strings.TrimSpace(lines[idx]))
	if marker == "end" {
		return tryBody, nil, nil, "", idx, nil
	}

	if marker == "finally" {
		finallyBody, idx, _ = collectUntil(lines, idx+1, end, []string{"end"})
		return tryBody, nil, finallyBody, "", idx, nil
	}

	if marker == "if it fails" || strings.HasPrefix(marker, "catch") {
		if strings.HasPrefix(marker, "catch ") {
			catchLine := strings.TrimSpace(lines[idx][len("catch "):])
			if strings.Contains(strings.ToLower(catchLine), " as ") {
				parts := splitTwo(catchLine, " as ")
				catchName = strings.TrimSpace(parts[1])
			} else if catchLine != "" {
				catchName = strings.TrimSpace(catchLine)
			}
		}
		catchBody, nextIdx, err = collectUntil(lines, idx+1, end, []string{"finally", "end"})
		if err != nil {
			return nil, nil, nil, "", 0, err
		}
		idx = nextIdx
		if idx < end && strings.ToLower(strings.TrimSpace(lines[idx])) == "finally" {
			finallyBody, idx, err = collectUntil(lines, idx+1, end, []string{"end"})
			if err != nil {
				return nil, nil, nil, "", 0, err
			}
		}
		return tryBody, catchBody, finallyBody, catchName, idx, nil
	}

	return tryBody, nil, nil, "", idx, nil
}

func (i *Interpreter) runImportedFile(modulePath string) error {
	resolved := i.resolveModulePath(modulePath)
	if i.imported[resolved] {
		return nil
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return err
	}
	i.imported[resolved] = true
	previous := i.baseDir
	i.baseDir = filepath.Dir(resolved)
	defer func() {
		i.baseDir = previous
	}()
	lines := normalizeLines(string(data))
	_, err = i.execBlock(lines, 0, len(lines), i.root)
	return err
}

func (i *Interpreter) loadModule(modulePath string) (ModuleSnapshot, error) {
	resolved := i.resolveModulePath(modulePath)
	if snapshot, ok := i.moduleCache[resolved]; ok {
		return snapshot, nil
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return ModuleSnapshot{}, err
	}

	module := New(i.in, i.out)
	module.baseDir = filepath.Dir(resolved)
	module.imported = i.imported
	module.moduleCache = i.moduleCache
	lines := normalizeLines(string(data))
	if _, err := module.execBlock(lines, 0, len(lines), module.root); err != nil {
		return ModuleSnapshot{}, err
	}

	snapshot := ModuleSnapshot{
		Values:  map[string]Value{},
		Tasks:   map[string]Task{},
		Classes: map[string]ClassDef{},
	}
	for name := range module.exportedValues {
		if value, ok := module.root.Get(name); ok {
			snapshot.Values[name] = value
		}
	}
	for name := range module.exportedTasks {
		if task, ok := module.tasks[name]; ok {
			snapshot.Tasks[name] = task
		}
	}
	for name := range module.exportedClasses {
		if classDef, ok := module.classes[name]; ok {
			snapshot.Classes[name] = classDef
		}
	}
	i.moduleCache[resolved] = snapshot
	return snapshot, nil
}

func (i *Interpreter) parseClassBody(name string, body []string, scope *Scope) (ClassDef, error) {
	classDef := ClassDef{
		Name:    name,
		Fields:  map[string]Value{},
		Methods: map[string]Task{},
	}

	for idx := 0; idx < len(body); idx++ {
		line := strings.TrimSpace(body[idx])
		lower := strings.ToLower(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		if strings.HasPrefix(lower, "field ") {
			rest := strings.TrimSpace(line[len("field "):])
			if strings.Contains(strings.ToLower(rest), " as ") {
				parts := splitTwo(rest, " as ")
				value, err := i.eval(parts[1], scope)
				if err != nil {
					return ClassDef{}, err
				}
				classDef.Fields[strings.TrimSpace(parts[0])] = value
			}
			continue
		}

		if strings.HasPrefix(lower, "remember ") || strings.HasPrefix(lower, "let ") {
			rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "remember "), "let "))
			if strings.Contains(strings.ToLower(rest), " as ") {
				parts := splitTwo(rest, " as ")
				value, err := i.eval(parts[1], scope)
				if err != nil {
					return ClassDef{}, err
				}
				classDef.Fields[strings.TrimSpace(parts[0])] = value
			}
			continue
		}

		if strings.HasPrefix(lower, "method ") || strings.HasPrefix(lower, "task ") {
			header := line
			if strings.HasPrefix(lower, "task ") {
				header = line[len("task "):]
			} else {
				header = line[len("method "):]
			}
			methodName := strings.TrimSpace(header)
			params := []string{}
			if strings.Contains(strings.ToLower(header), " using ") {
				parts := splitTwo(header, " using ")
				methodName = strings.TrimSpace(parts[0])
				params = splitArgs(parts[1])
			}
			endings := []string{"end method", "end task"}
			methodBody, stop, err := collectUntil(body, idx+1, len(body), endings)
			if err != nil {
				return ClassDef{}, err
			}
			classDef.Methods[methodName] = Task{Params: params, Body: methodBody}
			idx = stop
		}
	}

	return classDef, nil
}

func parseObjectHeader(line string) (string, string, string, error) {
	lower := strings.ToLower(line)
	name := ""
	className := ""
	rest := ""
	if strings.HasPrefix(lower, "make an object called ") {
		rest = strings.TrimSpace(line[len("make an object called "):])
	} else if strings.HasPrefix(lower, "make object ") {
		rest = strings.TrimSpace(line[len("make object "):])
	} else {
		return "", "", "", fmt.Errorf("not an object declaration")
	}
	if strings.Contains(strings.ToLower(rest), " as ") {
		parts := splitTwo(rest, " as ")
		name = strings.TrimSpace(parts[0])
		rest = strings.TrimSpace(parts[1])
	} else {
		return "", "", "", fmt.Errorf("object declaration needs a class name")
	}
	if strings.Contains(strings.ToLower(rest), " with ") {
		parts := splitTwo(rest, " with ")
		className = strings.TrimSpace(parts[0])
		rest = strings.TrimSpace(parts[1])
	} else {
		className = strings.TrimSpace(rest)
		rest = ""
	}
	return name, className, rest, nil
}

func (i *Interpreter) createObject(className, dataExpr string, scope *Scope) (HSObject, error) {
	classDef, ok := i.classes[className]
	if !ok {
		return HSObject{}, fmt.Errorf("class not found: %s", className)
	}
	fields := map[string]Value{}
	for key, value := range classDef.Fields {
		fields[key] = value
	}
	if strings.TrimSpace(dataExpr) != "" {
		for _, entry := range splitArgs(dataExpr) {
			pair := splitTwo(entry, " as ")
			if pair[0] == "" || pair[1] == "" {
				continue
			}
			keyValue, err := i.eval(pair[0], scope)
			if err != nil {
				return HSObject{}, err
			}
			value, err := i.eval(pair[1], scope)
			if err != nil {
				return HSObject{}, err
			}
			fields[fmt.Sprint(keyValue)] = value
		}
	}
	return HSObject{ClassName: className, Fields: fields}, nil
}

func (i *Interpreter) resolveModulePath(modulePath string) string {
	resolved := modulePath
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(i.baseDir, modulePath)
	}
	return filepath.Clean(resolved)
}

func (i *Interpreter) evalModulePath(expr string, scope *Scope) (string, error) {
	rawPath := strings.Trim(strings.TrimSpace(expr), "\"'")
	if strings.HasSuffix(strings.ToLower(rawPath), ".hs") {
		return rawPath, nil
	}
	modulePathValue, err := i.eval(expr, scope)
	if err != nil {
		return "", err
	}
	modulePath := fmt.Sprint(modulePathValue)
	if !strings.HasSuffix(strings.ToLower(modulePath), ".hs") {
		return "", fmt.Errorf("import expects a .hs file path")
	}
	return modulePath, nil
}

func collectIfBlocks(lines []string, start, end int) ([]string, int, string, error) {
	body := []string{}
	depth := 0
	for idx := start; idx < end; idx++ {
		current := strings.ToLower(strings.TrimSpace(lines[idx]))
		if StartsBlock(current) {
			depth++
		}
		if depth == 0 && (current == "otherwise" || current == "else" || current == "end") {
			return body, idx, current, nil
		}
		if EndsBlock(current) && depth > 0 {
			depth--
		}
		body = append(body, lines[idx])
	}
	return nil, 0, "", fmt.Errorf("if block was not closed")
}

func splitTwo(text, delimiter string) [2]string {
	index := strings.Index(strings.ToLower(text), strings.ToLower(delimiter))
	return [2]string{
		text[:index],
		text[index+len(delimiter):],
	}
}

func splitArgs(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	parts := []string{}
	var current strings.Builder
	inString := false
	stringChar := byte(0)
	depth := 0

	for idx := 0; idx < len(text); idx++ {
		ch := text[idx]
		if inString {
			current.WriteByte(ch)
			if ch == stringChar {
				inString = false
			}
			continue
		}
		switch ch {
		case '"', '\'':
			inString = true
			stringChar = ch
			current.WriteByte(ch)
		case '[':
			depth++
			current.WriteByte(ch)
		case ']':
			depth--
			current.WriteByte(ch)
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteByte(ch)
			}
		default:
			current.WriteByte(ch)
		}
	}
	if strings.TrimSpace(current.String()) != "" {
		parts = append(parts, strings.TrimSpace(current.String()))
	}
	return parts
}

func splitBinary(text, delimiter string) []string {
	index := strings.LastIndex(strings.ToLower(text), strings.ToLower(delimiter))
	if index == -1 {
		return nil
	}
	return []string{
		strings.TrimSpace(text[:index]),
		strings.TrimSpace(text[index+len(delimiter):]),
	}
}

func matchAnyPrefix(text string, prefixes []string) (string, bool) {
	lower := strings.ToLower(text)
	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, strings.ToLower(prefix)) {
			return strings.TrimSpace(text[len(prefix):]), true
		}
	}
	return "", false
}

func display(value Value) string {
	switch typed := value.(type) {
	case nil:
		return "nothing"
	case bool:
		if typed {
			return "yes"
		}
		return "no"
	case []Value:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, display(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case HSObject:
		return fmt.Sprintf("<%s %v>", typed.ClassName, typed.Fields)
	case *Job:
		return "<job>"
	default:
		return fmt.Sprint(value)
	}
}

func sameValue(left, right Value) bool {
	return fmt.Sprint(left) == fmt.Sprint(right)
}

func toSlice(value Value) []Value {
	switch typed := value.(type) {
	case []Value:
		return typed
	case []string:
		items := make([]Value, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items
	default:
		return []Value{value}
	}
}

func truthy(value Value) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case int:
		return typed != 0
	case float64:
		return typed != 0
	case string:
		return typed != ""
	case []Value:
		return len(typed) > 0
	default:
		return true
	}
}

func isNumber(value Value) bool {
	switch value.(type) {
	case int, float64:
		return true
	default:
		return false
	}
}

func isFloat(value Value) bool {
	_, ok := value.(float64)
	return ok
}

func toInt(value Value) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
		return parsed
	case bool:
		if typed {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func toFloat(value Value) float64 {
	switch typed := value.(type) {
	case int:
		return float64(typed)
	case float64:
		return typed
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed
	case bool:
		if typed {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func (i *Interpreter) tryBuiltinCall(text string, scope *Scope) (Value, bool, error) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return nil, false, nil
	}

	moduleName := strings.ToLower(parts[0])
	bestMethod := ""
	bestWidth := 0
	maxWidth := len(parts) - 1
	if maxWidth > 3 {
		maxWidth = 3
	}
	for width := maxWidth; width >= 1; width-- {
		candidate := strings.ToLower(strings.Join(parts[1:1+width], " "))
		if isBuiltinMethod(moduleName, candidate) {
			bestMethod = candidate
			bestWidth = width
			break
		}
	}

	if bestMethod == "" {
		return nil, false, nil
	}

	rawMethod := strings.Join(parts[1:1+bestWidth], " ")
	argText := strings.TrimSpace(text[len(parts[0])+1+len(rawMethod):])
	args := []Value{}
	for _, part := range splitArgs(argText) {
		value, err := i.eval(part, scope)
		if err != nil {
			return nil, true, err
		}
		args = append(args, value)
	}

	result, err := executeBuiltin(moduleName, bestMethod, args)
	return result, true, err
}

func isBuiltinMethod(moduleName, methodName string) bool {
	switch moduleName {
	case "math":
		switch methodName {
		case "sqrt", "power", "random number", "max of", "min of":
			return true
		}
	case "text":
		switch methodName {
		case "uppercase", "lowercase", "contains text", "split by", "join with":
			return true
		}
	case "files":
		switch methodName {
		case "read", "write", "append", "exists", "list folder", "make folder", "join path":
			return true
		}
	case "time":
		switch methodName {
		case "now", "today", "timestamp":
			return true
		}
	case "data":
		switch methodName {
		case "json parse", "json text", "keys", "values":
			return true
		}
	case "system":
		switch methodName {
		case "get os", "run command":
			return true
		}
	case "web":
		switch methodName {
		case "get", "download":
			return true
		}
	}
	return false
}

func executeBuiltin(moduleName, methodName string, args []Value) (Value, error) {
	switch moduleName {
	case "math":
		switch methodName {
		case "sqrt":
			if len(args) < 1 {
				return nil, fmt.Errorf("math sqrt needs 1 value")
			}
			return sqrtNewton(toFloat(args[0])), nil
		case "power":
			if len(args) < 2 {
				return nil, fmt.Errorf("math power needs 2 values")
			}
			base := toFloat(args[0])
			exp := toInt(args[1])
			result := 1.0
			for range exp {
				result *= base
			}
			return result, nil
		case "random number":
			low := 0
			high := 100
			if len(args) >= 1 {
				low = toInt(args[0])
			}
			if len(args) >= 2 {
				high = toInt(args[1])
			}
			if high < low {
				low, high = high, low
			}
			return low + int(time.Now().UnixNano()%int64(high-low+1)), nil
		case "max of":
			if len(args) < 1 {
				return nil, fmt.Errorf("math max of needs values")
			}
			best := toFloat(args[0])
			for _, item := range args[1:] {
				v := toFloat(item)
				if v > best {
					best = v
				}
			}
			return best, nil
		case "min of":
			if len(args) < 1 {
				return nil, fmt.Errorf("math min of needs values")
			}
			best := toFloat(args[0])
			for _, item := range args[1:] {
				v := toFloat(item)
				if v < best {
					best = v
				}
			}
			return best, nil
		}
	case "text":
		switch methodName {
		case "uppercase":
			if len(args) < 1 {
				return nil, fmt.Errorf("text uppercase needs 1 value")
			}
			return strings.ToUpper(fmt.Sprint(args[0])), nil
		case "lowercase":
			if len(args) < 1 {
				return nil, fmt.Errorf("text lowercase needs 1 value")
			}
			return strings.ToLower(fmt.Sprint(args[0])), nil
		case "contains text":
			if len(args) < 2 {
				return nil, fmt.Errorf("text contains text needs 2 values")
			}
			return strings.Contains(strings.ToLower(fmt.Sprint(args[0])), strings.ToLower(fmt.Sprint(args[1]))), nil
		case "split by":
			if len(args) < 1 {
				return nil, fmt.Errorf("text split by needs text")
			}
			sep := " "
			if len(args) >= 2 {
				sep = fmt.Sprint(args[1])
			}
			items := strings.Split(fmt.Sprint(args[0]), sep)
			values := make([]Value, 0, len(items))
			for _, item := range items {
				values = append(values, item)
			}
			return values, nil
		case "join with":
			if len(args) < 1 {
				return nil, fmt.Errorf("text join with needs a list")
			}
			sep := " "
			if len(args) >= 2 {
				sep = fmt.Sprint(args[1])
			}
			items := toSlice(args[0])
			parts := make([]string, 0, len(items))
			for _, item := range items {
				parts = append(parts, fmt.Sprint(item))
			}
			return strings.Join(parts, sep), nil
		}
	case "files":
		switch methodName {
		case "read":
			if len(args) < 1 {
				return nil, fmt.Errorf("files read needs a path")
			}
			data, err := os.ReadFile(fmt.Sprint(args[0]))
			if err != nil {
				return nil, err
			}
			return string(data), nil
		case "write":
			if len(args) < 2 {
				return nil, fmt.Errorf("files write needs path and content")
			}
			return true, os.WriteFile(fmt.Sprint(args[0]), []byte(fmt.Sprint(args[1])), 0o644)
		case "append":
			if len(args) < 2 {
				return nil, fmt.Errorf("files append needs path and content")
			}
			file, err := os.OpenFile(fmt.Sprint(args[0]), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				return nil, err
			}
			defer file.Close()
			_, err = file.WriteString(fmt.Sprint(args[1]))
			return err == nil, err
		case "exists":
			if len(args) < 1 {
				return nil, fmt.Errorf("files exists needs a path")
			}
			_, err := os.Stat(fmt.Sprint(args[0]))
			return err == nil, nil
		case "list folder":
			path := "."
			if len(args) >= 1 {
				path = fmt.Sprint(args[0])
			}
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}
			values := make([]Value, 0, len(entries))
			for _, entry := range entries {
				values = append(values, entry.Name())
			}
			return values, nil
		case "make folder":
			if len(args) < 1 {
				return nil, fmt.Errorf("files make folder needs a path")
			}
			return true, os.MkdirAll(fmt.Sprint(args[0]), 0o755)
		case "join path":
			if len(args) < 1 {
				return nil, fmt.Errorf("files join path needs values")
			}
			parts := make([]string, 0, len(args))
			for _, arg := range args {
				parts = append(parts, fmt.Sprint(arg))
			}
			return filepath.Join(parts...), nil
		}
	case "time":
		switch methodName {
		case "now":
			return time.Now().Format("03:04 PM"), nil
		case "today":
			return time.Now().Format("January 02, 2006"), nil
		case "timestamp":
			return time.Now().Unix(), nil
		}
	case "data":
		switch methodName {
		case "json parse":
			if len(args) < 1 {
				return nil, fmt.Errorf("data json parse needs text")
			}
			var decoded any
			err := json.Unmarshal([]byte(fmt.Sprint(args[0])), &decoded)
			return normalizeJSON(decoded), err
		case "json text":
			if len(args) < 1 {
				return nil, fmt.Errorf("data json text needs value")
			}
			bytes, err := json.MarshalIndent(args[0], "", "  ")
			if err != nil {
				return nil, err
			}
			return string(bytes), nil
		case "keys":
			if len(args) < 1 {
				return nil, fmt.Errorf("data keys needs a map")
			}
			record, ok := args[0].(map[string]Value)
			if !ok {
				return nil, fmt.Errorf("data keys expects a map")
			}
			keys := make([]Value, 0, len(record))
			for key := range record {
				keys = append(keys, key)
			}
			return keys, nil
		case "values":
			if len(args) < 1 {
				return nil, fmt.Errorf("data values needs a map")
			}
			record, ok := args[0].(map[string]Value)
			if !ok {
				return nil, fmt.Errorf("data values expects a map")
			}
			values := make([]Value, 0, len(record))
			for _, value := range record {
				values = append(values, value)
			}
			return values, nil
		}
	case "system":
		switch methodName {
		case "get os":
			return runtimeOS(), nil
		case "run command":
			if len(args) < 1 {
				return nil, fmt.Errorf("system run command needs a command")
			}
			cmd := exec.Command("cmd", "/C", fmt.Sprint(args[0]))
			out, err := cmd.CombinedOutput()
			return strings.TrimRight(string(out), "\r\n"), err
		}
	case "web":
		switch methodName {
		case "get":
			if len(args) < 1 {
				return nil, fmt.Errorf("web get needs a url")
			}
			resp, err := http.Get(fmt.Sprint(args[0]))
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			bytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			return string(bytes), nil
		case "download":
			if len(args) < 2 {
				return nil, fmt.Errorf("web download needs url and path")
			}
			resp, err := http.Get(fmt.Sprint(args[0]))
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			bytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			return true, os.WriteFile(fmt.Sprint(args[1]), bytes, 0o644)
		}
	}
	return nil, fmt.Errorf("unknown builtin call")
}

func normalizeJSON(value any) Value {
	switch typed := value.(type) {
	case map[string]any:
		out := map[string]Value{}
		for key, item := range typed {
			out[key] = normalizeJSON(item)
		}
		return out
	case []any:
		out := make([]Value, 0, len(typed))
		for _, item := range typed {
			out = append(out, normalizeJSON(item))
		}
		return out
	default:
		return typed
	}
}

func runtimeOS() string {
	cmd := exec.Command("cmd", "/C", "ver")
	out, err := cmd.CombinedOutput()
	if err == nil && len(out) > 0 {
		return strings.TrimSpace(string(out))
	}
	return "Windows"
}

func sqrtNewton(value float64) float64 {
	if value <= 0 {
		return 0
	}
	guess := value
	for range 12 {
		guess = 0.5 * (guess + value/guess)
	}
	return guess
}
