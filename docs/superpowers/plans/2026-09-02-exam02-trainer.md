# Exam Rank 02 Trainer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A single-binary terminal application for macOS and Linux that presents 42 Exam Rank 02 subjects, gives the user a workspace, and grades their C automatically — including the allowed-functions check the real Moulinette enforces.

**Architecture:** The upstream reference solutions are the grading oracle. For each exercise the user's `.c` and the reference `.c` are compiled with identical flags, linked against the same hand-written test driver, and run on the same inputs; any difference in stdout, exit status or signal is a failure. This means no expected outputs are written by hand. Before anything runs, `nm -u` on the user's object file catches calls to functions the subject does not allow. A Bubble Tea TUI drives two modes over that engine: a timed four-exercise exam and untimed free practice.

**Tech Stack:** Go 1.24+, Bubble Tea v1.3.10, Lipgloss v1.1.0, gopkg.in/yaml.v3 v3.0.1. System `cc` (clang on macOS, gcc or clang on Linux) and `nm` are runtime requirements, not build requirements.

**Spec:** `docs/superpowers/specs/2026-09-02-exam02-trainer-design.md`

## Global Constraints

These apply to every task; each task's requirements implicitly include them.

- Module path is `exam02`. Go directive `go 1.24`.
- Dependencies are exactly: `github.com/charmbracelet/bubbletea v1.3.10`, `github.com/charmbracelet/lipgloss v1.1.0`, `gopkg.in/yaml.v3 v3.0.1`. Adding any other dependency needs an explicit reason recorded in the commit message.
- Build targets: `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64`. Windows is not supported; using `syscall.SysProcAttr` fields that exist only on Unix is fine.
- Every compile of user or reference code uses exactly `cc -Wall -Wextra -Werror`. A warning is a failure, as in the real exam.
- Exam rules: 3 hours total, 25 points per exercise passed, 100 points to pass. An exam is exactly four exercises, one drawn at random from each level's pool.
- Exercise pools: Level 1 has 12 exercises, Level 2 has 19, Level 3 has 15, Level 4 has 10. Total 56.
- The upstream repository (github.com/alexhiguera/Exam_Rank_02_42_School, MIT, Copyright (c) 2026 Alex Higuera) must be credited in the application README and in the in-app about screen. Its `LICENSE` stays at `third_party/exam_rank_02/LICENSE`.
- A file already present in `rendu/` is never overwritten. The user's work is sacred.
- TDD throughout: write the failing test, run it and watch it fail for the expected reason, write the minimal implementation, run it and watch it pass, commit.
- Test files live beside the code they test, Go style: `internal/catalog/subject_test.go` tests `internal/catalog/subject.go`.

## Scope

This plan delivers a complete, shippable application whose exercise content is Level 1 only (12 of 56). Every engine, mode, and release mechanism is finished and tested; only the driver and case files for Levels 2–4 are outstanding. That is deliberate — Task 7 proves the entire pipeline against 12 real exercises before the remaining 44 drivers are written, and a driver bug found at exercise 3 is far cheaper than the same bug found at exercise 50.

Levels 2–4 content is a follow-on plan. It repeats Task 7's pattern per exercise and needs no engine changes.

---

### Task 1: Module skeleton and subject parser

The parser reads the machine-readable header of an upstream subject. Getting it right first means every later task can rely on structured exercise metadata.

**Files:**
- Create: `go.mod`, `.gitignore` (already exists, leave it), `internal/catalog/subject.go`
- Test: `internal/catalog/subject_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `catalog.SubjectMeta{Name, ExpectedFile string; AllowedFunctions []string}` and `catalog.ParseSubject(md string) (SubjectMeta, error)`.

- [ ] **Step 1: Initialise the module**

```bash
cd /home/aselezen/exam02
go mod init exam02
go mod edit -go=1.24
```

- [ ] **Step 2: Write the failing test**

Create `internal/catalog/subject_test.go`. The five cases below are the five spellings that actually occur upstream — an ordinary list, `None`, `-`, an empty value, and a multi-file `Expected files`.

```go
package catalog

import (
	"reflect"
	"testing"
)

const strlenSubject = "## Subject\n\n```BASH\n" +
	"Assignment name  : ft_strlen\n" +
	"Expected files   : ft_strlen.c\n" +
	"Allowed functions: \n" +
	"--------------------------------------------------------------------------------\n\n" +
	"Write a function that returns the length of a string.\n\n" +
	"Your function must be declared as follows:\n\n" +
	"int\tft_strlen(char *str);\n```\n"

func TestParseSubjectReadsHeaderFields(t *testing.T) {
	got, err := ParseSubject(strlenSubject)
	if err != nil {
		t.Fatalf("ParseSubject() error = %v", err)
	}
	if got.Name != "ft_strlen" {
		t.Errorf("Name = %q, want %q", got.Name, "ft_strlen")
	}
	if got.ExpectedFile != "ft_strlen.c" {
		t.Errorf("ExpectedFile = %q, want %q", got.ExpectedFile, "ft_strlen.c")
	}
	if len(got.AllowedFunctions) != 0 {
		t.Errorf("AllowedFunctions = %v, want empty", got.AllowedFunctions)
	}
}

func TestParseSubjectAllowedFunctions(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  []string
	}{
		{"empty means none", "", nil},
		{"literal None", "None", nil},
		{"literal dash", "-", nil},
		{"single", "malloc", []string{"malloc"}},
		{"list", "write, malloc, free", []string{"write", "malloc", "free"}},
		{"ragged spacing", "  write ,malloc  ", []string{"write", "malloc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := "Assignment name  : x\nExpected files   : x.c\nAllowed functions: " + tt.field + "\n"
			got, err := ParseSubject(md)
			if err != nil {
				t.Fatalf("ParseSubject() error = %v", err)
			}
			if !reflect.DeepEqual(got.AllowedFunctions, tt.want) {
				t.Errorf("AllowedFunctions = %#v, want %#v", got.AllowedFunctions, tt.want)
			}
		})
	}
}

func TestParseSubjectMultipleExpectedFilesTakesTheC(t *testing.T) {
	md := "Assignment name  : ft_list_size\nExpected files   : ft_list_size.c, ft_list.h\nAllowed functions: \n"
	got, err := ParseSubject(md)
	if err != nil {
		t.Fatalf("ParseSubject() error = %v", err)
	}
	if got.ExpectedFile != "ft_list_size.c" {
		t.Errorf("ExpectedFile = %q, want %q", got.ExpectedFile, "ft_list_size.c")
	}
}

func TestParseSubjectGlobExpectedFilesIsUnresolved(t *testing.T) {
	// flood_fill and sort_list write "*.c, *.h". A glob cannot be resolved
	// automatically, so the parser reports no expected file and meta.yaml
	// supplies one by hand.
	md := "Assignment name  : flood_fill\nExpected files   : *.c, *.h\nAllowed functions: \n"
	got, err := ParseSubject(md)
	if err != nil {
		t.Fatalf("ParseSubject() error = %v", err)
	}
	if got.ExpectedFile != "" {
		t.Errorf("ExpectedFile = %q, want empty for a glob", got.ExpectedFile)
	}
}

func TestParseSubjectWithoutAssignmentNameIsAnError(t *testing.T) {
	if _, err := ParseSubject("just prose, no header\n"); err == nil {
		t.Fatal("ParseSubject() error = nil, want an error")
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/catalog/ -run TestParseSubject -v`
Expected: FAIL — `undefined: ParseSubject`.

- [ ] **Step 4: Write the implementation**

Create `internal/catalog/subject.go`:

```go
// Package catalog holds the exercise data: the upstream subjects, the
// reference solutions used as a grading oracle, and the drivers and input
// cases written for this project.
package catalog

import (
	"bufio"
	"fmt"
	"strings"
)

// SubjectMeta is the machine-readable header of an upstream exercise subject.
type SubjectMeta struct {
	Name             string
	ExpectedFile     string
	AllowedFunctions []string
}

// ParseSubject extracts the header fields from an upstream subject document.
//
// ExpectedFile may come back empty: four subjects write a glob such as
// "*.c, *.h" rather than naming a file, and a glob cannot be resolved without
// knowing the exercise. Those four carry a hand-written expected_file in their
// meta.yaml instead.
func ParseSubject(md string) (SubjectMeta, error) {
	var m SubjectMeta
	scanner := bufio.NewScanner(strings.NewReader(md))
	for scanner.Scan() {
		key, value, ok := splitField(scanner.Text())
		if !ok {
			continue
		}
		switch key {
		case "Assignment name":
			m.Name = value
		case "Expected files":
			m.ExpectedFile = sourceFile(value)
		case "Allowed functions":
			m.AllowedFunctions = parseAllowedFunctions(value)
		}
	}
	if err := scanner.Err(); err != nil {
		return m, fmt.Errorf("reading subject: %w", err)
	}
	if m.Name == "" {
		return m, fmt.Errorf("subject has no assignment name")
	}
	return m, nil
}

// splitField splits "Assignment name  : ft_strlen" into its key and value.
// Subject prose also contains colons, but only the three known keys are acted
// on, so a false split is harmless.
func splitField(line string) (key, value string, ok bool) {
	key, value, ok = strings.Cut(line, ":")
	if !ok {
		return "", "", false
	}
	return strings.TrimSpace(key), strings.TrimSpace(value), true
}

// sourceFile picks the .c file the candidate is expected to write out of an
// "Expected files" value. The value may name one file, name a .c alongside a
// header that is provided rather than written, or be a glob.
func sourceFile(value string) string {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "*") {
			continue
		}
		if strings.HasSuffix(part, ".c") {
			return part
		}
	}
	return ""
}

// parseAllowedFunctions reads the allowed-functions field. Upstream writes an
// empty value, "None", or "-" to mean that no library function may be used.
func parseAllowedFunctions(value string) []string {
	if value == "" || value == "-" || strings.EqualFold(value, "none") {
		return nil
	}
	var out []string
	for _, part := range strings.Split(value, ",") {
		if fn := strings.TrimSpace(part); fn != "" {
			out = append(out, fn)
		}
	}
	return out
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/catalog/ -v`
Expected: PASS, all subtests.

- [ ] **Step 6: Commit**

```bash
git add go.mod internal/catalog/
git commit -m "feat(catalog): parse upstream exercise subjects

Reads assignment name, expected file and allowed functions out of the
subject header. Handles the five spellings that occur upstream, including
the four subjects whose expected files are a glob and so cannot be
resolved automatically.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 2: Exercise types, case files, and the embedded loader

**Files:**
- Create: `internal/catalog/exercise.go`, `internal/catalog/cases.go`, `internal/catalog/load.go`
- Test: `internal/catalog/cases_test.go`, `internal/catalog/load_test.go`

**Interfaces:**
- Consumes: `catalog.SubjectMeta` from Task 1.
- Produces: `catalog.Kind`, `catalog.Exercise`, `catalog.Case`, `catalog.ParseCases(string) ([]Case, error)`, `catalog.Load(fs.FS) (*Catalog, error)`, `(*Catalog).All() []Exercise`, `(*Catalog).ByLevel(int) []Exercise`, `(*Catalog).ByName(string) (Exercise, bool)`.

- [ ] **Step 1: Write the failing case-parser test**

Create `internal/catalog/cases_test.go`:

```go
package catalog

import (
	"reflect"
	"testing"
)

func TestParseCases(t *testing.T) {
	in := `{"args": ["hello world"]}
{"args": [""]}
{"args": []}

{"args": ["a", "b"], "stdin": "x\n"}
`
	got, err := ParseCases(in)
	if err != nil {
		t.Fatalf("ParseCases() error = %v", err)
	}
	want := []Case{
		{Args: []string{"hello world"}},
		{Args: []string{""}},
		{Args: []string{}},
		{Args: []string{"a", "b"}, Stdin: "x\n"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseCases() = %#v, want %#v", got, want)
	}
}

func TestParseCasesDistinguishesEmptyArgFromNoArg(t *testing.T) {
	// Several exercises print different things for ./prog "" and ./prog.
	// This distinction is the whole reason the format is JSON.
	got, err := ParseCases("{\"args\": [\"\"]}\n{\"args\": []}\n")
	if err != nil {
		t.Fatalf("ParseCases() error = %v", err)
	}
	if len(got[0].Args) != 1 || got[0].Args[0] != "" {
		t.Errorf("first case Args = %#v, want one empty string", got[0].Args)
	}
	if len(got[1].Args) != 0 {
		t.Errorf("second case Args = %#v, want none", got[1].Args)
	}
}

func TestParseCasesRejectsMalformedLine(t *testing.T) {
	if _, err := ParseCases("not json\n"); err == nil {
		t.Fatal("ParseCases() error = nil, want an error")
	}
}

func TestParseCasesRejectsEmptyFile(t *testing.T) {
	// An exercise with no cases would silently pass everything.
	if _, err := ParseCases("\n\n"); err == nil {
		t.Fatal("ParseCases() error = nil, want an error for a caseless file")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/catalog/ -run TestParseCases -v`
Expected: FAIL — `undefined: ParseCases`, `undefined: Case`.

- [ ] **Step 3: Write the exercise types**

Create `internal/catalog/exercise.go`:

```go
package catalog

// Kind says how an exercise is built and run.
type Kind string

const (
	// KindFunction means the candidate writes a function with no main. The
	// project supplies a driver that calls it and prints the result.
	KindFunction Kind = "function"
	// KindProgram means the candidate writes a whole program, main included,
	// which is run directly on the case arguments.
	KindProgram Kind = "program"
)

// Case is one set of inputs an exercise is run against.
type Case struct {
	Args  []string `json:"args"`
	Stdin string   `json:"stdin,omitempty"`
}

// Meta is the hand-reviewed metadata stored as meta.yaml beside each exercise.
type Meta struct {
	Name             string   `yaml:"name"`
	Level            int      `yaml:"level"`
	Kind             Kind     `yaml:"kind"`
	ExpectedFile     string   `yaml:"expected_file"`
	AllowedFunctions []string `yaml:"allowed_functions"`
	Prototype        string   `yaml:"prototype"`
	Header           string   `yaml:"header,omitempty"`
	TimeoutMS        int      `yaml:"timeout_ms,omitempty"`
}

// Exercise is everything needed to present and grade one exercise.
type Exercise struct {
	Meta
	Subject       string // the upstream subject, shown to the candidate
	Spanish       string // the upstream Spanish explainer, practice mode only
	Reference     string // the oracle solution
	Driver        string // test main; empty when Kind is KindProgram
	HeaderContent string // contents of Header; empty when Header is empty
	Cases         []Case
}

// Timeout is how long one run of this exercise may take, in milliseconds.
func (e Exercise) Timeout() int {
	if e.TimeoutMS > 0 {
		return e.TimeoutMS
	}
	return 5000
}
```

- [ ] **Step 4: Write the case parser**

Create `internal/catalog/cases.go`:

```go
package catalog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
)

// ParseCases reads a cases.txt file: one JSON object per line, blank lines
// ignored. JSON rather than shell-like quoting because it needs no custom
// parser and is unambiguous about an empty-string argument versus an absent
// one, a distinction several exercises depend on.
func ParseCases(text string) ([]Case, error) {
	var cases []Case
	scanner := bufio.NewScanner(strings.NewReader(text))
	for line := 1; scanner.Scan(); line++ {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var c Case
		if err := json.Unmarshal([]byte(raw), &c); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		if c.Args == nil {
			c.Args = []string{}
		}
		cases = append(cases, c)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading cases: %w", err)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("no cases: an exercise with no cases would pass anything")
	}
	return cases, nil
}
```

- [ ] **Step 5: Run the case tests to verify they pass**

Run: `go test ./internal/catalog/ -run TestParseCases -v`
Expected: PASS.

- [ ] **Step 6: Write the failing loader test**

Create `internal/catalog/load_test.go`:

```go
package catalog

import (
	"testing"
	"testing/fstest"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"exercises/ft_strlen/meta.yaml": {Data: []byte(
			"name: ft_strlen\nlevel: 1\nkind: function\nexpected_file: ft_strlen.c\n" +
				"allowed_functions: []\nprototype: \"int ft_strlen(char *str);\"\n")},
		"exercises/ft_strlen/subject.md":  {Data: []byte("subject text")},
		"exercises/ft_strlen/spanish.md":  {Data: []byte("texto")},
		"exercises/ft_strlen/reference.c": {Data: []byte("int ft_strlen(char *s){return 0;}")},
		"exercises/ft_strlen/driver.c":    {Data: []byte("int main(void){return 0;}")},
		"exercises/ft_strlen/cases.txt":   {Data: []byte("{\"args\": [\"abc\"]}\n")},

		"exercises/rot_13/meta.yaml": {Data: []byte(
			"name: rot_13\nlevel: 1\nkind: program\nexpected_file: rot_13.c\n" +
				"allowed_functions: [write]\nprototype: \"\"\n")},
		"exercises/rot_13/subject.md":  {Data: []byte("subject text")},
		"exercises/rot_13/spanish.md":  {Data: []byte("texto")},
		"exercises/rot_13/reference.c": {Data: []byte("int main(void){return 0;}")},
		"exercises/rot_13/cases.txt":   {Data: []byte("{\"args\": [\"abc\"]}\n")},
	}
}

func TestLoadReadsExercises(t *testing.T) {
	c, err := Load(testFS())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if n := len(c.All()); n != 2 {
		t.Fatalf("All() returned %d exercises, want 2", n)
	}
	ex, ok := c.ByName("ft_strlen")
	if !ok {
		t.Fatal("ByName(ft_strlen) not found")
	}
	if ex.Kind != KindFunction {
		t.Errorf("Kind = %q, want %q", ex.Kind, KindFunction)
	}
	if ex.Driver == "" {
		t.Error("Driver is empty, want the driver source")
	}
	if ex.Subject != "subject text" {
		t.Errorf("Subject = %q, want %q", ex.Subject, "subject text")
	}
	if len(ex.Cases) != 1 {
		t.Errorf("Cases = %d, want 1", len(ex.Cases))
	}
	if ex.Timeout() != 5000 {
		t.Errorf("Timeout() = %d, want the 5000 default", ex.Timeout())
	}
}

func TestLoadProgramKindNeedsNoDriver(t *testing.T) {
	c, err := Load(testFS())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	ex, _ := c.ByName("rot_13")
	if ex.Driver != "" {
		t.Errorf("Driver = %q, want empty for a program exercise", ex.Driver)
	}
}

func TestLoadRejectsFunctionExerciseWithoutDriver(t *testing.T) {
	fs := testFS()
	delete(fs, "exercises/ft_strlen/driver.c")
	if _, err := Load(fs); err == nil {
		t.Fatal("Load() error = nil, want an error for a function exercise with no driver")
	}
}

func TestByLevelIsSortedByName(t *testing.T) {
	c, err := Load(testFS())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	got := c.ByLevel(1)
	if len(got) != 2 {
		t.Fatalf("ByLevel(1) returned %d, want 2", len(got))
	}
	if got[0].Name != "ft_strlen" || got[1].Name != "rot_13" {
		t.Errorf("ByLevel(1) = [%s %s], want sorted [ft_strlen rot_13]", got[0].Name, got[1].Name)
	}
	if n := len(c.ByLevel(4)); n != 0 {
		t.Errorf("ByLevel(4) returned %d, want 0", n)
	}
}
```

- [ ] **Step 7: Run it to verify it fails**

Run: `go test ./internal/catalog/ -run 'TestLoad|TestByLevel' -v`
Expected: FAIL — `undefined: Load`.

- [ ] **Step 8: Write the loader**

Create `internal/catalog/load.go`:

```go
package catalog

import (
	"fmt"
	"io/fs"
	"sort"

	"gopkg.in/yaml.v3"
)

// Catalog is the set of exercises available to the application.
type Catalog struct {
	byName map[string]Exercise
	names  []string // sorted
}

// Load reads every exercise under the "exercises" directory of fsys. It is
// given an fs.FS rather than a path so the production catalog can come from
// go:embed while tests use an in-memory filesystem.
func Load(fsys fs.FS) (*Catalog, error) {
	entries, err := fs.ReadDir(fsys, "exercises")
	if err != nil {
		return nil, fmt.Errorf("reading exercises directory: %w", err)
	}
	c := &Catalog{byName: make(map[string]Exercise, len(entries))}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		ex, err := loadExercise(fsys, "exercises/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("exercise %s: %w", entry.Name(), err)
		}
		c.byName[ex.Name] = ex
		c.names = append(c.names, ex.Name)
	}
	if len(c.names) == 0 {
		return nil, fmt.Errorf("no exercises found")
	}
	sort.Strings(c.names)
	return c, nil
}

func loadExercise(fsys fs.FS, dir string) (Exercise, error) {
	var ex Exercise

	metaBytes, err := fs.ReadFile(fsys, dir+"/meta.yaml")
	if err != nil {
		return ex, fmt.Errorf("reading meta.yaml: %w", err)
	}
	if err := yaml.Unmarshal(metaBytes, &ex.Meta); err != nil {
		return ex, fmt.Errorf("parsing meta.yaml: %w", err)
	}
	if ex.Name == "" || ex.Level == 0 || ex.ExpectedFile == "" {
		return ex, fmt.Errorf("meta.yaml is missing name, level or expected_file")
	}
	if ex.Kind != KindFunction && ex.Kind != KindProgram {
		return ex, fmt.Errorf("meta.yaml has kind %q, want %q or %q", ex.Kind, KindFunction, KindProgram)
	}

	required := map[string]*string{
		"subject.md":  &ex.Subject,
		"reference.c": &ex.Reference,
	}
	for file, dest := range required {
		b, err := fs.ReadFile(fsys, dir+"/"+file)
		if err != nil {
			return ex, fmt.Errorf("reading %s: %w", file, err)
		}
		*dest = string(b)
	}

	// The Spanish explainer is a nicety; an exercise without one still works.
	if b, err := fs.ReadFile(fsys, dir+"/spanish.md"); err == nil {
		ex.Spanish = string(b)
	}

	if ex.Kind == KindFunction {
		b, err := fs.ReadFile(fsys, dir+"/driver.c")
		if err != nil {
			return ex, fmt.Errorf("a function exercise needs driver.c: %w", err)
		}
		ex.Driver = string(b)
	}

	if ex.Header != "" {
		b, err := fs.ReadFile(fsys, dir+"/"+ex.Header)
		if err != nil {
			return ex, fmt.Errorf("reading declared header %s: %w", ex.Header, err)
		}
		ex.HeaderContent = string(b)
	}

	caseBytes, err := fs.ReadFile(fsys, dir+"/cases.txt")
	if err != nil {
		return ex, fmt.Errorf("reading cases.txt: %w", err)
	}
	if ex.Cases, err = ParseCases(string(caseBytes)); err != nil {
		return ex, fmt.Errorf("parsing cases.txt: %w", err)
	}
	return ex, nil
}

// All returns every exercise, ordered by name.
func (c *Catalog) All() []Exercise {
	out := make([]Exercise, 0, len(c.names))
	for _, n := range c.names {
		out = append(out, c.byName[n])
	}
	return out
}

// ByLevel returns the exercises of one level, ordered by name. This is the
// draw pool an exam picks a single exercise from.
func (c *Catalog) ByLevel(level int) []Exercise {
	var out []Exercise
	for _, n := range c.names {
		if ex := c.byName[n]; ex.Level == level {
			out = append(out, ex)
		}
	}
	return out
}

// ByName looks up one exercise.
func (c *Catalog) ByName(name string) (Exercise, bool) {
	ex, ok := c.byName[name]
	return ex, ok
}
```

- [ ] **Step 9: Add the yaml dependency and run the tests**

```bash
go get gopkg.in/yaml.v3@v3.0.1
go test ./internal/catalog/ -v
```
Expected: PASS, every test.

- [ ] **Step 10: Commit**

```bash
git add go.mod go.sum internal/catalog/
git commit -m "feat(catalog): exercise types and embedded loader

Load takes an fs.FS so the production catalog can come from go:embed
while tests use an in-memory filesystem. A function exercise without a
driver and a cases.txt with no cases are both load errors, since either
would let an exercise silently pass anything.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 3: Importer tool

Converts the vendored upstream repository into `data/exercises/`. Run once, then its output is edited by hand and committed. It is a development tool, not part of the shipped binary.

**Files:**
- Create: `tools/import/main.go`
- Test: none. This is a one-shot generator whose output is reviewed by hand and committed; the integration test in Task 7 is what actually validates the generated data.

**Interfaces:**
- Consumes: `catalog.ParseSubject` from Task 1.
- Produces: `data/exercises/<name>/{subject.md,spanish.md,reference.c,meta.yaml}` for all 56 exercises, plus any upstream header file.

- [ ] **Step 1: Write the importer**

Create `tools/import/main.go`:

```go
// Command import converts the vendored upstream exercise repository into the
// data/exercises tree the application embeds.
//
// It is run by hand, and its output is committed. Two fields it writes are
// guesses that a human must confirm: kind, because a phrase heuristic
// classifies only 43 of 56 subjects, and expected_file for the four subjects
// whose expected files are a glob.
//
// Usage: go run ./tools/import -src third_party/exam_rank_02 -dst data/exercises
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"exam02/internal/catalog"
	"gopkg.in/yaml.v3"
)

