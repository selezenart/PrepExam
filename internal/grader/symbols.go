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
	undefined, defined = parseNMOutput(string(out))
	return undefined, defined, nil
}

// parseNMOutput reads nm's listing. Each line is an optional address, a
// one-letter type, and a name; type U means the symbol is undefined, so the
// object refers to it without defining it.
//
// macOS nm prefixes every C symbol with an underscore, Linux nm does not, so
// one leading underscore is stripped on darwin to make the two comparable.
func parseNMOutput(out string) (undefined, defined []string) {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		symType, name := fields[len(fields)-2], fields[len(fields)-1]
		if len(symType) != 1 {
			continue
		}
		name = normalizeSymbol(name)
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

// normalizeSymbol strips the leading underscore macOS adds to C symbols.
func normalizeSymbol(name string) string {
	if runtime.GOOS == "darwin" {
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
	permitted := make(map[string]bool, len(allowed))
	for _, fn := range allowed {
		permitted[fn] = true
	}
	var out []string
	for _, sym := range undefined {
		if permitted[sym] || compilerEmitted[sym] {
			continue
		}
		out = append(out, sym)
	}
	return out
}

// DefinesMain reports whether an object defines main. A function exercise
// whose file defines main would fail to link against the driver, and saying
// so plainly beats showing a duplicate-symbol linker error.
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
