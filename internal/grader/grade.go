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
//
// NM is not a field here: the only place that runs nm is objectSymbols
// (symbols.go), a package-level function that hardcodes "nm" directly and is
// also called straight from symbols_test.go. A Grader.NM field would not
// reach that call without changing objectSymbols' already-committed
// signature, so it would be configuration that silently did nothing.
type Grader struct {
	CC string
}

// New returns a Grader using the system toolchain.
func New() *Grader { return &Grader{CC: "cc"} }

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
	// An exercise with no cases (every level 2-4 exercise, until its driver
	// and cases are authored) has nothing to compare against. Passing one
	// through anyway would award every candidate a free OK, so this is
	// treated as grading not being possible at all — a defect in the
	// exercise's data, never the candidate's fault — rather than as a
	// verdict.
	if !ex.Gradable() {
		return Verdict{}, fmt.Errorf("exercise %s is not gradable yet: missing cases or driver", ex.Name)
	}
	if _, err := os.ReadFile(srcPath); err != nil {
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