func main() {
	src := flag.String("src", "third_party/exam_rank_02", "vendored upstream repository")
	dst := flag.String("dst", "data/exercises", "output directory")
	flag.Parse()

	if err := run(*src, *dst); err != nil {
		log.Fatal(err)
	}
}

func run(src, dst string) error {
	var imported, needsReview int
	for level := 1; level <= 4; level++ {
		levelDir := filepath.Join(src, "Level_"+strconv.Itoa(level))
		entries, err := os.ReadDir(levelDir)
		if err != nil {
			return fmt.Errorf("reading %s: %w", levelDir, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			review, err := importOne(filepath.Join(levelDir, entry.Name()), dst, level, entry.Name())
			if err != nil {
				return fmt.Errorf("%s: %w", entry.Name(), err)
			}
			imported++
			if review != "" {
				needsReview++
				fmt.Printf("REVIEW %-20s %s\n", entry.Name(), review)
			}
		}
	}
	fmt.Printf("\nimported %d exercises, %d need review\n", imported, needsReview)
	return nil
}

// importOne writes one exercise directory. It returns a non-empty string
// describing what a human still has to check, or "" when nothing does.
func importOne(srcDir, dstRoot string, level int, name string) (string, error) {
	subject, err := os.ReadFile(filepath.Join(srcDir, "README.md"))
	if err != nil {
		return "", fmt.Errorf("reading README.md: %w", err)
	}
	meta, err := catalog.ParseSubject(string(subject))
	if err != nil {
		return "", err
	}

	outDir := filepath.Join(dstRoot, name)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(outDir, "subject.md"), subject, 0o644); err != nil {
		return "", err
	}

	if spanish, err := os.ReadFile(filepath.Join(srcDir, "spanish.md")); err == nil {
		if err := os.WriteFile(filepath.Join(outDir, "spanish.md"), spanish, 0o644); err != nil {
			return "", err
		}
	}

	// The reference solution is whichever .c file upstream ships. Any commented
	// out demo main it contains is left as is: it is inside a comment, so it
	// neither compiles nor interferes with linking against a driver.
	refName, err := singleFileWithSuffix(srcDir, ".c")
	if err != nil {
		return "", err
	}
	reference, err := os.ReadFile(filepath.Join(srcDir, refName))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(outDir, "reference.c"), reference, 0o644); err != nil {
		return "", err
	}

	header, _ := singleFileWithSuffix(srcDir, ".h")
	if header != "" {
		b, err := os.ReadFile(filepath.Join(srcDir, header))
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(outDir, header), b, 0o644); err != nil {
			return "", err
		}
	}

	kind, kindCertain := guessKind(string(subject))
	expected := meta.ExpectedFile
	if expected == "" {
		expected = name + ".c" // a guess for the four glob subjects
	}

	out := catalog.Meta{
		Name:             name,
		Level:            level,
		Kind:             kind,
		ExpectedFile:     expected,
		AllowedFunctions: meta.AllowedFunctions,
		Prototype:        extractPrototype(string(subject)),
		Header:           header,
	}
	if out.AllowedFunctions == nil {
		out.AllowedFunctions = []string{}
	}
	encoded, err := yaml.Marshal(out)
	if err != nil {
		return "", err
	}
	banner := "# Generated by tools/import, then reviewed by hand.\n" +
		"# kind and expected_file are guesses until a human confirms them.\n"
	if err := os.WriteFile(filepath.Join(outDir, "meta.yaml"), append([]byte(banner), encoded...), 0o644); err != nil {
		return "", err
	}

	switch {
	case !kindCertain && meta.ExpectedFile == "":
		return "kind and expected_file both guessed", nil
	case !kindCertain:
		return "kind guessed as " + string(kind), nil
	case meta.ExpectedFile == "":
		return "expected_file guessed as " + expected, nil
	}
	return "", nil
}

// guessKind classifies a subject by the phrase it uses. The phrases cover 43
// of 56 subjects; the rest come back as a low-confidence guess for review.
func guessKind(subject string) (kind catalog.Kind, certain bool) {
	switch {
	case strings.Contains(subject, "function must be declared"):
		return catalog.KindFunction, true
	case strings.Contains(subject, "Write a program"):
		return catalog.KindProgram, true
	case strings.Contains(subject, "must be declared as follows"):
		return catalog.KindFunction, false
	default:
		return catalog.KindProgram, false
	}
}

// extractPrototype pulls the C declaration out of a subject, used to seed the
// candidate's workspace file. An empty result is fine for program exercises.
func extractPrototype(subject string) string {
	lines := strings.Split(subject, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "declared as follows") {
			continue
		}
		for _, candidate := range lines[i+1:] {
			candidate = strings.TrimSpace(candidate)
			if candidate == "" {
				continue
			}
			if strings.HasSuffix(candidate, ";") {
				return strings.Join(strings.Fields(candidate), " ")
			}
			return ""
		}
	}
	return ""
}

// singleFileWithSuffix returns the one file in dir with the given suffix, or
// "" if there is none. More than one is an error, because the importer would
// otherwise silently pick an arbitrary file as the reference solution.
func singleFileWithSuffix(dir, suffix string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var found []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), suffix) {
			found = append(found, e.Name())
		}
	}
	switch len(found) {
	case 0:
		return "", nil
	case 1:
		return found[0], nil
	default:
		return "", fmt.Errorf("expected one %s file in %s, found %v", suffix, dir, found)
	}
}
```

- [ ] **Step 2: Run the importer**

```bash
go run ./tools/import -src third_party/exam_rank_02 -dst data/exercises
```
Expected: `imported 56 exercises, N need review`, with a `REVIEW` line per uncertain exercise.

- [ ] **Step 3: Verify the shape of the output**

```bash
ls data/exercises | wc -l          # expect 56
cat data/exercises/ft_strlen/meta.yaml
cat data/exercises/flood_fill/meta.yaml
```
Expected: 56 directories; `ft_strlen` has `kind: function`; `flood_fill` has a guessed `expected_file: flood_fill.c` and a `header:` naming its upstream header.

- [ ] **Step 4: Review the Level 1 metadata by hand**

Set `kind` correctly for all 12 Level 1 exercises. The correct values, read from the subjects:

| Exercise | kind | expected_file |
| --- | --- | --- |
| first_word | program | first_word.c |
| fizzbuzz | program | fizzbuzz.c |
| ft_putstr | function | ft_putstr.c |
| ft_strcpy | function | ft_strcpy.c |
| ft_strlen | function | ft_strlen.c |
| ft_swap | function | ft_swap.c |
| repeat_alpha | program | repeat_alpha.c |
| rev_print | program | rev_print.c |
| rot_13 | program | rot_13.c |
| rotone | program | rotone.c |
| search_and_replace | program | search_and_replace.c |
| ulstr | program | ulstr.c |

Levels 2–4 metadata is reviewed in the follow-on plan, alongside their drivers.

- [ ] **Step 5: Commit**

```bash
git add tools/ data/
git commit -m "feat(import): generate exercise data from the vendored upstream

One-shot generator; its output is committed and hand-reviewed. It prints a
REVIEW line for each exercise whose kind or expected_file it had to guess,
so the ambiguous ones cannot be missed. Level 1 metadata is confirmed;
levels 2-4 are confirmed alongside their drivers.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 4: Sandbox — running a child with a timeout

**Files:**
- Create: `internal/sandbox/run.go`
- Test: `internal/sandbox/run_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `sandbox.Result{Stdout, Stderr string; ExitCode int; Signal string; TimedOut bool}` and `sandbox.Run(ctx context.Context, bin string, args []string, stdin string, timeout time.Duration) (Result, error)`.

- [ ] **Step 1: Write the failing test**

Create `internal/sandbox/run_test.go`:

```go
package sandbox

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunCapturesStdout(t *testing.T) {
	got, err := Run(context.Background(), "/bin/echo", []string{"hello"}, "", time.Second)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Stdout != "hello\n" {
		t.Errorf("Stdout = %q, want %q", got.Stdout, "hello\n")
	}
	if got.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", got.ExitCode)
	}
	if got.TimedOut {
		t.Error("TimedOut = true, want false")
	}
}

func TestRunCapturesExitCode(t *testing.T) {
	got, err := Run(context.Background(), "/bin/sh", []string{"-c", "exit 3"}, "", time.Second)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", got.ExitCode)
	}
}

func TestRunFeedsStdin(t *testing.T) {
	got, err := Run(context.Background(), "/bin/cat", nil, "piped\n", time.Second)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Stdout != "piped\n" {
		t.Errorf("Stdout = %q, want %q", got.Stdout, "piped\n")
	}
}

func TestRunReportsSignal(t *testing.T) {
	got, err := Run(context.Background(), "/bin/sh", []string{"-c", "kill -SEGV $$"}, "", time.Second)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Signal != "SIGSEGV" {
		t.Errorf("Signal = %q, want %q", got.Signal, "SIGSEGV")
	}
}

func TestRunTimesOut(t *testing.T) {
	start := time.Now()
	got, err := Run(context.Background(), "/bin/sh", []string{"-c", "sleep 30"}, "", 200*time.Millisecond)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !got.TimedOut {
		t.Error("TimedOut = false, want true")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Run() took %v, want it to give up promptly", elapsed)
	}
}

func TestRunTimeoutKillsGrandchildren(t *testing.T) {
	// A child that backgrounds work must not outlive the timeout. If the
	// process group is not killed, this Run blocks until the grandchild's
	// sleep finishes, well past the deadline.
	start := time.Now()
	got, err := Run(context.Background(), "/bin/sh",
		[]string{"-c", "sleep 30 & wait"}, "", 200*time.Millisecond)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !got.TimedOut {
		t.Error("TimedOut = false, want true")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Run() took %v; the grandchild was not killed", elapsed)
	}
}

