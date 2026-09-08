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
	// The null pointer is indirected through a variable rather than written
	// as the literal *(int *)0: clang treats that literal form as a
	// compile-time-constant null dereference and flags it via
	// -Wnull-dereference unconditionally (not gated by -Wall/-Wextra), which
	// -Werror then turns into a compile error instead of the runtime crash
	// this test means to exercise.
	v := gradeSource(t, ex, "int ft_strlen(char *str){(void)str;int *p=(int *)0;return *p;}")
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
