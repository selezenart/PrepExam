package grader

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"exam02/internal/catalog"
)

// readyExercises returns the exercises that have the driver and cases needed
// to be graded. Levels 2 to 4 arrive in a follow-on plan; until then they are
// present as subjects but not yet gradable, and skipping them here keeps the
// suite honest rather than silently green.
func readyExercises(t *testing.T) []catalog.Exercise {
	t.Helper()
	c, err := catalog.Embedded()
	if err != nil {
		t.Fatalf("catalog.Embedded() error = %v", err)
	}
	var out []catalog.Exercise
	for _, ex := range c.All() {
		if ex.Gradable() {
			out = append(out, ex)
		}
	}
	return out
}

func requireCC(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("cc not available")
	}
}

// TestEveryReferenceSolutionPasses is the load-bearing test of this project.
// It validates every driver and every case file at once: if a driver calls the
// function wrongly or a case file expects something the subject does not
// require, the reference itself fails here.
func TestEveryReferenceSolutionPasses(t *testing.T) {
	requireCC(t)
	exercises := readyExercises(t)
	if len(exercises) < 12 {
		t.Fatalf("only %d exercises are ready; Level 1 has 12", len(exercises))
	}
	g := New()
	for _, ex := range exercises {
		t.Run(ex.Name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, ex.ExpectedFile)
			if err := os.WriteFile(path, []byte(ex.Reference), 0o644); err != nil {
				t.Fatal(err)
			}
			v, err := g.Grade(context.Background(), ex, path)
			if err != nil {
				t.Fatalf("Grade() error = %v", err)
			}
			if !v.Passed() {
				t.Errorf("the reference solution failed its own grader: %s: %s\n%s",
					v.Status, v.Summary, v.Detail)
			}
		})
	}
}

// TestMutatedReferencesFail proves the case files actually discriminate.
// Without it, a driver that printed nothing would sail through the test above
// while testing nothing at all.
func TestMutatedReferencesFail(t *testing.T) {
	requireCC(t)
	g := New()
	for _, ex := range readyExercises(t) {
		mutated, ok := mutate(ex.Reference)
		if r, found := mutationOverrides[ex.Name]; found {
			mutated, ok = strings.Replace(ex.Reference, r.from, r.to, 1), strings.Contains(ex.Reference, r.from)
		}
		if !ok {
			t.Errorf("%s: no mutation could be applied; add a case that would catch one", ex.Name)
			continue
		}
		t.Run(ex.Name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, ex.ExpectedFile)
			if err := os.WriteFile(path, []byte(mutated), 0o644); err != nil {
				t.Fatal(err)
			}
			v, err := g.Grade(context.Background(), ex, path)
			if err != nil {
				t.Fatalf("Grade() error = %v", err)
			}
			if v.Passed() {
				t.Errorf("a mutated reference still passed; the cases do not discriminate")
			}
		})
	}
}

// mutationOverrides replaces mutate's first pick where that pick is an
// equivalent mutant — an edit no input can ever expose, so demanding that a
// case catch it would be demanding the impossible. Each entry says why.
var mutationOverrides = map[string]struct{ from, to string }{
	// The first " <= " sizes the array; at start == end both branches
	// give a length of 1, so flipping it changes nothing observable.
	"ft_range":  {"result[i] = start - i;", "result[i] = start - i - 1;"},
	"ft_rrange": {"result[i] = end + i;", "result[i] = end + i + 1;"},
}

// mutate makes one behaviour-changing edit to C source. It tries each
// replacement in turn and stops at the first that applies.
func mutate(src string) (string, bool) {
	replacements := []struct{ from, to string }{
		{" <= ", " < "},
		{" >= ", " > "},
		{" == ", " != "},
		{"i++", "i += 2"},
		{"len++", "len += 2"},
		// After the counters: " + 1" often sizes a malloc, where growing it
		// is an equivalent mutant no case can ever catch (ft_strdup).
		{" + 1", " + 2"},
		{" = *b;", " = *a;"},
		{" 13", " 12"},
		{" 26", " 25"},
		// Bit-twiddling exercises have none of the operators above.
		{" >> ", " << "},
		{" % 2", " % 3"},
	}
	// Match against a comment-masked copy: the 42 header banner on every
	// reference.c contains dates like "12:58:12", and ft_putstr keeps a
	// commented-out demo main with "argc == 2" in it. Both contain text that
	// coincidentally matches a replacement, and both are dead: editing them
	// changes nothing the compiler sees, so the "mutated" binary behaves
	// identically to the reference and the test above would falsely accept
	// it. Masking comments before searching, then applying the replacement
	// to the same byte range in the real source, guarantees every mutation
	// this function reports as applied actually reaches compiled code.
	masked := maskComments(src)
	for _, r := range replacements {
		if idx := strings.Index(masked, r.from); idx >= 0 {
			return src[:idx] + r.to + src[idx+len(r.from):], true
		}
	}
	return "", false
}

// maskComments returns src with the interior of every /* ... */ and //...
// comment overwritten with spaces (newlines preserved), so byte offsets
// found in the result still address the same bytes in src.
//
// It has no awareness of string or character literals, so a reference that
// embedded "/*" or "//" inside a string literal would be mis-treated as
// starting a comment there. None of the 56 vendored reference.c files do
// that, so literal-aware parsing is not built here.
func maskComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	inBlock, inLine := false, false
	for i := 0; i < len(src); i++ {
		switch {
		case inLine && src[i] == '\n':
			inLine = false
			b.WriteByte('\n')
		case inLine:
			b.WriteByte(' ')
		case inBlock && i+1 < len(src) && src[i] == '*' && src[i+1] == '/':
			inBlock = false
			b.WriteString("  ")
			i++
		case inBlock && src[i] == '\n':
			b.WriteByte('\n')
		case inBlock:
			b.WriteByte(' ')
		case i+1 < len(src) && src[i] == '/' && src[i+1] == '*':
			inBlock = true
			b.WriteString("  ")
			i++
		case i+1 < len(src) && src[i] == '/' && src[i+1] == '/':
			inLine = true
			b.WriteString("  ")
			i++
		default:
			b.WriteByte(src[i])
		}
	}
	return b.String()
}