func TestRunMissingBinaryIsAnError(t *testing.T) {
	_, err := Run(context.Background(), "/nonexistent/binary", nil, "", time.Second)
	if err == nil {
		t.Fatal("Run() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "binary") {
		t.Logf("error was %v", err) // message content is not contractual
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/sandbox/ -v`
Expected: FAIL — `undefined: Run`.

- [ ] **Step 3: Write the implementation**

Create `internal/sandbox/run.go`:

```go
// Package sandbox runs candidate binaries under a time limit.
//
// The timeout is enforced here in Go rather than by shelling out to
// timeout(1), which macOS does not ship.
package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Result is the observable outcome of one run.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Signal   string // e.g. "SIGSEGV"; empty when the process exited normally
	TimedOut bool
}

// Run executes bin with args, feeding it stdin, and gives up after timeout.
//
// The child is put in its own process group so that on timeout the whole
// group can be killed: a candidate program that forks must not leave work
// running after its attempt is over.
//
// A non-nil error means the run could not be performed at all. A program that
// crashes or is killed is a successful Run with that outcome recorded in the
// Result.
func Run(ctx context.Context, bin string, args []string, stdin string, timeout time.Duration) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.Command(bin, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdin = strings.NewReader(stdin)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return Result{}, fmt.Errorf("starting binary %s: %w", bin, err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	var result Result
	select {
	case err := <-done:
		result = describe(err)
	case <-ctx.Done():
		// Negating the pid addresses the whole process group.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		<-done
		result = Result{TimedOut: true, ExitCode: -1}
	}

	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	return result, nil
}

// describe turns the error from cmd.Wait into an exit code and signal name.
func describe(err error) Result {
	if err == nil {
		return Result{ExitCode: 0}
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return Result{ExitCode: -1}
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return Result{ExitCode: exitErr.ExitCode()}
	}
	if status.Signaled() {
		return Result{ExitCode: -1, Signal: signalName(status.Signal())}
	}
	return Result{ExitCode: status.ExitStatus()}
}

// signalName renders a signal the way a candidate would recognise it.
func signalName(sig syscall.Signal) string {
	switch sig {
	case syscall.SIGSEGV:
		return "SIGSEGV"
	case syscall.SIGABRT:
		return "SIGABRT"
	case syscall.SIGBUS:
		return "SIGBUS"
	case syscall.SIGFPE:
		return "SIGFPE"
	case syscall.SIGKILL:
		return "SIGKILL"
	default:
		return sig.String()
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/sandbox/ -v`
Expected: PASS, all seven tests. `TestRunTimeoutKillsGrandchildren` completing in well under a second is the one that proves the process-group kill works.

- [ ] **Step 5: Commit**

```bash
git add internal/sandbox/
git commit -m "feat(sandbox): run candidate binaries under a time limit

Timeout is enforced in Go because macOS ships no timeout(1). The child
runs in its own process group so a program that forks cannot leave work
running past its deadline; the grandchild test is what proves it.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 5: Forbidden-function detection

The check that distinguishes this trainer from running the code by hand. It catches the single most common real-exam failure: calling `printf` when the subject allows only `write`.

**Files:**
- Create: `internal/grader/symbols.go`, `internal/grader/builtins.go`
- Test: `internal/grader/symbols_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `grader.parseNMOutput(string) []string`, `grader.normalizeSymbol(string) string`, `grader.Forbidden(undefined, allowed []string) []string`, `grader.DefinesMain([]string) bool`, `grader.objectSymbols(ctx, objPath string) (undefined, defined []string, err error)`.

- [ ] **Step 1: Write the failing unit test**

Create `internal/grader/symbols_test.go`:

```go
package grader

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseNMOutput(t *testing.T) {
	// Linux nm prints bare names; macOS nm prefixes them with an underscore.
	out := "                 U memcpy\n" +
		"                 U printf\n" +
		"0000000000000000 T ft_strlen\n" +
		"                 U write\n"
	undefined, defined := parseNMOutput(out)
	wantU := []string{"memcpy", "printf", "write"}
	if !reflect.DeepEqual(undefined, wantU) {
		t.Errorf("undefined = %v, want %v", undefined, wantU)
	}
	wantD := []string{"ft_strlen"}
	if !reflect.DeepEqual(defined, wantD) {
		t.Errorf("defined = %v, want %v", defined, wantD)
	}
}

func TestParseNMOutputStripsDarwinUnderscore(t *testing.T) {
	out := "                 U _printf\n0000000000000000 T _main\n"
	undefined, defined := parseNMOutput(out)
	if !reflect.DeepEqual(undefined, []string{"printf"}) {
		t.Errorf("undefined = %v, want [printf]", undefined)
	}
	if !reflect.DeepEqual(defined, []string{"main"}) {
		t.Errorf("defined = %v, want [main]", defined)
	}
}

func TestForbiddenSubtractsAllowedAndBuiltins(t *testing.T) {
	undefined := []string{"printf", "write", "memcpy", "__stack_chk_fail"}
	got := Forbidden(undefined, []string{"write"})
	want := []string{"printf"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Forbidden() = %v, want %v", got, want)
	}
}

func TestForbiddenWithNothingAllowed(t *testing.T) {
	got := Forbidden([]string{"write"}, nil)
	if !reflect.DeepEqual(got, []string{"write"}) {
		t.Errorf("Forbidden() = %v, want [write]", got)
	}
}

func TestForbiddenAllowsACleanObject(t *testing.T) {
	if got := Forbidden([]string{"memcpy"}, nil); len(got) != 0 {
		t.Errorf("Forbidden() = %v, want empty: memcpy is compiler-emitted", got)
	}
}

func TestDefinesMain(t *testing.T) {
	if !DefinesMain([]string{"ft_strlen", "main"}) {
		t.Error("DefinesMain() = false, want true")
	}
	if DefinesMain([]string{"ft_strlen"}) {
		t.Error("DefinesMain() = true, want false")
	}
}

// compileFixture builds a C snippet and returns the object file path.
func compileFixture(t *testing.T, src string) string {
	t.Helper()
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("cc not available")
	}
	dir := t.TempDir()
	cPath := filepath.Join(dir, "fixture.c")
	if err := os.WriteFile(cPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	oPath := filepath.Join(dir, "fixture.o")
	cmd := exec.Command("cc", "-Wall", "-Wextra", "-Werror", "-c", cPath, "-o", oPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compiling fixture: %v\n%s", err, out)
	}
	return oPath
}

func TestObjectSymbolsFindsPrintf(t *testing.T) {
	obj := compileFixture(t, `
#include <stdio.h>
int ft_thing(void) { printf("x"); return 0; }
`)
	undefined, defined, err := objectSymbols(context.Background(), obj)
	if err != nil {
		t.Fatalf("objectSymbols() error = %v", err)
	}
	if got := Forbidden(undefined, []string{"write"}); !reflect.DeepEqual(got, []string{"printf"}) {
		t.Errorf("Forbidden() = %v, want [printf]", got)
	}
	if !contains(defined, "ft_thing") {
		t.Errorf("defined = %v, want it to contain ft_thing", defined)
	}
}

func TestObjectSymbolsToleratesCompilerEmittedMemcpy(t *testing.T) {
	// Assigning a large struct makes the compiler synthesise a memcpy the
	// candidate never wrote. Reporting that as a forbidden call would be a
	// false failure, so this must come back clean.
	obj := compileFixture(t, `
typedef struct { char buf[512]; } big_t;
big_t copy_it(big_t a) { big_t b = a; return b; }
`)
	undefined, _, err := objectSymbols(context.Background(), obj)
	if err != nil {
		t.Fatalf("objectSymbols() error = %v", err)
	}
	if got := Forbidden(undefined, nil); len(got) != 0 {
		t.Errorf("Forbidden() = %v, want empty; undefined was %v", got, undefined)
	}
}

func TestObjectSymbolsIgnoresRecursion(t *testing.T) {
	// A recursive call resolves inside the translation unit and never appears
	// as undefined, so no special case is needed for the exercise's own name.
	obj := compileFixture(t, `
int ft_list_size(void *p) { if (!p) return 0; return 1 + ft_list_size(*(void **)p); }
`)
	undefined, _, err := objectSymbols(context.Background(), obj)
	if err != nil {
		t.Fatalf("objectSymbols() error = %v", err)
	}
	if got := Forbidden(undefined, nil); len(got) != 0 {
		t.Errorf("Forbidden() = %v, want empty", got)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/grader/ -v`
Expected: FAIL — `undefined: parseNMOutput`.

- [ ] **Step 3: Write the builtin allowlist**

Create `internal/grader/builtins.go`:

```go
package grader

import "runtime"

// compilerEmitted lists symbols a compiler synthesises from code that never
// names them: memcpy and memset for aggregate assignment and large
// initialisers, the stack-protector hooks, and the runtime entry points a
// linked object refers to.
//
// Reporting one of these as a forbidden call would fail a candidate for
// something they did not write, so they are subtracted before the check. The
// lists differ by platform because clang on macOS and gcc on Linux emit
// different names.
var compilerEmitted = func() map[string]bool {
	shared := []string{
		"memcpy", "memmove", "memset", "bcopy",
		"__stack_chk_fail", "__stack_chk_guard",
	}
	platform := map[string][]string{
		"darwin": {
			"___stack_chk_fail", "___stack_chk_guard",
			"dyld_stub_binder", "__Unwind_Resume",
		},
		"linux": {
			"__stack_chk_fail_local", "_GLOBAL_OFFSET_TABLE_",
			"__gmon_start__", "_Unwind_Resume",
		},
	}
	set := make(map[string]bool)
	for _, s := range append(shared, platform[runtime.GOOS]...) {
		set[s] = true
	}
	return set
}()
```

- [ ] **Step 4: Write the symbol reader**

Create `internal/grader/symbols.go`:

```go
package grader

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
)

// objectSymbols runs nm over an object file and returns its undefined and
// defined symbol names.
func objectSymbols(ctx context.Context, objPath string) (undefined, defined []string, err error) {
	out, err := exec.CommandContext(ctx, "nm", objPath).Output()
	if err != nil {
		return nil, nil, fmt.Errorf("running nm on %s: %w", objPath, err)
	}
	undefined, defined = parseNMOutput(string(out))
	return undefined, defined, nil
}

// parseNMOutput reads nm's listing. Each line is an optional address, a
// one-letter type, and a name; type U means the symbol is undefined, so the
// object refers to it without defining it.
//
// macOS nm prefixes every C symbol with an underscore, Linux nm does not, so
// one leading underscore is stripped on darwin to make the two comparable.
func parseNMOutput(out string) (undefined, defined []string) {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		symType, name := fields[len(fields)-2], fields[len(fields)-1]
		if len(symType) != 1 {
			continue
		}
		name = normalizeSymbol(name)
		if symType == "U" {
			undefined = append(undefined, name)
		} else if symType == strings.ToUpper(symType) {
			// An uppercase type means a global definition; lowercase means a
			// file-local one, which no other object can call.
			defined = append(defined, name)
		}
	}
	sort.Strings(undefined)
	sort.Strings(defined)
	return dedupe(undefined), dedupe(defined)
}

// normalizeSymbol strips the leading underscore macOS adds to C symbols.
func normalizeSymbol(name string) string {
	if runtime.GOOS == "darwin" {
		return strings.TrimPrefix(name, "_")
	}
	return name
}

// Forbidden returns the undefined symbols that the subject does not allow.
//
// Nothing needs subtracting for functions the candidate defined themselves,
// including a recursive call to the exercise function: those resolve inside
// the translation unit and never appear as undefined.
func Forbidden(undefined, allowed []string) []string {
	permitted := make(map[string]bool, len(allowed))
	for _, fn := range allowed {
		permitted[fn] = true
	}
	var out []string
	for _, sym := range undefined {
		if permitted[sym] || compilerEmitted[sym] {
			continue
		}
		out = append(out, sym)
	}
	return out
}

// DefinesMain reports whether an object defines main. A function exercise
// whose file defines main would fail to link against the driver, and saying
// so plainly beats showing a duplicate-symbol linker error.
func DefinesMain(defined []string) bool {
	for _, sym := range defined {
		if sym == "main" {
			return true
		}
	}
	return false
}

func dedupe(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := in[:1]
	for _, s := range in[1:] {
		if s != out[len(out)-1] {
			out = append(out, s)
		}
	}
	return out
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/grader/ -v`
Expected: PASS. `TestObjectSymbolsToleratesCompilerEmittedMemcpy` is the guard against false failures; if it fails, add the symbol it names to `compilerEmitted` rather than loosening the check.

- [ ] **Step 6: Commit**

```bash
git add internal/grader/symbols.go internal/grader/builtins.go internal/grader/symbols_test.go
git commit -m "feat(grader): detect calls to functions a subject forbids

Reads undefined symbols from nm and subtracts the subject's allowed list
and a per-platform list of compiler-emitted symbols. Without that
subtraction a struct assignment synthesises memcpy and fails a candidate
for something they never wrote, which the fixture test pins down.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 6: The grading pipeline

**Files:**
- Create: `internal/grader/verdict.go`, `internal/grader/grade.go`
- Test: `internal/grader/grade_test.go`

**Interfaces:**
- Consumes: `catalog.Exercise` (Task 2), `sandbox.Run` (Task 4), `Forbidden`/`DefinesMain`/`objectSymbols` (Task 5).
- Produces: `grader.Status`, `grader.Verdict{Status, Summary, Detail}`, `(Verdict).Passed() bool`, `grader.New() *Grader`, `(*Grader).Grade(ctx, catalog.Exercise, srcPath string) (Verdict, error)`, `grader.CheckToolchain() error`.

- [ ] **Step 1: Write the verdict type**

Create `internal/grader/verdict.go`:

```go
package grader

// Status is the outcome of a grading attempt.
type Status string

const (
	StatusOK           Status = "OK"
	StatusMissingFile  Status = "missing file"
	StatusCompileError Status = "compile error"
	StatusForbidden    Status = "forbidden function"
	StatusUnexpectedMain Status = "unexpected main"
	StatusWrongOutput  Status = "wrong output"
	StatusCrash        Status = "crash"
	StatusTimeout      Status = "timeout"
)

// Verdict is the result of grading one attempt.
type Verdict struct {
	Status  Status
	Summary string // one line, always set
	Detail  string // compiler output or an input/output comparison; may be empty
}

// Passed reports whether the attempt would score points.
func (v Verdict) Passed() bool { return v.Status == StatusOK }
```

- [ ] **Step 2: Write the failing test**

Create `internal/grader/grade_test.go`:

```go
package grader

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"exam02/internal/catalog"
)

func strlenExercise() catalog.Exercise {
	return catalog.Exercise{
		Meta: catalog.Meta{
			Name: "ft_strlen", Level: 1, Kind: catalog.KindFunction,
			ExpectedFile: "ft_strlen.c", AllowedFunctions: nil,
			Prototype: "int ft_strlen(char *str);",
		},
		Reference: "int ft_strlen(char *str){int i=0;while(str[i])i++;return i;}",
		Driver: `#include <stdio.h>
int ft_strlen(char *str);
int main(int argc, char **argv) {
	if (argc < 2) return 1;
	printf("%d\n", ft_strlen(argv[1]));
	return 0;
}`,
		Cases: []catalog.Case{
			{Args: []string{"hello"}},
			{Args: []string{""}},
		},
	}
}

func rot13Exercise() catalog.Exercise {
	return catalog.Exercise{
		Meta: catalog.Meta{
			Name: "rot_13", Level: 1, Kind: catalog.KindProgram,
			ExpectedFile: "rot_13.c", AllowedFunctions: []string{"write"},
		},
		Reference: `#include <unistd.h>
int main(int argc, char **argv) {
	int i = 0;
	if (argc == 2)
		while (argv[1][i]) {
			char c = argv[1][i];
			if (c >= 'a' && c <= 'z') c = (c - 'a' + 13) % 26 + 'a';
			else if (c >= 'A' && c <= 'Z') c = (c - 'A' + 13) % 26 + 'A';
			write(1, &c, 1);
			i++;
		}
	write(1, "\n", 1);
	return 0;
}`,
		Cases: []catalog.Case{{Args: []string{"abc"}}, {Args: []string{}}},
	}
}

// gradeSource writes src as the exercise's expected file and grades it.
func gradeSource(t *testing.T, ex catalog.Exercise, src string) Verdict {
	t.Helper()
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("cc not available")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, ex.ExpectedFile)
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := New().Grade(context.Background(), ex, path)
	if err != nil {
		t.Fatalf("Grade() error = %v", err)
	}
	return v
}

func TestGradeAcceptsTheReference(t *testing.T) {
	ex := strlenExercise()
	if v := gradeSource(t, ex, ex.Reference); !v.Passed() {
		t.Errorf("Grade() = %s: %s\n%s", v.Status, v.Summary, v.Detail)
	}
}

func TestGradeRejectsAWrongAnswer(t *testing.T) {
	ex := strlenExercise()
	v := gradeSource(t, ex, "int ft_strlen(char *str){int i=0;while(str[i])i++;return i+1;}")
	if v.Status != StatusWrongOutput {
		t.Errorf("Status = %q, want %q (summary: %s)", v.Status, StatusWrongOutput, v.Summary)
	}
	if v.Detail == "" {
		t.Error("Detail is empty; a wrong answer must show the comparison")
	}
}

func TestGradeRejectsAWarning(t *testing.T) {
	// An unused parameter is -Wextra plus -Werror, which is a real exam failure.
	ex := strlenExercise()
	v := gradeSource(t, ex, "int ft_strlen(char *str){(void)0;return 7;}")
	if v.Status != StatusCompileError {
		t.Errorf("Status = %q, want %q", v.Status, StatusCompileError)
	}
	if v.Detail == "" {
		t.Error("Detail is empty; a compile error must show the compiler output")
	}
}

func TestGradeRejectsAForbiddenFunction(t *testing.T) {
	ex := strlenExercise()
	v := gradeSource(t, ex, `#include <string.h>
int ft_strlen(char *str){return (int)strlen(str);}`)
	if v.Status != StatusForbidden {
		t.Errorf("Status = %q, want %q", v.Status, StatusForbidden)
	}
	if !strings.Contains(v.Summary, "strlen") {
		t.Errorf("Summary = %q, want it to name strlen", v.Summary)
	}
}

func TestGradeRejectsAMainInAFunctionExercise(t *testing.T) {
	ex := strlenExercise()
	v := gradeSource(t, ex, `int ft_strlen(char *str){int i=0;while(str[i])i++;return i;}
int main(void){return 0;}`)
	if v.Status != StatusUnexpectedMain {
		t.Errorf("Status = %q, want %q", v.Status, StatusUnexpectedMain)
	}
}

func TestGradeReportsAMissingFile(t *testing.T) {
	ex := strlenExercise()
	v, err := New().Grade(context.Background(), ex, filepath.Join(t.TempDir(), "absent.c"))
	if err != nil {
		t.Fatalf("Grade() error = %v", err)
	}
	if v.Status != StatusMissingFile {
		t.Errorf("Status = %q, want %q", v.Status, StatusMissingFile)
	}
}

func TestGradeReportsACrash(t *testing.T) {
	ex := strlenExercise()
	v := gradeSource(t, ex, "int ft_strlen(char *str){(void)str;return *(int *)0;}")
	if v.Status != StatusCrash {
		t.Errorf("Status = %q, want %q (summary: %s)", v.Status, StatusCrash, v.Summary)
	}
}

func TestGradeProgramKindAcceptsTheReference(t *testing.T) {
	ex := rot13Exercise()
	if v := gradeSource(t, ex, ex.Reference); !v.Passed() {
		t.Errorf("Grade() = %s: %s\n%s", v.Status, v.Summary, v.Detail)
	}
}

func TestGradeProgramKindRejectsAWrongAnswer(t *testing.T) {
	ex := rot13Exercise()
	// rot 12 instead of rot 13.
	v := gradeSource(t, ex, `#include <unistd.h>
int main(int argc, char **argv) {
	int i = 0;
	if (argc == 2)
		while (argv[1][i]) {
			char c = argv[1][i];
			if (c >= 'a' && c <= 'z') c = (c - 'a' + 12) % 26 + 'a';
			write(1, &c, 1);
			i++;
		}
	write(1, "\n", 1);
	return 0;
}`)
	if v.Status != StatusWrongOutput {
		t.Errorf("Status = %q, want %q", v.Status, StatusWrongOutput)
	}
}
```

- [ ] **Step 3: Run it to verify it fails**

Run: `go test ./internal/grader/ -run TestGrade -v`
Expected: FAIL — `undefined: New`.

- [ ] **Step 4: Write the pipeline**

Create `internal/grader/grade.go`:

```go
package grader

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"exam02/internal/catalog"
	"exam02/internal/sandbox"
)

// compileFlags are the flags the real exam compiles with. A warning is a
// failure, so -Werror is not negotiable.
var compileFlags = []string{"-Wall", "-Wextra", "-Werror"}

// Grader compiles and runs candidate solutions against the reference.
type Grader struct {
	CC string
	NM string
}

// New returns a Grader using the system toolchain.
func New() *Grader { return &Grader{CC: "cc", NM: "nm"} }

// CheckToolchain reports whether the tools the grader needs are present.
func CheckToolchain() error {
	for _, tool := range []string{"cc", "nm"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s not found in PATH", tool)
		}
	}
	return nil
}

// Grade compiles the candidate's file, checks it for forbidden calls, and runs
// it against the reference solution on every case.
//
// A non-nil error means grading could not be carried out — a missing compiler,
// an unwritable temporary directory. A candidate whose code is wrong is not an
// error: it is a Verdict that did not pass.
func (g *Grader) Grade(ctx context.Context, ex catalog.Exercise, srcPath string) (Verdict, error) {
	source, err := os.ReadFile(srcPath)
	if err != nil {
		return Verdict{
			Status:  StatusMissingFile,
			Summary: fmt.Sprintf("%s not found", ex.ExpectedFile),
		}, nil
	}

	work, err := os.MkdirTemp("", "exam02-grade-")
	if err != nil {
		return Verdict{}, fmt.Errorf("creating work directory: %w", err)
	}
	defer os.RemoveAll(work)

	// Compile the candidate.
	userObj := filepath.Join(work, "user.o")
	if out, err := g.compile(ctx, srcPath, userObj); err != nil {
		return Verdict{
			Status:  StatusCompileError,
			Summary: "your file did not compile with -Wall -Wextra -Werror",
			Detail:  out,
		}, nil
	}

	// Check what it calls before running anything.
	undefined, defined, err := objectSymbols(ctx, userObj)
	if err != nil {
		return Verdict{}, err
	}
	if bad := Forbidden(undefined, ex.AllowedFunctions); len(bad) > 0 {
		return Verdict{
			Status:  StatusForbidden,
			Summary: fmt.Sprintf("forbidden function: %s", strings.Join(bad, ", ")),
			Detail:  allowedSummary(ex),
		}, nil
	}
	if ex.Kind == catalog.KindFunction && DefinesMain(defined) {
		return Verdict{
			Status:  StatusUnexpectedMain,
			Summary: "your file defines main; this exercise expects only the function",
			Detail:  "Submit " + ex.Prototype + " on its own. Comment out any main you used for testing.",
		}, nil
	}

	// Build the reference the same way.
	refPath := filepath.Join(work, "reference.c")
	if err := os.WriteFile(refPath, []byte(ex.Reference), 0o644); err != nil {
		return Verdict{}, err
	}
	refObj := filepath.Join(work, "ref.o")
	if out, err := g.compile(ctx, refPath, refObj); err != nil {
		return Verdict{}, fmt.Errorf("the reference solution for %s did not compile: %s", ex.Name, out)
	}

	userBin, refBin := filepath.Join(work, "user_bin"), filepath.Join(work, "ref_bin")
	var extra []string
	if ex.Kind == catalog.KindFunction {
		driverPath := filepath.Join(work, "driver.c")
		if err := os.WriteFile(driverPath, []byte(ex.Driver), 0o644); err != nil {
			return Verdict{}, err
		}
		driverObj := filepath.Join(work, "driver.o")
		// The driver is ours, not the candidate's, so its own calls are never
		// subject to the allowed-functions check.
		if out, err := g.compile(ctx, driverPath, driverObj); err != nil {
			return Verdict{}, fmt.Errorf("the driver for %s did not compile: %s", ex.Name, out)
		}
		extra = []string{driverObj}
	}
	if out, err := g.link(ctx, append([]string{userObj}, extra...), userBin); err != nil {
		return Verdict{
			Status:  StatusCompileError,
			Summary: "your file compiled but did not link",
			Detail:  out,
		}, nil
	}
	if out, err := g.link(ctx, append([]string{refObj}, extra...), refBin); err != nil {
		return Verdict{}, fmt.Errorf("the reference for %s did not link: %s", ex.Name, out)
	}

	// Compare on every case, stopping at the first difference.
	timeout := time.Duration(ex.Timeout()) * time.Millisecond
	for _, c := range ex.Cases {
		want, err := sandbox.Run(ctx, refBin, c.Args, c.Stdin, timeout)
		if err != nil {
			return Verdict{}, fmt.Errorf("running the reference: %w", err)
		}
		got, err := sandbox.Run(ctx, userBin, c.Args, c.Stdin, timeout)
		if err != nil {
			return Verdict{}, fmt.Errorf("running your program: %w", err)
		}
		if v, ok := compare(ex, c, want, got, timeout); !ok {
			return v, nil
		}
	}
	return Verdict{Status: StatusOK, Summary: "OK"}, nil
}

// compare checks one case. The bool is false when the candidate differs.
func compare(ex catalog.Exercise, c catalog.Case, want, got sandbox.Result, timeout time.Duration) (Verdict, bool) {
	invocation := describeInvocation(ex, c)
	switch {
	case got.TimedOut:
		return Verdict{
			Status:  StatusTimeout,
			Summary: fmt.Sprintf("timed out after %s", timeout),
			Detail:  invocation + "\n\nAn infinite loop is the usual cause.",
		}, false
	case got.Signal != "":
		return Verdict{
			Status:  StatusCrash,
			Summary: crashSummary(got.Signal),
			Detail:  invocation,
		}, false
	case got.Stdout != want.Stdout:
		return Verdict{
			Status:  StatusWrongOutput,
			Summary: "wrong output",
			Detail: fmt.Sprintf("%s\n\nexpected: %s\nactual:   %s",
				invocation, visible(want.Stdout), visible(got.Stdout)),
		}, false
	case got.ExitCode != want.ExitCode:
		return Verdict{
			Status:  StatusWrongOutput,
			Summary: fmt.Sprintf("wrong exit status: expected %d, got %d", want.ExitCode, got.ExitCode),
			Detail:  invocation,
		}, false
	}
	return Verdict{}, true
}

func (g *Grader) compile(ctx context.Context, src, obj string) (string, error) {
	args := append(append([]string{}, compileFlags...), "-c", src, "-o", obj)
	out, err := exec.CommandContext(ctx, g.CC, args...).CombinedOutput()
	return string(out), err
}

func (g *Grader) link(ctx context.Context, objs []string, bin string) (string, error) {
	out, err := exec.CommandContext(ctx, g.CC, append(objs, "-o", bin)...).CombinedOutput()
	return string(out), err
}

func crashSummary(signal string) string {
	switch signal {
	case "SIGSEGV", "SIGBUS":
		return "segfault"
	case "SIGABRT":
		return "aborted"
	case "SIGFPE":
		return "arithmetic error, usually division by zero"
	default:
		return "killed by " + signal
	}
}

// describeInvocation renders the case the way the candidate would run it.
func describeInvocation(ex catalog.Exercise, c catalog.Case) string {
	parts := []string{"$> ./" + ex.Name}
	for _, a := range c.Args {
		parts = append(parts, quote(a))
	}
	line := strings.Join(parts, " ")
	if c.Stdin != "" {
		line += "\nstdin: " + visible(c.Stdin)
	}
	return line
}

func quote(s string) string {
	if s == "" || strings.ContainsAny(s, " \t\n\"") {
		return fmt.Sprintf("%q", s)
	}
	return s
}

// visible renders output so that trailing whitespace and missing newlines are
// not invisible in the diff, the way cat -e is used in the subjects.
func visible(s string) string {
	s = strings.ReplaceAll(s, "\n", "$\n")
	if s == "" {
		return "(nothing)"
	}
	return s
}

func allowedSummary(ex catalog.Exercise) string {
	if len(ex.AllowedFunctions) == 0 {
		return "This exercise allows no library functions at all."
	}
	return "This exercise allows only: " + strings.Join(ex.AllowedFunctions, ", ")
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/grader/ -v`
Expected: PASS, every test.

- [ ] **Step 6: Commit**

```bash
git add internal/grader/
git commit -m "feat(grader): compile, check symbols, and diff against the reference

The reference solution is the oracle: candidate and reference are built
with identical flags, linked against the same driver, and run on the same
inputs. A difference in stdout or exit status fails the attempt. This is
why no expected outputs have to be written by hand.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 7: Level 1 drivers, cases, and the suite that validates them

The vertical slice. When this task is green, the whole pipeline is proven against 12 real exercises and writing the remaining 44 is mechanical.

**Files:**
- Create: `data/exercises/ft_putstr/driver.c`, `data/exercises/ft_strcpy/driver.c`, `data/exercises/ft_strlen/driver.c`, `data/exercises/ft_swap/driver.c`
- Create: `cases.txt` in all 12 Level 1 exercise directories
- Create: `internal/catalog/embed.go`, `internal/grader/suite_test.go`
- Test: `internal/grader/suite_test.go`

**Interfaces:**
- Consumes: everything from Tasks 2, 5 and 6.
- Produces: `catalog.Embedded() (*Catalog, error)` — the production catalog, read from the embedded data.

- [ ] **Step 1: Write the four drivers**

A driver supplies `main`, calls the candidate's function, and prints the result so that a wrong answer shows up as different text. Drivers may use `printf` freely: they are our code, compiled separately, and are never subject to the allowed-functions check.

`data/exercises/ft_strlen/driver.c`:

```c
#include <stdio.h>

int	ft_strlen(char *str);

int	main(int argc, char **argv)
{
	if (argc < 2)
		return (1);
	printf("%d\n", ft_strlen(argv[1]));
	return (0);
}
```

`data/exercises/ft_putstr/driver.c`:

```c
#include <unistd.h>

void	ft_putstr(char *str);

int	main(int argc, char **argv)
{
	if (argc < 2)
		return (1);
	ft_putstr(argv[1]);
	write(1, "|end\n", 5);
	return (0);
}
```

The `|end` marker matters: without it a function that writes nothing and one that writes a trailing space look identical once the terminal has rendered them.

`data/exercises/ft_strcpy/driver.c`:

```c
#include <stdio.h>
#include <string.h>

char	*ft_strcpy(char *s1, char *s2);

int	main(int argc, char **argv)
{
	char	dest[1024];
	char	*ret;

	if (argc < 2)
		return (1);
	memset(dest, '#', sizeof(dest));
	ret = ft_strcpy(dest, argv[1]);
	printf("[%s]\n", dest);
	printf("returns_dest=%d\n", ret == dest);
	printf("byte_after_nul=%c\n", dest[strlen(argv[1]) + 1]);
	return (0);
}
```

Filling the buffer with `#` and printing the byte after the terminator catches a copy that writes too much, which a plain string comparison would miss.

`data/exercises/ft_swap/driver.c`:

```c
#include <stdio.h>
#include <stdlib.h>

void	ft_swap(int *a, int *b);

int	main(int argc, char **argv)
{
	int	a;
	int	b;

	if (argc < 3)
		return (1);
	a = atoi(argv[1]);
	b = atoi(argv[2]);
	ft_swap(&a, &b);
	printf("%d %d\n", a, b);
	return (0);
}
```

- [ ] **Step 2: Write the case files**

Every case file must include the argument counts the subject calls out, because "if the number of arguments is not 1, display a newline" is the rule candidates forget most often.

`data/exercises/ft_strlen/cases.txt`:
```
{"args": ["hello"]}
{"args": [""]}
{"args": ["a"]}
{"args": ["with spaces and	tabs"]}
{"args": ["0123456789012345678901234567890123456789"]}
```

`data/exercises/ft_putstr/cases.txt`:
```
{"args": ["hello"]}
{"args": [""]}
{"args": ["multi\nline"]}
{"args": ["trailing space "]}
```

`data/exercises/ft_strcpy/cases.txt`:
```
{"args": ["hello"]}
{"args": [""]}
{"args": ["a"]}
{"args": ["a longer string to copy"]}
```

`data/exercises/ft_swap/cases.txt`:
```
{"args": ["1", "2"]}
{"args": ["0", "0"]}
{"args": ["-5", "5"]}
{"args": ["2147483647", "-2147483648"]}
```

`data/exercises/first_word/cases.txt`:
```
{"args": ["FOR PONY"]}
{"args": ["this        ...       is sparta, then?"]}
{"args": [""]}
{"args": ["   "]}
{"args": ["  lorem,ipsum  "]}
{"args": []}
{"args": ["a", "b"]}
```

`data/exercises/fizzbuzz/cases.txt`:
```
{"args": []}
```

`data/exercises/repeat_alpha/cases.txt`:
```
{"args": ["abc"]}
{"args": ["Alex Antoine Hilaire Zebulon"]}
{"args": ["abcdefghijklmnopqrstuvwxyz"]}
{"args": [""]}
{"args": []}
{"args": ["a", "b"]}
```

`data/exercises/rev_print/cases.txt`:
```
{"args": ["zaz"]}
{"args": ["dub0 a POIL"]}
{"args": [""]}
{"args": ["a"]}
{"args": []}
{"args": ["a", "b"]}
```

`data/exercises/rot_13/cases.txt`:
```
{"args": ["abc"]}
{"args": ["Salut, comment tu vas ? 32 chars cheer"]}
{"args": ["zZ"]}
{"args": ["nopqrstuvwxyz"]}
{"args": [""]}
{"args": []}
{"args": ["a", "b"]}
```

`data/exercises/rotone/cases.txt`:
```
{"args": ["abc"]}
{"args": ["zZ"]}
{"args": ["Les stagiaires du staff ne sentent pas toujours tres bon."]}
{"args": [""]}
{"args": []}
{"args": ["a", "b"]}
```

`data/exercises/search_and_replace/cases.txt`:
```
{"args": ["Papache est un sabre", "a", "o"]}
{"args": ["zaz", "z", "m"]}
{"args": ["zaz", "v", "m"]}
{"args": ["", "a", "b"]}
{"args": ["abc", "a"]}
{"args": []}
{"args": ["abc", "ab", "c"]}
```

`data/exercises/ulstr/cases.txt`:
```
{"args": ["L'eSPrit nE peUt plUs pRogResSer s'Il staGne."]}
{"args": ["abcXYZ123"]}
{"args": [""]}
{"args": []}
{"args": ["a", "b"]}
```

- [ ] **Step 3: Write the embed file**

Create `internal/catalog/embed.go`:

```go
package catalog

import (
	"embed"
	"io/fs"
)

//go:embed all:exercises
var embedded embed.FS

// Embedded returns the catalog compiled into the binary. Using all: means
// files are included even if a future exercise directory starts with a dot or
// an underscore, which go:embed would otherwise skip.
func Embedded() (*Catalog, error) {
	return Load(embedded)
}

// EmbeddedFS exposes the raw filesystem, for tests that need to check what
// was actually compiled in.
func EmbeddedFS() fs.FS { return embedded }
```

- [ ] **Step 4: Move the data under the catalog package**

`go:embed` can only read files in or below its own directory, so the generated tree has to live beside `embed.go`.

```bash
git mv data/exercises internal/catalog/exercises
rmdir data 2>/dev/null || true
```

Then update the importer's default output path in `tools/import/main.go`:

```go
	dst := flag.String("dst", "internal/catalog/exercises", "output directory")
```

- [ ] **Step 5: Write the failing suite test**

Create `internal/grader/suite_test.go`:

```go
package grader

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"exam02/internal/catalog"
)

// readyExercises returns the exercises that have the driver and cases needed
// to be graded. Levels 2 to 4 arrive in a follow-on plan; until then they are
// present as subjects but not yet gradable, and skipping them here keeps the
// suite honest rather than silently green.
func readyExercises(t *testing.T) []catalog.Exercise {
	t.Helper()
	c, err := catalog.Embedded()
	if err != nil {
		t.Fatalf("catalog.Embedded() error = %v", err)
	}
	var out []catalog.Exercise
	for _, ex := range c.All() {
		if len(ex.Cases) > 0 {
			out = append(out, ex)
		}
	}
	return out
}

func requireCC(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("cc not available")
	}
}

// TestEveryReferenceSolutionPasses is the load-bearing test of this project.
// It validates every driver and every case file at once: if a driver calls the
// function wrongly or a case file expects something the subject does not
// require, the reference itself fails here.
func TestEveryReferenceSolutionPasses(t *testing.T) {
	requireCC(t)
	exercises := readyExercises(t)
	if len(exercises) < 12 {
		t.Fatalf("only %d exercises are ready; Level 1 has 12", len(exercises))
	}
	g := New()
	for _, ex := range exercises {
		t.Run(ex.Name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, ex.ExpectedFile)
			if err := os.WriteFile(path, []byte(ex.Reference), 0o644); err != nil {
				t.Fatal(err)
			}
			v, err := g.Grade(context.Background(), ex, path)
			if err != nil {
				t.Fatalf("Grade() error = %v", err)
			}
			if !v.Passed() {
				t.Errorf("the reference solution failed its own grader: %s: %s\n%s",
					v.Status, v.Summary, v.Detail)
			}
		})
	}
}

// TestMutatedReferencesFail proves the case files actually discriminate.
// Without it, a driver that printed nothing would sail through the test above
// while testing nothing at all.
func TestMutatedReferencesFail(t *testing.T) {
	requireCC(t)
	g := New()
	for _, ex := range readyExercises(t) {
		mutated, ok := mutate(ex.Reference)
		if !ok {
			t.Errorf("%s: no mutation could be applied; add a case that would catch one", ex.Name)
			continue
		}
		t.Run(ex.Name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, ex.ExpectedFile)
			if err := os.WriteFile(path, []byte(mutated), 0o644); err != nil {
				t.Fatal(err)
			}
			v, err := g.Grade(context.Background(), ex, path)
			if err != nil {
				t.Fatalf("Grade() error = %v", err)
			}
			if v.Passed() {
				t.Errorf("a mutated reference still passed; the cases do not discriminate")
			}
		})
	}
}

// mutate makes one behaviour-changing edit to C source. It tries each
// replacement in turn and stops at the first that applies.
func mutate(src string) (string, bool) {
	replacements := []struct{ from, to string }{
		{" <= ", " < "},
		{" >= ", " > "},
		{" == ", " != "},
		{" + 1", " + 2"},
		{"i++", "i += 2"},
		{" 13", " 12"},
		{" 26", " 25"},
	}
	for _, r := range replacements {
		if strings.Contains(src, r.from) {
			return strings.Replace(src, r.from, r.to, 1), true
		}
	}
	return "", false
}
```

- [ ] **Step 6: Run the suite and fix what it finds**

Run: `go test ./internal/grader/ -run 'TestEveryReference|TestMutated' -v`

Expected on the first run: some failures. That is the point of the task. Work through them:

- A reference failing with `wrong output` usually means the driver prints something the reference cannot produce — fix the driver.
- A reference failing with `forbidden function` means `allowed_functions` in that `meta.yaml` is wrong — check it against the subject.
- A mutated reference still passing means the case file does not exercise the mutated path — add a case that does.
- `no mutation could be applied` means the reference contains none of the patterns; add a pattern to `mutate` that fits it.

Iterate until both tests are fully green for all 12 Level 1 exercises.

- [ ] **Step 7: Run the whole suite**

Run: `go test ./... -v`
Expected: PASS everywhere.

- [ ] **Step 8: Commit**

```bash
git add internal/catalog/ internal/grader/suite_test.go tools/
git commit -m "feat(data): Level 1 drivers, cases, and the suite that validates them

Grading all 12 reference solutions proves every driver and case file at
once. The mutation test is what keeps that honest: without it a driver
that printed nothing would pass while testing nothing.

Exercise data moves under internal/catalog because go:embed cannot read
above its own directory.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 8: Configuration and persisted state

**Files:**
- Create: `internal/store/config.go`, `internal/store/state.go`
- Test: `internal/store/state_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `store.Config{ExamDuration time.Duration; PointsPerExercise, PassMark int}`, `store.DefaultConfig()`, `store.LoadConfig(dir string) (Config, error)`, `store.Stats{Attempts, Passes int; LastStatus string; LastAt time.Time}`, `store.State`, `store.Load(dir string) (*State, error)`, `(*State).Save(dir string) error`, `(*State).Record(name, status string, passed bool)`, `(*State).Weakest([]string) string`, `store.Dir() (string, error)`.

- [ ] **Step 1: Write the failing test**

Create `internal/store/state_test.go`:

```go
package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFromAnEmptyDirectoryGivesEmptyState(t *testing.T) {
	s, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(s.Exercises) != 0 {
		t.Errorf("Exercises = %v, want empty", s.Exercises)
	}
	if s.Exam != nil {
		t.Errorf("Exam = %v, want nil", s.Exam)
	}
}

func TestRecordAndRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	s.Record("ft_strlen", "OK", true)
	s.Record("rot_13", "wrong output", false)
	s.Record("rot_13", "wrong output", false)
	if err := s.Save(dir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	again, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := again.Exercises["ft_strlen"]; got.Attempts != 1 || got.Passes != 1 {
		t.Errorf("ft_strlen = %+v, want 1 attempt 1 pass", got)
	}
	if got := again.Exercises["rot_13"]; got.Attempts != 2 || got.Passes != 0 {
		t.Errorf("rot_13 = %+v, want 2 attempts 0 passes", got)
	}
	if again.Exercises["rot_13"].LastStatus != "wrong output" {
		t.Errorf("LastStatus = %q, want %q", again.Exercises["rot_13"].LastStatus, "wrong output")
	}
}

func TestSaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir)
	s.Record("ft_strlen", "OK", true)
	if err := s.Save(dir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	// A crash mid-save must not leave a partial file behind under a name the
	// next Load would read.
	for _, e := range entries {
		if e.Name() != stateFile {
			t.Errorf("Save() left %q behind, want only %q", e.Name(), stateFile)
		}
	}
}

func TestLoadIgnoresCorruptState(t *testing.T) {
	// Losing statistics is annoying; refusing to start is worse.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, stateFile), []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v, want it to recover", err)
	}
	if len(s.Exercises) != 0 {
		t.Errorf("Exercises = %v, want empty", s.Exercises)
	}
}

func TestWeakestPicksMostFailedThenLeastRecent(t *testing.T) {
	s, _ := Load(t.TempDir())
	pool := []string{"a", "b", "c"}

	// Never attempted beats attempted: you cannot be weaker than untested.
	s.Exercises["a"] = &Stats{Attempts: 5, Passes: 1, LastAt: time.Now()}
	if got := s.Weakest(pool); got != "b" && got != "c" {
		t.Errorf("Weakest() = %q, want an unattempted exercise", got)
	}

	old := time.Now().Add(-72 * time.Hour)
	s.Exercises["b"] = &Stats{Attempts: 3, Passes: 3, LastAt: time.Now()}
	s.Exercises["c"] = &Stats{Attempts: 4, Passes: 0, LastAt: old}
	if got := s.Weakest(pool); got != "c" {
		t.Errorf("Weakest() = %q, want c: it has the most failures", got)
	}
}

func TestDefaultConfigMatchesTheRealExam(t *testing.T) {
	c := DefaultConfig()
	if c.ExamDuration != 3*time.Hour {
		t.Errorf("ExamDuration = %v, want 3h", c.ExamDuration)
	}
	if c.PointsPerExercise != 25 {
		t.Errorf("PointsPerExercise = %d, want 25", c.PointsPerExercise)
	}
	if c.PassMark != 100 {
		t.Errorf("PassMark = %d, want 100", c.PassMark)
	}
}

func TestLoadConfigFallsBackToDefaults(t *testing.T) {
	c, err := LoadConfig(t.TempDir())
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if c != DefaultConfig() {
		t.Errorf("LoadConfig() = %+v, want the defaults", c)
	}
}

func TestLoadConfigReadsOverrides(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, configFile),
		[]byte("exam_duration: 90m\npoints_per_exercise: 25\npass_mark: 50\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if c.ExamDuration != 90*time.Minute {
		t.Errorf("ExamDuration = %v, want 90m", c.ExamDuration)
	}
	if c.PassMark != 50 {
		t.Errorf("PassMark = %d, want 50", c.PassMark)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/store/ -v`
Expected: FAIL — `undefined: Load`.

- [ ] **Step 3: Write the config**

Create `internal/store/config.go`:

```go
// Package store holds the user's configuration and their saved progress.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const configFile = "config.yaml"

// Config holds the exam rules. They are configurable so a variant ruleset can
// be tried without a rebuild, but the defaults are the real exam's.
type Config struct {
	ExamDuration      time.Duration
	PointsPerExercise int
	PassMark          int
}

// DefaultConfig returns the real Exam Rank 02 rules: three hours, 25 points
// per exercise, 100 points to pass, which means clearing all four levels.
func DefaultConfig() Config {
	return Config{
		ExamDuration:      3 * time.Hour,
		PointsPerExercise: 25,
		PassMark:          100,
	}
}

// configFileFormat is the on-disk shape, kept separate so the duration can be
// written the way a person would type it.
type configFileFormat struct {
	ExamDuration      string `yaml:"exam_duration"`
	PointsPerExercise int    `yaml:"points_per_exercise"`
	PassMark          int    `yaml:"pass_mark"`
}

// LoadConfig reads config.yaml from dir, falling back to the defaults when it
// is absent. A malformed file is an error rather than a silent fallback:
// having quietly ignored the rules someone wrote would be worse than saying so.
func LoadConfig(dir string) (Config, error) {
	c := DefaultConfig()
	raw, err := os.ReadFile(filepath.Join(dir, configFile))
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, fmt.Errorf("reading %s: %w", configFile, err)
	}
	var f configFileFormat
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return c, fmt.Errorf("parsing %s: %w", configFile, err)
	}
	if f.ExamDuration != "" {
		d, err := time.ParseDuration(f.ExamDuration)
		if err != nil {
			return c, fmt.Errorf("exam_duration %q: %w", f.ExamDuration, err)
		}
		c.ExamDuration = d
	}
	if f.PointsPerExercise > 0 {
		c.PointsPerExercise = f.PointsPerExercise
	}
	if f.PassMark > 0 {
		c.PassMark = f.PassMark
	}
	return c, nil
}
```

- [ ] **Step 4: Write the state**

Create `internal/store/state.go`:

```go
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const stateFile = "state.json"

// Stats is what is remembered about one exercise.
type Stats struct {
	Attempts   int       `json:"attempts"`
	Passes     int       `json:"passes"`
	LastStatus string    `json:"last_status"`
	LastAt     time.Time `json:"last_at"`
}

// Failures is how many attempts did not pass.
func (s Stats) Failures() int { return s.Attempts - s.Passes }

// State is the persisted progress. ExamJSON carries an in-progress exam as
// opaque JSON so that store does not depend on the session package and the
// two can change independently.
type State struct {
	Exercises map[string]*Stats `json:"exercises"`
	Exam      json.RawMessage   `json:"exam,omitempty"`
}

// Dir returns the per-user directory holding config and state.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating the user config directory: %w", err)
	}
	dir := filepath.Join(base, "exam02")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}
	return dir, nil
}

// Load reads the saved state from dir. A missing or unreadable file gives
// empty state rather than an error: losing statistics is a nuisance, but
// refusing to start the application over them would be worse.
func Load(dir string) (*State, error) {
	s := &State{Exercises: map[string]*Stats{}}
	raw, err := os.ReadFile(filepath.Join(dir, stateFile))
	if err != nil {
		return s, nil
	}
	if err := json.Unmarshal(raw, s); err != nil {
		return &State{Exercises: map[string]*Stats{}}, nil
	}
	if s.Exercises == nil {
		s.Exercises = map[string]*Stats{}
	}
	return s, nil
}

// Save writes the state to dir atomically: a crash partway through must not
// leave a truncated file where the next Load would find it.
func (s *State) Save(dir string) error {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".state-*.tmp")
	if err != nil {
		return fmt.Errorf("creating a temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename has succeeded

	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return fmt.Errorf("writing state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing state: %w", err)
	}
	if err := os.Rename(tmpName, filepath.Join(dir, stateFile)); err != nil {
		return fmt.Errorf("replacing state: %w", err)
	}
	return nil
}

// Record notes one grading attempt.
func (s *State) Record(exercise, status string, passed bool) {
	st, ok := s.Exercises[exercise]
	if !ok {
		st = &Stats{}
		s.Exercises[exercise] = st
	}
	st.Attempts++
	if passed {
		st.Passes++
	}
	st.LastStatus = status
	st.LastAt = time.Now()
}

// Weakest picks the exercise from pool most worth practising: the one never
// attempted, else the one with the most failures, ties going to whichever was
// attempted longest ago. Returns "" for an empty pool.
func (s *State) Weakest(pool []string) string {
	var best string
	var bestFailures int
	var bestAt time.Time

	for _, name := range pool {
		st, seen := s.Exercises[name]
		if !seen || st.Attempts == 0 {
			return name
		}
		failures := st.Failures()
		if best == "" || failures > bestFailures ||
			(failures == bestFailures && st.LastAt.Before(bestAt)) {
			best, bestFailures, bestAt = name, failures, st.LastAt
		}
	}
	return best
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/store/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/store/
git commit -m "feat(store): exam configuration and persisted progress

Saves atomically through a temporary file and rename, and treats corrupt
state as empty rather than refusing to start: losing statistics is a
nuisance, being unable to open the application is not.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 9: Exam state machine

Pure logic, no compiler and no terminal, so every rule is testable directly.

**Files:**
- Create: `internal/session/exam.go`
- Test: `internal/session/exam_test.go`

**Interfaces:**
- Consumes: `store.Config` (Task 8), `catalog.Catalog` (Task 2).
- Produces: `session.Exam`, `session.NewExam(cfg store.Config, pools map[int][]string, rng *rand.Rand, now time.Time) *Exam`, `(*Exam).Current() string`, `(*Exam).Level() int`, `(*Exam).Record(passed bool, now time.Time)`, `(*Exam).Points() int`, `(*Exam).Remaining(now time.Time) time.Duration`, `(*Exam).Over(now time.Time) bool`, `(*Exam).Passed() bool`, `(*Exam).Attempts() []session.Attempt`.

- [ ] **Step 1: Write the failing test**

Create `internal/session/exam_test.go`:

```go
package session

import (
	"math/rand"
	"testing"
	"time"

	"exam02/internal/store"
)

func testPools() map[int][]string {
	return map[int][]string{
		1: {"ft_strlen", "rot_13"},
		2: {"ft_atoi"},
		3: {"epur_str"},
		4: {"ft_split"},
	}
}

func newTestExam(t *testing.T, start time.Time) *Exam {
	t.Helper()
	e, err := NewExam(store.DefaultConfig(), testPools(), rand.New(rand.NewSource(1)), start)
	if err != nil {
		t.Fatalf("NewExam() error = %v", err)
	}
	return e
}

func TestNewExamStartsAtLevelOne(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	if e.Level() != 1 {
		t.Errorf("Level() = %d, want 1", e.Level())
	}
	if e.Points() != 0 {
		t.Errorf("Points() = %d, want 0", e.Points())
	}
	if got := e.Current(); got != "ft_strlen" && got != "rot_13" {
		t.Errorf("Current() = %q, want one of the level 1 pool", got)
	}
}

func TestPassingAdvancesALevelAndScores(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	e.Record(true, start.Add(time.Minute))
	if e.Level() != 2 {
		t.Errorf("Level() = %d, want 2", e.Level())
	}
	if e.Points() != 25 {
		t.Errorf("Points() = %d, want 25", e.Points())
	}
	if e.Current() != "ft_atoi" {
		t.Errorf("Current() = %q, want ft_atoi", e.Current())
	}
}

func TestFailingRedrawsFromTheSameLevel(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	e.Record(false, start.Add(time.Minute))
	if e.Level() != 1 {
		t.Errorf("Level() = %d, want 1", e.Level())
	}
	if e.Points() != 0 {
		t.Errorf("Points() = %d, want 0", e.Points())
	}
	if e.Over(start.Add(2 * time.Minute)) {
		t.Error("Over() = true; a failed attempt must not end the exam")
	}
}

func TestClearingAllFourLevelsPasses(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	for i := 0; i < 4; i++ {
		e.Record(true, start.Add(time.Duration(i)*time.Minute))
	}
	if e.Points() != 100 {
		t.Errorf("Points() = %d, want 100", e.Points())
	}
	if !e.Passed() {
		t.Error("Passed() = false, want true at 100 points")
	}
	if !e.Over(start.Add(5 * time.Minute)) {
		t.Error("Over() = false, want true after the last level")
	}
	if e.Current() != "" {
		t.Errorf("Current() = %q, want empty once the exam is over", e.Current())
	}
}

func TestThreeOfFourLevelsFails(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	for i := 0; i < 3; i++ {
		e.Record(true, start.Add(time.Duration(i)*time.Minute))
	}
	if e.Points() != 75 {
		t.Errorf("Points() = %d, want 75", e.Points())
	}
	if e.Passed() {
		t.Error("Passed() = true at 75 points, want false: the pass mark is 100")
	}
}

func TestTimeExpiryEndsTheExam(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	if e.Over(start.Add(2 * time.Hour)) {
		t.Error("Over() = true after 2h, want false")
	}
	if !e.Over(start.Add(3 * time.Hour)) {
		t.Error("Over() = false after 3h, want true")
	}
	if got := e.Remaining(start.Add(4 * time.Hour)); got != 0 {
		t.Errorf("Remaining() = %v past the deadline, want 0", got)
	}
}

func TestRemainingCountsDown(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	if got := e.Remaining(start.Add(time.Hour)); got != 2*time.Hour {
		t.Errorf("Remaining() = %v, want 2h", got)
	}
}

func TestAttemptsAreRecordedInOrder(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	first := e.Current()
	e.Record(false, start.Add(time.Minute))
	e.Record(true, start.Add(2*time.Minute))

	got := e.Attempts()
	if len(got) != 2 {
		t.Fatalf("Attempts() = %d, want 2", len(got))
	}
	if got[0].Exercise != first || got[0].Passed {
		t.Errorf("first attempt = %+v, want %q failed", got[0], first)
	}
	if !got[1].Passed {
		t.Errorf("second attempt = %+v, want passed", got[1])
	}
}

func TestRecordAfterTheExamIsOverIsIgnored(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	for i := 0; i < 4; i++ {
		e.Record(true, start.Add(time.Duration(i)*time.Minute))
	}
	e.Record(true, start.Add(10*time.Minute))
	if e.Points() != 100 {
		t.Errorf("Points() = %d, want it capped at 100", e.Points())
	}
}

func TestNewExamNeedsEveryLevelStocked(t *testing.T) {
	pools := testPools()
	delete(pools, 3)
	if _, err := NewExam(store.DefaultConfig(), pools, rand.New(rand.NewSource(1)), time.Now()); err == nil {
		t.Fatal("NewExam() error = nil, want an error when a level has no exercises")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/session/ -v`
Expected: FAIL — `undefined: NewExam`.

- [ ] **Step 3: Write the state machine**

Create `internal/session/exam.go`:

```go
// Package session holds the rules of an exam run: which exercise is current,
// what a pass or a fail does, and when the run is over. It touches neither the
// compiler nor the terminal, so every rule here is testable on its own.
package session

import (
	"fmt"
	"math/rand"
	"time"

	"exam02/internal/store"
)

// levels is how many levels an exam covers. One exercise is drawn from each,
// so four passes at 25 points each is the 100-point pass mark.
const levels = 4

// Attempt is one graded submission during an exam.
type Attempt struct {
	Exercise string    `json:"exercise"`
	Level    int       `json:"level"`
	Passed   bool      `json:"passed"`
	At       time.Time `json:"at"`
}

// Exam is one run: four exercises, one drawn from each level, against a clock.
type Exam struct {
	StartedAt time.Time     `json:"started_at"`
	Duration  time.Duration `json:"duration"`
	PerLevel  int           `json:"per_level"`
	PassMark  int           `json:"pass_mark"`

	CurrentLevel    int       `json:"current_level"`
	CurrentExercise string    `json:"current_exercise"`
	Cleared         int       `json:"cleared"`
	History         []Attempt `json:"history"`

	pools map[int][]string
	rng   *rand.Rand
}

// NewExam starts a run, drawing the first exercise from the Level 1 pool.
//
// pools maps a level to the names of its exercises. Every level must have at
// least one: an exam that cannot draw an exercise for a level could never be
// passed, and failing at the start says so more clearly than failing later.
func NewExam(cfg store.Config, pools map[int][]string, rng *rand.Rand, now time.Time) (*Exam, error) {
	for level := 1; level <= levels; level++ {
		if len(pools[level]) == 0 {
			return nil, fmt.Errorf("level %d has no exercises to draw from", level)
		}
	}
	e := &Exam{
		StartedAt:    now,
		Duration:     cfg.ExamDuration,
		PerLevel:     cfg.PointsPerExercise,
		PassMark:     cfg.PassMark,
		CurrentLevel: 1,
		pools:        pools,
		rng:          rng,
	}
	e.draw()
	return e, nil
}

// Attach restores the pools and randomness after an exam has been decoded from
// saved state, which cannot carry either.
func (e *Exam) Attach(pools map[int][]string, rng *rand.Rand) {
	e.pools, e.rng = pools, rng
}

// draw picks a fresh exercise from the current level's pool, avoiding the one
// just attempted where the pool is big enough to allow it.
func (e *Exam) draw() {
	pool := e.pools[e.CurrentLevel]
	if len(pool) == 0 {
		e.CurrentExercise = ""
		return
	}
	if len(pool) == 1 {
		e.CurrentExercise = pool[0]
		return
	}
	for {
		candidate := pool[e.rng.Intn(len(pool))]
		if candidate != e.CurrentExercise {
			e.CurrentExercise = candidate
			return
		}
	}
}

// Record notes the outcome of grading the current exercise. Passing clears the
// level and draws from the next one; failing draws again from the same level,
// which costs time rather than points.
func (e *Exam) Record(passed bool, now time.Time) {
	if e.Over(now) {
		return
	}
	e.History = append(e.History, Attempt{
		Exercise: e.CurrentExercise,
		Level:    e.CurrentLevel,
		Passed:   passed,
		At:       now,
	})
	if !passed {
		e.draw()
		return
	}
	e.Cleared++
	if e.Cleared >= levels {
		e.CurrentExercise = ""
		return
	}
	e.CurrentLevel++
	e.draw()
}

// Current is the exercise being attempted, or "" once the exam is over.
func (e *Exam) Current() string { return e.CurrentExercise }

// Level is the level being attempted.
func (e *Exam) Level() int { return e.CurrentLevel }

// Points is the score so far.
func (e *Exam) Points() int { return e.Cleared * e.PerLevel }

// Passed reports whether the score has reached the pass mark.
func (e *Exam) Passed() bool { return e.Points() >= e.PassMark }

// Remaining is the time left, floored at zero.
func (e *Exam) Remaining(now time.Time) time.Duration {
	left := e.Duration - now.Sub(e.StartedAt)
	if left < 0 {
		return 0
	}
	return left
}

// Over reports whether the run has finished, by the clock or by clearing every
// level.
func (e *Exam) Over(now time.Time) bool {
	return e.Cleared >= levels || e.Remaining(now) == 0
}

// Attempts is the history of graded submissions, oldest first.
func (e *Exam) Attempts() []Attempt { return e.History }
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/session/ -v`
Expected: PASS.

- [ ] **Step 5: Write the failing resume test**

The spec requires an in-progress exam to survive a crash or an accidental
quit. A three hour run is far too much to lose to a stray Ctrl-C.

Append to `internal/session/exam_test.go`:

```go
func TestExamSurvivesASaveAndReload(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	e.Record(true, start.Add(time.Minute))    // clears level 1
	e.Record(false, start.Add(2*time.Minute)) // fails once at level 2

	encoded, err := Encode(e)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	restored, err := Decode(encoded, testPools(), rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if restored.Level() != 2 {
		t.Errorf("Level() = %d, want 2", restored.Level())
	}
	if restored.Points() != 25 {
		t.Errorf("Points() = %d, want 25", restored.Points())
	}
	if len(restored.Attempts()) != 2 {
		t.Errorf("Attempts() = %d, want 2", len(restored.Attempts()))
	}
	if restored.Current() == "" {
		t.Error("Current() is empty; the restored exam has nothing to work on")
	}
}

func TestRestoredExamKeepsCountingFromItsOriginalStart(t *testing.T) {
	// The clock must not restart on resume, or quitting and reopening would
	// be a way to buy unlimited time.
	start := time.Now().Add(-time.Hour)
	e := newTestExam(t, start)
	encoded, err := Encode(e)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	restored, err := Decode(encoded, testPools(), rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got := restored.Remaining(time.Now()); got > 2*time.Hour {
		t.Errorf("Remaining() = %v, want about 2h: the clock must not restart", got)
	}
}

func TestDecodeOfAFinishedExamIsNotResumable(t *testing.T) {
	start := time.Now()
	e := newTestExam(t, start)
	for i := 0; i < 4; i++ {
		e.Record(true, start.Add(time.Duration(i)*time.Minute))
	}
	encoded, err := Encode(e)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	restored, err := Decode(encoded, testPools(), rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if !restored.Over(time.Now()) {
		t.Error("Over() = false; a finished exam must not come back resumable")
	}
}
```

- [ ] **Step 6: Run it to verify it fails**

Run: `go test ./internal/session/ -run 'TestExamSurvives|TestRestored|TestDecodeOf' -v`
Expected: FAIL — `undefined: Encode`.

- [ ] **Step 7: Write Encode and Decode**

Append to `internal/session/exam.go`:

```go
// Encode serialises an exam for storage. The draw pools and the random source
// are deliberately left out: pools are rebuilt from the catalog on load, so a
// saved exam cannot pin the application to a stale copy of the exercise list.
func Encode(e *Exam) (json.RawMessage, error) {
	raw, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("encoding the exam: %w", err)
	}
	return raw, nil
}

// Decode restores a saved exam and reattaches the pools and randomness that
// could not be serialised.
//
// The start time is restored as it was, so the clock carries on from where it
// stopped. Restarting it would turn quitting and reopening into a way to buy
// unlimited time.
func Decode(raw json.RawMessage, pools map[int][]string, rng *rand.Rand) (*Exam, error) {
	var e Exam
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, fmt.Errorf("decoding the exam: %w", err)
	}
	if e.CurrentLevel < 1 || e.CurrentLevel > levels {
		return nil, fmt.Errorf("saved exam has an impossible level %d", e.CurrentLevel)
	}
	e.Attach(pools, rng)
	return &e, nil
}
```

Add `"encoding/json"` to the file's imports.

- [ ] **Step 8: Run the tests to verify they pass**

Run: `go test ./internal/session/ -v`
Expected: PASS, every test.

- [ ] **Step 9: Commit**

```bash
git add internal/session/
git commit -m "feat(session): exam rules as a testable state machine

Four exercises, one drawn per level, 25 points each against a 100-point
pass mark and a three hour clock. A failure redraws from the same level,
so it costs time rather than points. No compiler and no terminal here,
which is what makes every rule directly testable.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 10: Workspace preparation and the editor

**Files:**
- Create: `internal/workspace/workspace.go`
- Test: `internal/workspace/workspace_test.go`

**Interfaces:**
- Consumes: `catalog.Exercise` (Task 2).
- Produces: `workspace.Prepare(root string, ex catalog.Exercise) (string, error)`, `workspace.EditorCommand(path string) *exec.Cmd`.

- [ ] **Step 1: Write the failing test**

Create `internal/workspace/workspace_test.go`:

```go
package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"exam02/internal/catalog"
)

func strlenExercise() catalog.Exercise {
	return catalog.Exercise{
		Meta: catalog.Meta{
			Name: "ft_strlen", Level: 1, Kind: catalog.KindFunction,
			ExpectedFile: "ft_strlen.c", Prototype: "int ft_strlen(char *str);",
		},
	}
}

func TestPrepareCreatesTheFile(t *testing.T) {
	root := t.TempDir()
	path, err := Prepare(root, strlenExercise())
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if want := filepath.Join(root, "rendu", "ft_strlen", "ft_strlen.c"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the prepared file: %v", err)
	}
	if !strings.Contains(string(body), "int ft_strlen(char *str);") {
		t.Errorf("prepared file does not carry the prototype:\n%s", body)
	}
}

func TestPrepareNeverOverwritesExistingWork(t *testing.T) {
	// The candidate's work is sacred. Re-entering an exercise must not wipe it.
	root := t.TempDir()
	path, err := Prepare(root, strlenExercise())
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("my hard work"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Prepare(root, strlenExercise()); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "my hard work" {
		t.Errorf("file = %q, want the candidate's work untouched", body)
	}
}

func TestPrepareWritesTheProvidedHeader(t *testing.T) {
	root := t.TempDir()
	ex := strlenExercise()
	ex.Header = "ft_list.h"
	ex.HeaderContent = "typedef struct s_list t_list;\n"
	if _, err := Prepare(root, ex); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, "rendu", "ft_strlen", "ft_list.h"))
	if err != nil {
		t.Fatalf("the provided header was not written: %v", err)
	}
	if string(body) != ex.HeaderContent {
		t.Errorf("header = %q, want %q", body, ex.HeaderContent)
	}
}

func TestPrepareProgramKindHasNoPrototypeStub(t *testing.T) {
	root := t.TempDir()
	ex := strlenExercise()
	ex.Kind = catalog.KindProgram
	ex.Prototype = ""
	ex.Name = "rot_13"
	ex.ExpectedFile = "rot_13.c"
	path, err := Prepare(root, ex)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "int main(int argc, char **argv)") {
		t.Errorf("a program exercise should be seeded with a main:\n%s", body)
	}
}

func TestEditorCommandHonoursTheEnvironment(t *testing.T) {
	t.Setenv("EDITOR", "nano")
	cmd := EditorCommand("/tmp/x.c")
	if cmd.Args[0] != "nano" {
		t.Errorf("editor = %q, want nano", cmd.Args[0])
	}
	t.Setenv("EDITOR", "")
	if cmd := EditorCommand("/tmp/x.c"); cmd.Args[0] != "vim" {
		t.Errorf("editor = %q, want the vim default", cmd.Args[0])
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/workspace/ -v`
Expected: FAIL — `undefined: Prepare`.

- [ ] **Step 3: Write the implementation**

Create `internal/workspace/workspace.go`:

```go
// Package workspace prepares the directory a candidate writes their answer in.
//
// The layout mirrors the real exam: rendu/<exercise>/<file>.c under the
// working directory.
package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"exam02/internal/catalog"
)

// Prepare creates the exercise's directory and seeds its source file, and
// returns the path to that file.
//
// An existing file is left exactly as it is. Re-entering an exercise must
// never destroy work, so seeding only ever happens once.
func Prepare(root string, ex catalog.Exercise) (string, error) {
	dir := filepath.Join(root, "rendu", ex.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}

	if ex.Header != "" {
		headerPath := filepath.Join(dir, ex.Header)
		if _, err := os.Stat(headerPath); os.IsNotExist(err) {
			if err := os.WriteFile(headerPath, []byte(ex.HeaderContent), 0o644); err != nil {
				return "", fmt.Errorf("writing %s: %w", ex.Header, err)
			}
		}
	}

	path := filepath.Join(dir, ex.ExpectedFile)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := os.WriteFile(path, []byte(stub(ex)), 0o644); err != nil {
		return "", fmt.Errorf("writing %s: %w", ex.ExpectedFile, err)
	}
	return path, nil
}

// stub is the starting content of a fresh answer file: enough to compile
// against, never enough to be a hint.
func stub(ex catalog.Exercise) string {
	header := fmt.Sprintf("/* %s */\n\n", ex.Name)
	if ex.Kind == catalog.KindProgram {
		return header + "int\tmain(int argc, char **argv)\n{\n\t(void)argc;\n\t(void)argv;\n\treturn (0);\n}\n"
	}
	if ex.Prototype == "" {
		return header
	}
	// Turn the declaration into an empty definition the candidate fills in.
	body := ex.Prototype
	if body[len(body)-1] == ';' {
		body = body[:len(body)-1]
	}
	return header + body + "\n{\n}\n"
}

// EditorCommand builds the command that opens path in the candidate's editor.
func EditorCommand(path string) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/workspace/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/workspace/
git commit -m "feat(workspace): seed rendu/ the way the real exam lays it out

Seeding happens once and only once: an existing answer file is never
rewritten, so re-entering an exercise cannot destroy work in progress.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 11: TUI shell — menu and exercise screen

**Files:**
- Create: `internal/ui/styles.go`, `internal/ui/model.go`, `internal/ui/menu.go`, `internal/ui/exercise.go`
- Test: `internal/ui/model_test.go`

**Interfaces:**
- Consumes: everything from Tasks 2, 6, 8, 9, 10.
- Produces: `ui.Model`, `ui.New(deps Deps) Model`, `ui.Deps{Catalog, Grader, Config, State, StateDir, Root string}`, and the Bubble Tea `Init`/`Update`/`View` triple.

- [ ] **Step 1: Add the TUI dependencies**

```bash
go get github.com/charmbracelet/bubbletea@v1.3.10
go get github.com/charmbracelet/lipgloss@v1.1.0
```

- [ ] **Step 2: Write the styles**

Create `internal/ui/styles.go`:

```go
package ui

import "github.com/charmbracelet/lipgloss"

// Colours are chosen from the 256-colour palette so the interface looks the
// same in the terminals 42 machines actually run.
var (
	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("87"))
	styleDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleOK    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("47"))
	styleKO    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))
	styleWarn  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleKey   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
)
```

- [ ] **Step 3: Write the failing model test**

Create `internal/ui/model_test.go`:

```go
package ui

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"exam02/internal/catalog"
	"exam02/internal/grader"
	"exam02/internal/store"
)

func testCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	fsys := fstest.MapFS{}
	for _, name := range []string{"ft_strlen", "ft_atoi", "epur_str", "ft_split"} {
		level := map[string]int{"ft_strlen": 1, "ft_atoi": 2, "epur_str": 3, "ft_split": 4}[name]
		dir := "exercises/" + name + "/"
		fsys[dir+"meta.yaml"] = &fstest.MapFile{Data: []byte(
			"name: " + name + "\nlevel: " + string(rune('0'+level)) + "\nkind: program\n" +
				"expected_file: " + name + ".c\nallowed_functions: []\nprototype: \"\"\n")}
		fsys[dir+"subject.md"] = &fstest.MapFile{Data: []byte("subject for " + name)}
		fsys[dir+"reference.c"] = &fstest.MapFile{Data: []byte("int main(void){return 0;}")}
		fsys[dir+"cases.txt"] = &fstest.MapFile{Data: []byte("{\"args\": []}\n")}
	}
	c, err := catalog.Load(fsys)
	if err != nil {
		t.Fatalf("catalog.Load() error = %v", err)
	}
	return c
}

func newTestModel(t *testing.T) Model {
	t.Helper()
	state, _ := store.Load(t.TempDir())
	return New(Deps{
		Catalog:  testCatalog(t),
		Grader:   grader.New(),
		Config:   store.DefaultConfig(),
		State:    state,
		StateDir: t.TempDir(),
		Root:     t.TempDir(),
	})
}

func key(s string) tea.KeyMsg {
	if len(s) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func TestMenuIsTheFirstScreen(t *testing.T) {
	m := newTestModel(t)
	view := m.View()
	for _, want := range []string{"Exam", "Practice", "Quit"} {
		if !strings.Contains(view, want) {
			t.Errorf("menu does not offer %q:\n%s", want, view)
		}
	}
}

func TestQuittingFromTheMenu(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatal("pressing q returned no command, want tea.Quit")
	}
	if msg := cmd(); msg == nil {
		t.Error("the command produced no message, want a quit message")
	}
}

func TestStartingAnExamShowsTheSubjectAndTheClock(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(key("1"))
	view := next.View()
	if !strings.Contains(view, "subject for") {
		t.Errorf("the exam screen does not show a subject:\n%s", view)
	}
	if !strings.Contains(view, "Level 1") {
		t.Errorf("the exam screen does not show the level:\n%s", view)
	}
	if !strings.Contains(view, "0 / 100") {
		t.Errorf("the exam screen does not show points against the pass mark:\n%s", view)
	}
}

func TestPracticeListsEveryExercise(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(key("2"))
	view := next.View()
	for _, want := range []string{"ft_strlen", "ft_atoi", "epur_str", "ft_split"} {
		if !strings.Contains(view, want) {
			t.Errorf("the practice list is missing %q:\n%s", want, view)
		}
	}
}

func TestExamScreenRendersTheRemainingTime(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(key("1"))
	// Three hours, so the clock should read close to it on the first frame.
	if !strings.Contains(next.View(), "2:59") && !strings.Contains(next.View(), "3:00") {
		t.Errorf("the exam screen does not show a countdown:\n%s", next.View())
	}
}

func TestTickAdvancesTheClockWithoutEndingTheExam(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(key("1"))
	after, cmd := next.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Error("a tick did not schedule the next one; the clock would stop")
	}
	if !strings.Contains(after.View(), "Level 1") {
		t.Errorf("the exam screen was lost on a tick:\n%s", after.View())
	}
}
```

- [ ] **Step 4: Run it to verify it fails**

Run: `go test ./internal/ui/ -v`
Expected: FAIL — `undefined: New`.

- [ ] **Step 5: Write the model**

Create `internal/ui/model.go`:

```go
// Package ui is the terminal interface: a menu, a timed exam, and untimed
// practice, all driven by the grader underneath.
package ui

import (
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"exam02/internal/catalog"
	"exam02/internal/grader"
	"exam02/internal/session"
	"exam02/internal/store"
)

// screen is which view is on show.
type screen int

const (
	screenMenu screen = iota
	screenExam
	screenPractice
	screenExercise
	screenResult
	screenAbout
)

// Deps is everything the interface needs from the rest of the application,
// passed in rather than constructed here so tests can substitute their own.
type Deps struct {
	Catalog  *catalog.Catalog
	Grader   *grader.Grader
	Config   store.Config
	State    *store.State
	StateDir string
	Root     string // where rendu/ is created
}

// Model is the Bubble Tea model for the whole application.
type Model struct {
	deps Deps

	screen   screen
	previous screen

	exam     *session.Exam
	practice []catalog.Exercise
	cursor   int

	current  catalog.Exercise
	srcPath  string
	verdict  *grader.Verdict
	grading  bool
	revealed bool
	spanish  bool

	status string
	err    error
	now    time.Time
	width  int
}

// tickMsg drives the countdown.
type tickMsg time.Time

// gradedMsg carries a finished grading run back to the interface.
type gradedMsg struct {
	verdict grader.Verdict
	err     error
}

// editorFinishedMsg arrives when the candidate leaves their editor.
type editorFinishedMsg struct{ err error }

// New builds the initial model, showing the menu.
func New(deps Deps) Model {
	return Model{deps: deps, screen: screenMenu, now: time.Now()}
}

// Init starts the clock.
func (m Model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Update routes a message to the active screen.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		if m.screen == screenExam && m.exam != nil && m.exam.Over(m.now) {
			m.screen = screenResult
		}
		return m, tick()

	case gradedMsg:
		return m.applyVerdict(msg)

	case editorFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.grading {
		return m, nil // ignore input while a compile is in flight
	}
	switch m.screen {
	case screenMenu:
		return m.updateMenu(msg)
	case screenPractice:
		return m.updatePractice(msg)
	case screenExam, screenExercise:
		return m.updateExercise(msg)
	case screenResult:
		if msg.String() == "q" {
			return m, tea.Quit
		}
		m.screen = screenMenu
		return m, nil
	case screenAbout:
		m.screen = screenMenu
		return m, nil
	}
	return m, nil
}

// View renders the active screen.
func (m Model) View() string {
	switch m.screen {
	case screenMenu:
		return m.viewMenu()
	case screenPractice:
		return m.viewPractice()
	case screenExam, screenExercise:
		return m.viewExercise()
	case screenResult:
		return m.viewResult()
	case screenAbout:
		return m.viewAbout()
	}
	return ""
}

// startExam draws the first exercise and switches to the exam screen.
func (m Model) startExam() (Model, tea.Cmd) {
	pools := map[int][]string{}
	for level := 1; level <= 4; level++ {
		for _, ex := range m.deps.Catalog.ByLevel(level) {
			pools[level] = append(pools[level], ex.Name)
		}
	}
	exam, err := session.NewExam(m.deps.Config, pools, rand.New(rand.NewSource(time.Now().UnixNano())), time.Now())
	if err != nil {
		m.err = err
		return m, nil
	}
	m.exam = exam
	m.screen = screenExam
	return m.loadCurrent()
}

// loadCurrent prepares the workspace for whichever exercise is now current.
func (m Model) loadCurrent() (Model, tea.Cmd) {
	name := m.current.Name
	if m.screen == screenExam && m.exam != nil {
		name = m.exam.Current()
	}
	ex, ok := m.deps.Catalog.ByName(name)
	if !ok {
		return m, nil
	}
	m.current = ex
	m.verdict = nil
	m.revealed = false
	m.spanish = false
	path, err := prepareWorkspace(m.deps.Root, ex)
	if err != nil {
		m.err = err
		return m, nil
	}
	m.srcPath = path
	return m, nil
}
```

- [ ] **Step 6: Write the menu**

Create `internal/ui/menu.go`:

```go
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateMenu(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "1":
		return m.startExam()
	case "2":
		m.practice = m.deps.Catalog.All()
		m.cursor = 0
		m.screen = screenPractice
		return m, nil
	case "3":
		m.screen = screenAbout
		return m, nil
	}
	return m, nil
}

func (m Model) viewMenu() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Exam Rank 02 — practice") + "\n\n")

	total, passed := 0, 0
	for _, ex := range m.deps.Catalog.All() {
		total++
		if st, ok := m.deps.State.Exercises[ex.Name]; ok && st.Passes > 0 {
			passed++
		}
	}
	b.WriteString(styleDim.Render(fmt.Sprintf("%d of %d exercises passed at least once", passed, total)) + "\n\n")

	b.WriteString(styleKey.Render("1") + "  Exam        three hours, one exercise per level, 100 points to pass\n")
	b.WriteString(styleKey.Render("2") + "  Practice    pick any exercise, no clock, solutions available\n")
	b.WriteString(styleKey.Render("3") + "  About\n")
	b.WriteString(styleKey.Render("q") + "  Quit\n")

	if m.err != nil {
		b.WriteString("\n" + styleKO.Render("error: "+m.err.Error()) + "\n")
	}
	return b.String()
}

