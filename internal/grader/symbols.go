package grader

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
)

// objectSymbols runs nm over an object file and returns its undefined and
// defined symbol names.
func objectSymbols(ctx context.Context, objPath string) (undefined, defined []string, err error) {
	out, err := exec.CommandContext(ctx, "nm", objPath).Output()
	if err != nil {
		return nil, nil, fmt.Errorf("running nm on %s: %w", objPath, err)
	}
	undefined, defined = parseNMOutput(string(out), runtime.GOOS)
	return undefined, defined, nil
}

// parseNMOutput reads nm's listing. Each line is an optional address, a
// one-letter type, and a name; type U means the symbol is undefined, so the
// object refers to it without defining it.
//
// macOS nm prefixes every C symbol with an underscore, Linux nm does not, so
// one leading underscore is stripped when goos is "darwin" to make the two
// comparable. goos is a parameter rather than a read of runtime.GOOS inside
// this function so the darwin path can be exercised deterministically by a
// test running on any platform; objectSymbols, the only production caller,
// always passes the real runtime.GOOS.
func parseNMOutput(out, goos string) (undefined, defined []string) {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		symType, name := fields[len(fields)-2], fields[len(fields)-1]
		if len(symType) != 1 {
			continue
		}
		name = normalizeSymbol(name, goos)
		if symType == "U" {
			undefined = append(undefined, name)
		} else if symType == strings.ToUpper(symType) {
			// An uppercase type means a global definition; lowercase means a
			// file-local one, which no other object can call.
			defined = append(defined, name)
		}
	}
	sort.Strings(undefined)
	sort.Strings(defined)
	return dedupe(undefined), dedupe(defined)
}

// normalizeSymbol strips the leading underscore macOS adds to C symbols,
// when goos is "darwin". It is a no-op on every other platform.
func normalizeSymbol(name, goos string) string {
	if goos == "darwin" {
		return strings.TrimPrefix(name, "_")
	}
	return name
}

// Forbidden returns the undefined symbols that the subject does not allow.
//
// Nothing needs subtracting for functions the candidate defined themselves,
// including a recursive call to the exercise function: those resolve inside
// the translation unit and never appear as undefined.
func Forbidden(undefined, allowed []string) []string {
	return forbidden(undefined, allowed, runtime.GOOS)
}

// forbidden is Forbidden's implementation with goos threaded explicitly
// rather than read from runtime.GOOS, so a test can check what a specific
// platform's compiler-emitted allowlist matches without depending on which
// platform the test binary happens to run on. Forbidden always calls this
// with the real runtime.GOOS, so production behaviour is unchanged.
func forbidden(undefined, allowed []string, goos string) []string {
	permitted := make(map[string]bool, len(allowed))
	for _, fn := range allowed {
		permitted[fn] = true
	}
	emitted := compilerEmittedFor(goos)
	var out []string
	for _, sym := range undefined {
		if permitted[sym] || emitted[sym] {
			continue
		}
		out = append(out, sym)
	}
	return out
}

// DefinesMain reports whether an object defines main. A function exercise
// whose file defines main would fail to link against the driver, and saying
// so plainly beats showing a duplicate-symbol linker error.
//
// This only looks at defined, which parseNMOutput fills from nm's uppercase
// (globally visible) type codes; a `static int main(void)` has nm type "t"
// (file-local) and is deliberately not counted here. No real exercise
// submission declares main static, so that narrower, link-visibility-based
// notion of "defined" is left as-is rather than special-cased.
func DefinesMain(defined []string) bool {
	for _, sym := range defined {
		if sym == "main" {
			return true
		}
	}
	return false
}

func dedupe(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := in[:1]
	for _, s := range in[1:] {
		if s != out[len(out)-1] {
			out = append(out, s)
		}
	}
	return out
}
