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
