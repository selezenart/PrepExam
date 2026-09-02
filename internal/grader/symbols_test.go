package grader

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
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
	if runtime.GOOS != "darwin" {
		// normalizeSymbol only strips the underscore macOS's nm adds; on
		// every other platform it is a no-op by design, so this darwin path
		// cannot be exercised on this machine without faking runtime.GOOS.
		t.Skip("darwin-only: normalizeSymbol strips the leading underscore only on darwin")
	}
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