func (m Model) viewAbout() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("About") + "\n\n")
	b.WriteString("A practice tool for the 42 Common Core Exam Rank 02.\n\n")
	b.WriteString("Exercise subjects and the reference solutions used for grading come from\n")
	b.WriteString(styleDim.Render("github.com/alexhiguera/Exam_Rank_02_42_School") + "\n")
	b.WriteString("MIT licensed, Copyright (c) 2026 Alex Higuera.\n\n")
	b.WriteString(styleWarn.Render("This is practice. The real exam is graded by the Moulinette,") + "\n")
	b.WriteString(styleWarn.Render("in a closed environment, and its rules are the ones that count.") + "\n\n")
	b.WriteString(styleDim.Render("any key to go back"))
	return b.String()
}
```

- [ ] **Step 7: Write the exercise screen**

Create `internal/ui/exercise.go`:

```go
package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"exam02/internal/catalog"
	"exam02/internal/grader"
	"exam02/internal/workspace"
)

// prepareWorkspace is a variable so tests can avoid touching the filesystem.
var prepareWorkspace = workspace.Prepare

func (m Model) updateExercise(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "q", "esc":
		if m.screen == screenExam {
			m.screen = screenResult
			return m, nil
		}
		m.screen = screenPractice
		return m, nil

	case "g":
		m.grading = true
		m.status = "compiling…"
		return m, m.gradeCmd()

	case "e":
		cmd := workspace.EditorCommand(m.srcPath)
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return editorFinishedMsg{err: err}
		})

	case "r":
		if m.screen == screenPractice || m.screen == screenExercise {
			m.revealed = !m.revealed
		}
		return m, nil

	case "x":
		if m.screen == screenExercise && m.current.Spanish != "" {
			m.spanish = !m.spanish
		}
		return m, nil
	}
	return m, nil
}

