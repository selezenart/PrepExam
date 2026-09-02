package catalog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
)

// ParseCases reads a cases.txt file: one JSON object per line, blank lines
// ignored. JSON rather than shell-like quoting because it needs no custom
// parser and is unambiguous about an empty-string argument versus an absent
// one, a distinction several exercises depend on.
func ParseCases(text string) ([]Case, error) {
	var cases []Case
	scanner := bufio.NewScanner(strings.NewReader(text))
	for line := 1; scanner.Scan(); line++ {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var c Case
		if err := json.Unmarshal([]byte(raw), &c); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		if c.Args == nil {
			c.Args = []string{}
		}
		cases = append(cases, c)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading cases: %w", err)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("no cases: an exercise with no cases would pass anything")
	}
	return cases, nil
}
