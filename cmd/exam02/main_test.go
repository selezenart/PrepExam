package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestToolchainAdviceIsPlatformSpecific(t *testing.T) {
	advice := toolchainAdvice()
	if advice == "" {
		t.Fatal("toolchainAdvice() is empty; a user with no compiler needs to be told what to do")
	}
	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(advice, "xcode-select") {
			t.Errorf("advice = %q, want it to mention xcode-select on macOS", advice)
		}
	case "linux":
		if !strings.Contains(advice, "build-essential") && !strings.Contains(advice, "gcc") {
			t.Errorf("advice = %q, want it to name a package to install on Linux", advice)
		}
	}
}
