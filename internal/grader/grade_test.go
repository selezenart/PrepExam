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

func TestGradeReportsATimeout(t *testing.T) {
	ex := strlenExercise()
	ex.TimeoutMS = 100 // small on purpose: this test must stay fast.
	v := gradeSource(t, ex, "int ft_strlen(char *str){(void)str;while(1);}")
	if v.Status != StatusTimeout {
		t.Errorf("Status = %q, want %q (summary: %s)", v.Status, StatusTimeout, v.Summary)
	}
}

func TestGradeErrorsOnAnUngradableExercise(t *testing.T) {
	// Every level 2-4 exercise is currently in this state: the importer
	// generated it with no cases.txt, and its driver and cases are written
	// by hand in a later task. Grading it must never quietly award a pass.
	ex := strlenExercise()
	ex.Cases = nil
	dir := t.TempDir()
	path := filepath.Join(dir, ex.ExpectedFile)
	if err := os.WriteFile(path, []byte(ex.Reference), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := New().Grade(context.Background(), ex, path)
	if err == nil {
		t.Fatal("Grade() error = nil, want an error for an exercise with no cases")
	}
	if v.Passed() {
		t.Errorf("Grade() returned a passing verdict for an ungradable exercise: %+v", v)
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

func listSizeExercise() catalog.Exercise {
	header := "typedef struct s_list { struct s_list *next; void *data; } t_list;\n"
	return catalog.Exercise{
		Meta: catalog.Meta{
			Name: "ft_list_size", Level: 3, Kind: catalog.KindFunction,
			ExpectedFile: "ft_list_size.c", Header: "ft_list_size.h",
			Prototype: "int ft_list_size(t_list *begin_list);",
		},
		HeaderContent: header,
		Reference: `#include "ft_list_size.h"
int ft_list_size(t_list *l){int n=0;while(l){n++;l=l->next;}return n;}`,
		Driver: `#include <stdio.h>
#include "ft_list_size.h"
int ft_list_size(t_list *begin_list);
int main(int argc, char **argv) {
	t_list nodes[8];
	t_list *head = 0;
	for (int i = argc - 1; i >= 1 && i < 8; i--) {
		nodes[i].data = argv[i]; nodes[i].next = head; head = &nodes[i];
	}
	printf("%d\n", ft_list_size(head));
	return 0;
}`,
		Cases: []catalog.Case{{Args: []string{}}, {Args: []string{"a", "b", "c"}}},
	}
}

// The candidate's directory here has no copy of the header, as when a
// candidate deletes it or submits only the .c file the subject asks for.
// The exercise's own header must still be found, as the Moulinette would.
func TestGradeSuppliesTheExerciseHeader(t *testing.T) {
	ex := listSizeExercise()
	v := gradeSource(t, ex, ex.Reference)
	if !v.Passed() {
		t.Fatalf("the reference failed without a header beside it: %s: %s\n%s", v.Status, v.Summary, v.Detail)
	}
}
