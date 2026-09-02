// Package catalog holds the exercise data: the upstream subjects, the
// reference solutions used as a grading oracle, and the drivers and input
// cases written for this project.
package catalog

import (
	"bufio"
	"fmt"
	"strings"
)

// SubjectMeta is the machine-readable header of an upstream exercise subject.
type SubjectMeta struct {
	Name             string
	ExpectedFile     string
	AllowedFunctions []string
}

// ParseSubject extracts the header fields from an upstream subject document.
//
// ExpectedFile may come back empty: four subjects write a glob such as
// "*.c, *.h" rather than naming a file, and a glob cannot be resolved without
// knowing the exercise. Those four carry a hand-written expected_file in their
// meta.yaml instead.
func ParseSubject(md string) (SubjectMeta, error) {
	var m SubjectMeta
	scanner := bufio.NewScanner(strings.NewReader(md))
	for scanner.Scan() {
		key, value, ok := splitField(scanner.Text())
		if !ok {
			continue
		}
		switch key {
		case "Assignment name":
			m.Name = value
		case "Expected files":
			m.ExpectedFile = sourceFile(value)
		case "Allowed functions":
			m.AllowedFunctions = parseAllowedFunctions(value)
		}
	}
	if err := scanner.Err(); err != nil {
		return m, fmt.Errorf("reading subject: %w", err)
	}
	if m.Name == "" {
		return m, fmt.Errorf("subject has no assignment name")
	}
	return m, nil
}

// splitField splits "Assignment name  : ft_strlen" into its key and value.
// Subject prose also contains colons, but only the three known keys are acted
// on, so a false split is harmless.
func splitField(line string) (key, value string, ok bool) {
	key, value, ok = strings.Cut(line, ":")
	if !ok {
		return "", "", false
	}
	return strings.TrimSpace(key), strings.TrimSpace(value), true
}

// sourceFile picks the .c file the candidate is expected to write out of an
// "Expected files" value. The value may name one file, name a .c alongside a
// header that is provided rather than written, or be a glob.
func sourceFile(value string) string {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "*") {
			continue
		}
		if strings.HasSuffix(part, ".c") {
			return part
		}
	}
	return ""
}

// parseAllowedFunctions reads the allowed-functions field. Upstream writes an
// empty value, "None", or "-" to mean that no library function may be used.
func parseAllowedFunctions(value string) []string {
	if value == "" || value == "-" || strings.EqualFold(value, "none") {
		return nil
	}
	var out []string
	for _, part := range strings.Split(value, ",") {
		if fn := strings.TrimSpace(part); fn != "" {
			out = append(out, fn)
		}
	}
	return out
}
