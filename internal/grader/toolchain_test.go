package grader

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// brokenCC writes a stand-in compiler that fails the way cc does on an Apple
// Silicon Mac when launched from an x86_64 process: it never looks at the
// source file, prints an xcrun loader error, and exits non-zero.
func brokenCC(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cc")
	script := "#!/bin/sh\necho 'xcrun: error: unable to load libxcrun (missing compatible architecture)' >&2\nexit 1\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGradeReportsABrokenCompilerAsAnErrorNotACompileFailure(t *testing.T) {
	requireCC(t)
	ex := strlenExercise()
	dir := t.TempDir()
	path := filepath.Join(dir, ex.ExpectedFile)
	if err := os.WriteFile(path, []byte(ex.Reference), 0o644); err != nil {
		t.Fatal(err)
	}
	g := &Grader{CC: brokenCC(t)}
	v, err := g.Grade(context.Background(), ex, path)
	if err == nil {
		t.Fatalf("Grade() = %s %q, want an error: the compiler never ran, so the candidate's code was not judged", v.Status, v.Summary)
	}
	if !strings.Contains(err.Error(), "xcrun: error") {
		t.Errorf("error = %q, want it to carry the compiler's own message", err)
	}
}

func TestGradeStillBlamesTheCandidateForARealCompileError(t *testing.T) {
	v := gradeSource(t, strlenExercise(), "int ft_strlen(char *str) { return undeclared; }")
	if v.Status != StatusCompileError {
		t.Errorf("status = %s, want %s", v.Status, StatusCompileError)
	}
}

func TestCheckToolchainRunsTheCompiler(t *testing.T) {
	g := &Grader{CC: brokenCC(t)}
	err := g.CheckToolchain(context.Background())
	if err == nil || !strings.Contains(err.Error(), "xcrun: error") {
		t.Errorf("CheckToolchain() = %v, want the compiler's failure, since cc exists but cannot run", err)
	}
}

func TestCheckToolchainAcceptsAWorkingToolchain(t *testing.T) {
	requireCC(t)
	if err := New().CheckToolchain(context.Background()); err != nil {
		t.Errorf("CheckToolchain() = %v, want nil on a machine where cc and nm work", err)
	}
}

func TestNativeArchPrefix(t *testing.T) {
	yes := func() bool { return true }
	no := func() bool { return false }
	cases := []struct {
		goos, goarch string
		translated   func() bool
		want         []string
	}{
		{"darwin", "amd64", yes, []string{"/usr/bin/arch", "-arm64"}},
		{"darwin", "amd64", no, nil},
		{"darwin", "arm64", yes, nil},
		{"linux", "amd64", yes, nil},
	}
	for _, c := range cases {
		got := nativeArchPrefix(c.goos, c.goarch, c.translated)
		if strings.Join(got, " ") != strings.Join(c.want, " ") {
			t.Errorf("nativeArchPrefix(%s, %s) = %v, want %v", c.goos, c.goarch, got, c.want)
		}
	}
}

// The prefix must reach every tool the grader runs, or a translated build
// keeps failing on whichever tool was missed (nm, in the issue's follow-up).
func TestToolsRunThroughTheNativeArchPrefix(t *testing.T) {
	requireCC(t)
	log := filepath.Join(t.TempDir(), "log")
	wrapper := filepath.Join(t.TempDir(), "wrap")
	script := "#!/bin/sh\necho \"$1\" >> " + log + "\nexec \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	saved := toolPrefix
	toolPrefix = []string{wrapper}
	defer func() { toolPrefix = saved }()

	if v := gradeSource(t, strlenExercise(), strlenExercise().Reference); !v.Passed() {
		t.Fatalf("grading through the prefix failed: %s %s", v.Status, v.Summary)
	}
	b, _ := os.ReadFile(log)
	for _, tool := range []string{"cc", "nm"} {
		if !strings.Contains(string(b), tool+"\n") {
			t.Errorf("%s did not run through the prefix; wrapper saw:\n%s", tool, b)
		}
	}
}
