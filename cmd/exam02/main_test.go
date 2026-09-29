package main

import (
	"path/filepath"
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

func TestTheWorkspaceRootIsMadeAbsolute(t *testing.T) {
	root, err := resolveRoot(".")
	if err != nil {
		t.Fatalf("resolveRoot(.) error = %v", err)
	}
	if !filepath.IsAbs(root) {
		t.Errorf("resolveRoot(.) = %q, want an absolute path so the screen and saved state say exactly where rendu/ is", root)
	}
}