// gradeCmd runs the grader off the interface goroutine so the clock keeps
// ticking while a compile is in flight.
func (m Model) gradeCmd() tea.Cmd {
	ex, path, g := m.current, m.srcPath, m.deps.Grader
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		v, err := g.Grade(ctx, ex, path)
		return gradedMsg{verdict: v, err: err}
	}
}

// applyVerdict records the outcome and moves the exam on.
func (m Model) applyVerdict(msg gradedMsg) (Model, tea.Cmd) {
	m.grading = false
	m.status = ""
	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}
	v := msg.verdict
	m.verdict = &v

	m.deps.State.Record(m.current.Name, string(v.Status), v.Passed())
	if err := m.deps.State.Save(m.deps.StateDir); err != nil {
		m.err = err
	}

	if m.screen == screenExam && m.exam != nil {
		m.exam.Record(v.Passed(), time.Now())
		if m.exam.Over(time.Now()) {
			m.screen = screenResult
			return m, nil
		}
		if v.Passed() {
			return m.loadCurrent()
		}
		// A failure redraws a different exercise from the same level; the
		// verdict stays on screen until the candidate moves on.
		next, cmd := m.loadCurrent()
		next.verdict = &v
		return next, cmd
	}
	return m, nil
}

func (m Model) viewExercise() string {
	var b strings.Builder

	if m.screen == screenExam && m.exam != nil {
		remaining := m.exam.Remaining(m.now)
		b.WriteString(styleTitle.Render(fmt.Sprintf(
			"Level %d   %d / %d points   %s left",
			m.exam.Level(), m.exam.Points(), m.deps.Config.PassMark, clock(remaining))))
		if remaining < 10*time.Minute {
			b.WriteString("  " + styleWarn.Render("!"))
		}
		b.WriteString("\n\n")
	} else {
		b.WriteString(styleTitle.Render(fmt.Sprintf("%s   level %d", m.current.Name, m.current.Level)) + "\n\n")
	}

	b.WriteString(styleBox.Render(strings.TrimSpace(m.current.Subject)) + "\n\n")
	b.WriteString(styleDim.Render("write your answer in  ") + m.srcPath + "\n")
	if len(m.current.AllowedFunctions) > 0 {
		b.WriteString(styleDim.Render("allowed: ") + strings.Join(m.current.AllowedFunctions, ", ") + "\n")
	} else {
		b.WriteString(styleDim.Render("allowed: nothing") + "\n")
	}
	b.WriteString("\n")

	switch {
	case m.grading:
		b.WriteString(m.status + "\n")
	case m.verdict != nil:
		b.WriteString(renderVerdict(*m.verdict) + "\n")
	}

	if m.revealed {
		b.WriteString("\n" + styleWarn.Render("reference solution") + "\n")
		b.WriteString(styleBox.Render(strings.TrimSpace(m.current.Reference)) + "\n")
	}
	if m.spanish {
		b.WriteString("\n" + styleBox.Render(strings.TrimSpace(m.current.Spanish)) + "\n")
	}

	b.WriteString("\n" + m.keyHints())
	return b.String()
}

