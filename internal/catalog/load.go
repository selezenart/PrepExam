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

	// A driver is required only once cases exist; see the cases.txt handling
	// below for why an exercise may legitimately have neither yet.
	if b, err := fs.ReadFile(fsys, dir+"/driver.c"); err == nil {
		ex.Driver = string(b)
	}

	if ex.Header != "" {
		b, err := fs.ReadFile(fsys, dir+"/"+ex.Header)
		if err != nil {
			return ex, fmt.Errorf("reading declared header %s: %w", ex.Header, err)
		}
		ex.HeaderContent = string(b)
	}

	// An exercise with no cases.txt is not yet authored: the importer creates
	// every exercise's subject and reference, and drivers and cases follow by
	// hand. Such an exercise loads so its subject can be read, and reports
	// itself non-gradable. A cases.txt that exists but does not parse is a
	// real mistake and still fails.
	caseBytes, err := fs.ReadFile(fsys, dir+"/cases.txt")
	if err != nil {
		return ex, nil
	}
	if ex.Cases, err = ParseCases(string(caseBytes)); err != nil {
		return ex, fmt.Errorf("parsing cases.txt: %w", err)
	}
	if ex.Kind == KindFunction && ex.Driver == "" {
		return ex, fmt.Errorf("has cases but no driver.c, which a function exercise needs")
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

// GradableByLevel returns the exercises of one level that can actually be
// graded. Exams draw from these: drawing an exercise that cannot be graded
// would strand the candidate on a level they have no way to clear.
func (c *Catalog) GradableByLevel(level int) []Exercise {
	var out []Exercise
	for _, ex := range c.ByLevel(level) {
		if ex.Gradable() {
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
