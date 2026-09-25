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

func TestLoadRejectsFunctionExerciseWithCasesButNoDriver(t *testing.T) {
	// Cases without a driver is a half-finished author's mistake, and must
	// not pass silently.
	fs := testFS()
	delete(fs, "exercises/ft_strlen/driver.c")
	if _, err := Load(fs); err == nil {
		t.Fatal("Load() error = nil, want an error for a function exercise with cases but no driver")
	}
}

func TestLoadAcceptsAnUnauthoredExerciseAsNonGradable(t *testing.T) {
	// The importer generates all 56 exercise directories with neither a
	// driver nor cases. Those are not yet authored, not corrupt: they must
	// load so their subjects can be read, and simply not be gradable.
	fs := testFS()
	delete(fs, "exercises/ft_strlen/driver.c")
	delete(fs, "exercises/ft_strlen/cases.txt")
	c, err := Load(fs)
	if err != nil {
		t.Fatalf("Load() error = %v, want an unauthored exercise to load", err)
	}
	ex, ok := c.ByName("ft_strlen")
	if !ok {
		t.Fatal("ByName(ft_strlen) not found")
	}
	if ex.Gradable() {
		t.Error("Gradable() = true for an exercise with no cases, want false")
	}
	if ex.Subject == "" {
		t.Error("Subject is empty; an unauthored exercise must still be readable")
	}
}

func TestGradableByLevelExcludesUnauthored(t *testing.T) {
	fs := testFS()
	delete(fs, "exercises/ft_strlen/driver.c")
	delete(fs, "exercises/ft_strlen/cases.txt")
	c, err := Load(fs)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if n := len(c.ByLevel(1)); n != 2 {
		t.Errorf("ByLevel(1) = %d, want 2: practice lists everything", n)
	}
	got := c.GradableByLevel(1)
	if len(got) != 1 || got[0].Name != "rot_13" {
		t.Errorf("GradableByLevel(1) = %v, want only rot_13", got)
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

func TestEveryLevelHasGradableExercises(t *testing.T) {
	// An exam draws one exercise per level. A level with nothing gradable
	// would make exam mode refuse to start.
	c, err := Embedded()
	if err != nil {
		t.Fatalf("Embedded() error = %v", err)
	}
	for level := 1; level <= 4; level++ {
		if n := len(c.GradableByLevel(level)); n == 0 {
			t.Errorf("level %d has no gradable exercises; exam mode cannot start", level)
		}
	}
	if n := len(c.All()); n != 56 {
		t.Errorf("All() = %d exercises, want 56", n)
	}
}
