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
	got, err := Run(context.Background(), "/bin/sh", []string{"-c", "kill -SEGV $$"}, "", 2*time.Second)
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