func (m Model) keyHints() string {
	hints := []string{
		styleKey.Render("g") + " grade",
		styleKey.Render("e") + " edit",
	}
	if m.screen == screenExercise {
		hints = append(hints, styleKey.Render("r")+" solution")
		if m.current.Spanish != "" {
			hints = append(hints, styleKey.Render("x")+" explicación")
		}
	}
	hints = append(hints, styleKey.Render("q")+" back")
	return styleDim.Render(strings.Join(hints, "   "))
}

func renderVerdict(v grader.Verdict) string {
	if v.Passed() {
		return styleOK.Render("OK") + "  " + v.Summary
	}
	out := styleKO.Render("KO") + "  " + v.Summary
	if v.Detail != "" {
		out += "\n\n" + styleBox.Render(strings.TrimSpace(v.Detail))
	}
	return out
}

// clock renders a duration as h:mm:ss.
func clock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Seconds())
	return fmt.Sprintf("%d:%02d:%02d", total/3600, (total/60)%60, total%60)
}

// subjectTitle is used by the practice list.
func subjectTitle(ex catalog.Exercise) string {
	return fmt.Sprintf("%-22s level %d", ex.Name, ex.Level)
}
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `go test ./internal/ui/ -v`
Expected: FAIL on the practice and result views, which Task 12 adds. The menu, exam and tick tests must pass.

