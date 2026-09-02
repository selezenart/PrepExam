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
		// Try to wait a tiny bit for the process to complete naturally
		// before killing it. This helps with systems where apport adds
		// delay to crash handling, allowing us to capture the actual signal.
		select {
		case err := <-done:
			result = describe(err)
		case <-time.After(200 * time.Millisecond):
			// Negating the pid addresses the whole process group.
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			<-done
			result = Result{TimedOut: true, ExitCode: -1}
		}
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
