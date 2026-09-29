package grader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// toolPrefix is prepended to every toolchain command the grader runs (cc, nm).
// It is empty except when an x86_64 build runs under Rosetta on an Apple
// Silicon Mac: children of a translated process inherit its x86_64
// preference, and the Command Line Tools front-ends in /usr/bin then fail to
// load their arm64-only libxcrun. Running them through `arch -arm64` puts
// them back on the native architecture.
var toolPrefix = nativeArchPrefix(runtime.GOOS, runtime.GOARCH, runningUnderRosetta)

// nativeArchPrefix decides toolPrefix. The platform and the translation check
// are parameters so every combination can be tested on any machine.
func nativeArchPrefix(goos, goarch string, translated func() bool) []string {
	if goos == "darwin" && goarch == "amd64" && translated() {
		return []string{"/usr/bin/arch", "-arm64"}
	}
	return nil
}

// runningUnderRosetta reports whether this process is an x86_64 binary being
// translated on Apple Silicon. The sysctl exists only on macOS 11 and later
// and reads 1 for a translated process.
func runningUnderRosetta() bool {
	out, err := exec.Command("/usr/sbin/sysctl", "-n", "sysctl.proc_translated").Output()
	return err == nil && strings.TrimSpace(string(out)) == "1"
}

// toolCommand builds a toolchain command, routed through toolPrefix.
func toolCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	if len(toolPrefix) > 0 {
		full := append(append(append([]string{}, toolPrefix[1:]...), name), args...)
		return exec.CommandContext(ctx, toolPrefix[0], full...)
	}
	return exec.CommandContext(ctx, name, args...)
}

// CheckToolchain proves the compiler and nm actually work by compiling a
// trivial file and listing its symbols. Finding them on PATH is not enough:
// on an Apple Silicon Mac a mismatched build finds cc fine and then cannot
// run it, and that must surface here, before an exam starts, rather than as
// a compile error blamed on the candidate's code.
func (g *Grader) CheckToolchain(ctx context.Context) error {
	dir, err := os.MkdirTemp("", "exam02-preflight-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	src, obj := filepath.Join(dir, "preflight.c"), filepath.Join(dir, "preflight.o")
	if err := os.WriteFile(src, []byte("int main(void)\n{\n\treturn (0);\n}\n"), 0o644); err != nil {
		return err
	}
	if out, err := g.compile(ctx, src, obj, dir); err != nil {
		return fmt.Errorf("the C compiler (%s) does not work: %s", g.CC, toolFailure(out, err))
	}
	if out, err := toolCommand(ctx, "nm", obj).CombinedOutput(); err != nil {
		return fmt.Errorf("nm does not work: %s", toolFailure(string(out), err))
	}
	return nil
}

// toolFailure describes a failed tool run using its own output when it has
// any, since that is where the real reason is.
func toolFailure(out string, err error) string {
	if s := strings.TrimSpace(out); s != "" {
		return s
	}
	return err.Error()
}

// blamesSource reports whether a failed compile is the source file's fault:
// the compiler ran and emitted diagnostics that name the file. Anything else
// — the compiler not starting, or dying before reading the file — is a
// broken toolchain, and reporting it as the candidate's compile error would
// reject correct code with the wrong reason.
func blamesSource(out string, err error, src string) bool {
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		return false
	}
	return strings.Contains(out, filepath.Base(src)+":")
}