- [ ] **Step 9: Commit**

```bash
git add go.mod go.sum internal/ui/
git commit -m "feat(ui): menu, exam header and exercise screen

Grading runs as a Bubble Tea command rather than inline so the countdown
keeps ticking while a compile is in flight. The exam header shows points
against the pass mark and the time left, the two numbers that actually
drive decisions during a real exam.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 12: Practice list and result screen

**Files:**
- Create: `internal/ui/practice.go`, `internal/ui/result.go`
- Modify: `internal/ui/model_test.go` — add the tests below.

**Interfaces:**
- Consumes: `Model` from Task 11, `store.State.Weakest` from Task 8.
- Produces: `(Model).updatePractice`, `(Model).viewPractice`, `(Model).viewResult`.

- [ ] **Step 1: Write the failing tests**

Append to `internal/ui/model_test.go`:

```go
func TestPracticeShowsPerExerciseStats(t *testing.T) {
	state, _ := store.Load(t.TempDir())
	state.Record("ft_strlen", "OK", true)
	state.Record("ft_atoi", "wrong output", false)
	m := New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: t.TempDir(),
	})
	next, _ := m.Update(key("2"))
	view := next.View()
	if !strings.Contains(view, "1/1") {
		t.Errorf("the practice list does not show ft_strlen's record:\n%s", view)
	}
	if !strings.Contains(view, "0/1") {
		t.Errorf("the practice list does not show ft_atoi's record:\n%s", view)
	}
}

func TestPracticeCursorMoves(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(key("2"))
	down, _ := next.Update(tea.KeyMsg{Type: tea.KeyDown})
	if down.cursor != 1 {
		t.Errorf("cursor = %d after down, want 1", down.cursor)
	}
	up, _ := down.Update(tea.KeyMsg{Type: tea.KeyUp})
	if up.cursor != 0 {
		t.Errorf("cursor = %d after up, want 0", up.cursor)
	}
	// The cursor must not run off the top of the list.
	again, _ := up.Update(tea.KeyMsg{Type: tea.KeyUp})
	if again.cursor != 0 {
		t.Errorf("cursor = %d at the top, want it to stay at 0", again.cursor)
	}
}

func TestPracticeDrillPicksTheWeakest(t *testing.T) {
	state, _ := store.Load(t.TempDir())
	// Everything attempted once, ft_split failed the most.
	state.Record("ft_strlen", "OK", true)
	state.Record("ft_atoi", "OK", true)
	state.Record("epur_str", "OK", true)
	state.Record("ft_split", "wrong output", false)
	state.Record("ft_split", "wrong output", false)
	m := New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: t.TempDir(),
	})
	list, _ := m.Update(key("2"))
	drill, _ := list.Update(key("w"))
	if drill.current.Name != "ft_split" {
		t.Errorf("drill chose %q, want ft_split, the most failed", drill.current.Name)
	}
}

func TestResultScreenStatesPassOrFail(t *testing.T) {
	m := newTestModel(t)
	exam, _ := m.Update(key("1"))
	exam.screen = screenResult
	view := exam.View()
	if !strings.Contains(view, "0 / 100") {
		t.Errorf("the result screen does not show the score:\n%s", view)
	}
	if !strings.Contains(strings.ToLower(view), "fail") {
		t.Errorf("the result screen does not state the outcome:\n%s", view)
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/ui/ -run 'TestPractice|TestResult' -v`
Expected: FAIL — `undefined: (Model).updatePractice`.

- [ ] **Step 3: Write the practice screen**

Create `internal/ui/practice.go`:

```go
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updatePractice(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "q", "esc":
		m.screen = screenMenu
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		if m.cursor < len(m.practice)-1 {
			m.cursor++
		}
		return m, nil

	case "enter":
		if len(m.practice) == 0 {
			return m, nil
		}
		m.current = m.practice[m.cursor]
		m.screen = screenExercise
		return m.loadCurrent()

	case "w":
		// Drill whichever exercise the record says is weakest.
		names := make([]string, 0, len(m.practice))
		for _, ex := range m.practice {
			names = append(names, ex.Name)
		}
		weakest := m.deps.State.Weakest(names)
		ex, ok := m.deps.Catalog.ByName(weakest)
		if !ok {
			return m, nil
		}
		m.current = ex
		m.screen = screenExercise
		return m.loadCurrent()
	}
	return m, nil
}

func (m Model) viewPractice() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Practice") + "\n\n")

	for i, ex := range m.practice {
		marker := "  "
		if i == m.cursor {
			marker = styleKey.Render("> ")
		}
		record := styleDim.Render("     —")
		if st, ok := m.deps.State.Exercises[ex.Name]; ok && st.Attempts > 0 {
			text := fmt.Sprintf("%5s", fmt.Sprintf("%d/%d", st.Passes, st.Attempts))
			if st.Passes > 0 {
				record = styleOK.Render(text)
			} else {
				record = styleKO.Render(text)
			}
		}
		b.WriteString(marker + subjectTitle(ex) + "  " + record + "\n")
	}

	b.WriteString("\n" + styleDim.Render(
		styleKey.Render("enter")+" open   "+
			styleKey.Render("w")+" drill weakest   "+
			styleKey.Render("q")+" back"))
	return b.String()
}
```

- [ ] **Step 4: Write the result screen**

Create `internal/ui/result.go`:

```go
package ui

import (
	"fmt"
	"strings"
)

func (m Model) viewResult() string {
	var b strings.Builder
	if m.exam == nil {
		return "no exam to report on"
	}

	points, mark := m.exam.Points(), m.deps.Config.PassMark
	headline := styleKO.Render("FAILED")
	if m.exam.Passed() {
		headline = styleOK.Render("PASSED")
	}
	b.WriteString(styleTitle.Render("Exam over") + "   " + headline + "\n\n")
	b.WriteString(fmt.Sprintf("%d / %d points\n", points, mark))
	b.WriteString(fmt.Sprintf("%s remaining on the clock\n\n", clock(m.exam.Remaining(m.now))))

	attempts := m.exam.Attempts()
	if len(attempts) == 0 {
		b.WriteString(styleDim.Render("nothing was submitted") + "\n")
	} else {
		b.WriteString(styleDim.Render("attempts") + "\n")
		for _, a := range attempts {
			outcome := styleKO.Render("KO")
			if a.Passed {
				outcome = styleOK.Render("OK")
			}
			b.WriteString(fmt.Sprintf("  %s  level %d  %s\n", outcome, a.Level, a.Exercise))
		}
	}

	if !m.exam.Passed() {
		b.WriteString("\n" + styleDim.Render(
			"The pass mark is every level cleared. Practice mode has no clock.") + "\n")
	}
	b.WriteString("\n" + styleDim.Render(
		styleKey.Render("q")+" quit   any other key returns to the menu"))
	return b.String()
}
```

- [ ] **Step 5: Write the failing resume test**

Append to `internal/ui/model_test.go`:

```go
func TestMenuOffersResumeOnlyWhenAnExamIsInFlight(t *testing.T) {
	m := newTestModel(t)
	if strings.Contains(m.View(), "Resume") {
		t.Errorf("the menu offers Resume with no exam in flight:\n%s", m.View())
	}

	pools := map[int][]string{1: {"ft_strlen"}, 2: {"ft_atoi"}, 3: {"epur_str"}, 4: {"ft_split"}}
	exam, err := session.NewExam(store.DefaultConfig(), pools, rand.New(rand.NewSource(1)), time.Now())
	if err != nil {
		t.Fatalf("NewExam() error = %v", err)
	}
	state, _ := store.Load(t.TempDir())
	withExam := New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: t.TempDir(), ResumeExam: exam,
	})
	if !strings.Contains(withExam.View(), "Resume") {
		t.Errorf("the menu does not offer Resume for an exam in flight:\n%s", withExam.View())
	}
}

func TestResumingReturnsToTheExamInProgress(t *testing.T) {
	pools := map[int][]string{1: {"ft_strlen"}, 2: {"ft_atoi"}, 3: {"epur_str"}, 4: {"ft_split"}}
	exam, err := session.NewExam(store.DefaultConfig(), pools, rand.New(rand.NewSource(1)), time.Now())
	if err != nil {
		t.Fatalf("NewExam() error = %v", err)
	}
	exam.Record(true, time.Now()) // already cleared level 1
	state, _ := store.Load(t.TempDir())
	m := New(Deps{
		Catalog: testCatalog(t), Grader: grader.New(), Config: store.DefaultConfig(),
		State: state, StateDir: t.TempDir(), Root: t.TempDir(), ResumeExam: exam,
	})
	next, _ := m.Update(key("c"))
	view := next.View()
	if !strings.Contains(view, "Level 2") {
		t.Errorf("resuming did not return to level 2:\n%s", view)
	}
	if !strings.Contains(view, "25 / 100") {
		t.Errorf("resuming lost the score:\n%s", view)
	}
}
```

The test file now needs `math/rand` and `exam02/internal/session` in its imports.

- [ ] **Step 6: Wire resume through the model**

In `internal/ui/model.go`, add the field to `Deps`:

```go
	// ResumeExam is an exam recovered from saved state, or nil when there is
	// none in flight.
	ResumeExam *session.Exam
```

Make `New` adopt it:

```go
// New builds the initial model, showing the menu.
func New(deps Deps) Model {
	return Model{deps: deps, screen: screenMenu, now: time.Now(), exam: deps.ResumeExam}
}
```

Add a method that persists the exam after every change, so a crash loses at
most the current attempt:

```go
// persistExam writes the in-flight exam into saved state. A finished exam is
// cleared instead, so the menu does not offer to resume a run that is over.
func (m *Model) persistExam() {
	if m.exam == nil || m.exam.Over(time.Now()) {
		m.deps.State.Exam = nil
	} else {
		raw, err := session.Encode(m.exam)
		if err != nil {
			m.err = err
			return
		}
		m.deps.State.Exam = raw
	}
	if err := m.deps.State.Save(m.deps.StateDir); err != nil {
		m.err = err
	}
}
```

Call it at the end of `startExam` (after `loadCurrent`) and in `applyVerdict`
immediately after `m.exam.Record(...)`.

- [ ] **Step 7: Add the menu entry**

In `internal/ui/menu.go`, handle the key in `updateMenu`:

```go
	case "c":
		if m.exam != nil && !m.exam.Over(time.Now()) {
			m.screen = screenExam
			return m.loadCurrent()
		}
		return m, nil
```

And render it in `viewMenu`, after the Practice line:

```go
	if m.exam != nil && !m.exam.Over(m.now) {
		b.WriteString(styleKey.Render("c") + "  Resume      " +
			styleWarn.Render(fmt.Sprintf("exam in progress, %s left", clock(m.exam.Remaining(m.now)))) + "\n")
	}
```

- [ ] **Step 8: Run the whole UI suite**

Run: `go test ./internal/ui/ -v`
Expected: PASS, every test including those from Task 11.

- [ ] **Step 9: Commit**

```bash
git add internal/ui/
git commit -m "feat(ui): practice list, result screen, and resuming an exam

The practice list carries each exercise's pass record so the weakest are
visible, and w jumps straight to whichever the record says is weakest.
The result screen states pass or fail against the 100 point mark rather
than only reporting the level reached.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 13: Entry point and toolchain preflight

**Files:**
- Create: `cmd/exam02/main.go`
- Test: `cmd/exam02/main_test.go`

**Interfaces:**
- Consumes: everything.
- Produces: the `exam02` binary.

- [ ] **Step 1: Write the failing test**

Create `cmd/exam02/main_test.go`:

```go
package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestToolchainAdviceIsPlatformSpecific(t *testing.T) {
	advice := toolchainAdvice()
	if advice == "" {
		t.Fatal("toolchainAdvice() is empty; a user with no compiler needs to be told what to do")
	}
	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(advice, "xcode-select") {
			t.Errorf("advice = %q, want it to mention xcode-select on macOS", advice)
		}
	case "linux":
		if !strings.Contains(advice, "build-essential") && !strings.Contains(advice, "gcc") {
			t.Errorf("advice = %q, want it to name a package to install on Linux", advice)
		}
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./cmd/exam02/ -v`
Expected: FAIL — `undefined: toolchainAdvice`.

- [ ] **Step 3: Write the entry point**

Create `cmd/exam02/main.go`:

```go
// Command exam02 is a practice tool for the 42 Common Core Exam Rank 02.
//
// Exercise subjects and the reference solutions used for grading come from
// github.com/alexhiguera/Exam_Rank_02_42_School, MIT licensed,
// Copyright (c) 2026 Alex Higuera.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"exam02/internal/catalog"
	"exam02/internal/grader"
	"exam02/internal/session"
	"exam02/internal/store"
	"exam02/internal/ui"
)

// version is stamped at build time by the release target.
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	root := flag.String("dir", ".", "directory to create rendu/ in")
	flag.Parse()

	if *showVersion {
		fmt.Printf("exam02 %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}

	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, "exam02:", err)
		os.Exit(1)
	}
}

func run(root string) error {
	// Fail here rather than at the first grading attempt: being told there is
	// no compiler before starting a three hour exam is worth a lot more than
	// being told forty minutes in.
	if err := grader.CheckToolchain(); err != nil {
		return fmt.Errorf("%w\n\n%s", err, toolchainAdvice())
	}

	c, err := catalog.Embedded()
	if err != nil {
		return fmt.Errorf("loading exercises: %w", err)
	}

	dir, err := store.Dir()
	if err != nil {
		return err
	}
	cfg, err := store.LoadConfig(dir)
	if err != nil {
		return err
	}
	state, err := store.Load(dir)
	if err != nil {
		return err
	}

	model := ui.New(ui.Deps{
		Catalog:    c,
		Grader:     grader.New(),
		Config:     cfg,
		State:      state,
		StateDir:   dir,
		Root:       root,
		ResumeExam: resumableExam(state, c),
	})

	program := tea.NewProgram(teaModel{model}, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return err
	}
	return state.Save(dir)
}

// teaModel adapts ui.Model, whose Update returns a concrete type for
// testability, to the tea.Model interface.
type teaModel struct{ m ui.Model }

func (t teaModel) Init() tea.Cmd { return t.m.Init() }

func (t teaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := t.m.Update(msg)
	return teaModel{next}, cmd
}

func (t teaModel) View() string { return t.m.View() }

// resumableExam recovers an exam left in flight by an earlier run, or returns
// nil when there is none. A saved exam that no longer decodes is discarded
// rather than reported: it is not worth blocking the application over, and the
// candidate can simply start a new run.
func resumableExam(state *store.State, c *catalog.Catalog) *session.Exam {
	if len(state.Exam) == 0 {
		return nil
	}
	pools := map[int][]string{}
	for level := 1; level <= 4; level++ {
		for _, ex := range c.ByLevel(level) {
			pools[level] = append(pools[level], ex.Name)
		}
	}
	exam, err := session.Decode(state.Exam, pools, rand.New(rand.NewSource(time.Now().UnixNano())))
	if err != nil || exam.Over(time.Now()) {
		return nil
	}
	return exam
}

// toolchainAdvice tells the user how to get a C compiler on their platform.
func toolchainAdvice() string {
	switch runtime.GOOS {
	case "darwin":
		return "Install the Xcode command line tools:\n    xcode-select --install"
	case "linux":
		return "Install a C toolchain, for example:\n" +
			"    sudo apt install build-essential   (Debian, Ubuntu)\n" +
			"    sudo dnf install gcc binutils      (Fedora)"
	default:
		return "Install a C compiler (cc) and binutils (nm)."
	}
}
```

- [ ] **Step 4: Run the tests and build**

```bash
go test ./... 
go build -o /tmp/exam02 ./cmd/exam02
/tmp/exam02 -version
```
Expected: all tests pass; the version line prints.

- [ ] **Step 5: Run it by hand**

```bash
cd /tmp && mkdir -p exam02-try && cd exam02-try && /tmp/exam02
```

Walk through: the menu appears; `2` lists exercises; `enter` opens one and shows its subject; `e` opens the editor on `rendu/<name>/<name>.c`; `g` grades it and shows OK or KO; `q` steps back out; `1` starts an exam whose header counts down.

Then test resume for real, since it is the one behaviour a unit test cannot fully prove: start an exam, pass or fail one exercise, kill the process with Ctrl-C, and run `exam02` again. The menu must offer `c  Resume` with the clock still counting down from where it was, not restarted.

Fix anything that does not behave, with a test first.

- [ ] **Step 6: Commit**

```bash
git add cmd/
git commit -m "feat(cmd): entry point with a toolchain preflight

Checks for cc and nm before the menu appears. Being told there is no
compiler up front is worth far more than discovering it forty minutes
into a three hour exam.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 14: Release pipeline and documentation

**Files:**
- Create: `Makefile`, `README.md`

**Interfaces:**
- Consumes: the built binary from Task 13.
- Produces: `dist/exam02_<os>_<arch>` for four platforms, plus `dist/SHA256SUMS`.

- [ ] **Step 1: Write the Makefile**

```makefile
BINARY  := exam02
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

.PHONY: all build test vet dist clean

all: test build

build:
	go build -ldflags '$(LDFLAGS)' -o $(BINARY) ./cmd/exam02

test:
	go test ./...

vet:
	go vet ./...

# Cross-compiles every supported platform. cgo stays off so the binaries are
# static and the macOS builds need no Mac to produce.
dist: test
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		echo "building $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build -ldflags '$(LDFLAGS)' -o dist/$(BINARY)_$${os}_$${arch} ./cmd/exam02 || exit 1; \
	done
	@cd dist && shasum -a 256 $(BINARY)_* > SHA256SUMS 2>/dev/null || sha256sum $(BINARY)_* > SHA256SUMS
	@ls -lh dist/

clean:
	rm -rf dist $(BINARY)
```

- [ ] **Step 2: Build every platform**

```bash
make dist
file dist/exam02_darwin_arm64
```
Expected: four binaries and a `SHA256SUMS`. `file` reports the darwin build as a Mach-O arm64 executable, proving the cross-compile worked from Linux.

- [ ] **Step 3: Write the README**

Create `README.md`:

````markdown
# exam02

A terminal trainer for the 42 Common Core Exam Rank 02.

It shows you a subject, gives you a workspace, and grades what you write —
including the allowed-functions check that the real Moulinette enforces and
that most informal practice setups skip.

## Install

Download the binary for your platform, make it executable, and put it on your
PATH:

```bash
chmod +x exam02_darwin_arm64
mv exam02_darwin_arm64 /usr/local/bin/exam02
```

The macOS builds are unsigned, so Gatekeeper quarantines them on download.
Clear it once:

```bash
xattr -d com.apple.quarantine /usr/local/bin/exam02
```

You need a C toolchain — `cc` and `nm`. macOS: `xcode-select --install`.
Debian or Ubuntu: `sudo apt install build-essential`.

## Use

```bash
mkdir practice && cd practice
exam02
```

`rendu/` is created in the current directory, exactly as in the real exam.

- **Exam** — three hours, one exercise drawn from each of the four levels.
  25 points each, 100 to pass, so passing means clearing every level. Failing
  an exercise draws another from the same level: it costs you time, not points.
- **Practice** — any exercise, no clock, unlimited attempts, and the reference
  solution available with `r` when you want it.

Keys: `g` grade, `e` edit in `$EDITOR` (`vim` by default), `r` reveal the
solution in practice, `w` drill your weakest exercise, `q` back.

## How grading works

Your file is compiled with `cc -Wall -Wextra -Werror`, so a warning fails you,
as it does in the real exam. Then `nm` is used to check you have not called a
function the subject forbids. Then your code and the reference solution are run
on the same inputs and their output compared.

The reference solutions are the oracle, which is why no expected outputs are
written by hand — and it means a bug shared by your code and the reference will
not be caught. Treat this as practice, not as the Moulinette.

## Build

```bash
make test     # run the suite
make build    # build for this machine
make dist     # cross-compile all four platforms
```

The suite's most important test grades every reference solution against its own
grader. If a driver or a case file is wrong, that test fails.

## Content status

Level 1 is complete and gradable. Levels 2 to 4 ship with their subjects and
reference solutions, and become gradable as their drivers and case files land.

## Credit

Exercise subjects, the Spanish explainers, and the reference solutions come
from [alexhiguera/Exam_Rank_02_42_School](https://github.com/alexhiguera/Exam_Rank_02_42_School),
MIT licensed, Copyright (c) 2026 Alex Higuera. The upstream licence is kept at
`third_party/exam_rank_02/LICENSE`.

This is a practice tool. The real exam runs in a closed environment under the
Moulinette, and its rules are the ones that count.
````

- [ ] **Step 4: Verify the whole thing once more**

```bash
go vet ./...
go test ./...
make dist
```
Expected: clean vet, green tests, four binaries.

- [ ] **Step 5: Commit**

```bash
git add Makefile README.md
git commit -m "build: cross-compile release targets and document the tool

CGO stays off so all four binaries are static and the macOS builds need no
Mac to produce. The README says plainly that the reference solutions are
the grading oracle, so a bug shared with the reference goes uncaught.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Follow-on: Levels 2 to 4

Not part of this plan. It repeats Task 7 for the remaining 44 exercises: confirm `kind` and `expected_file` in each `meta.yaml`, write a driver for every function exercise, write a case file for every exercise, and keep `TestEveryReferenceSolutionPasses` and `TestMutatedReferencesFail` green. No engine change is expected; if one turns out to be needed — the linked-list exercises in Levels 3 and 4 are the likeliest to demand it, since their drivers must build a list from the provided header — that is a signal to stop and design it rather than to bend a driver around it.
