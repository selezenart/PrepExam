// Package workspace prepares the directory a candidate writes their answer in.
//
// The layout mirrors the real exam: rendu/<exercise>/<file>.c under the
// working directory.
package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"exam02/internal/catalog"
)

// Prepare creates the exercise's directory and seeds its source file, and
// returns the path to that file.
//
// An existing file is left exactly as it is. Re-entering an exercise must
// never destroy work, so seeding only ever happens once.
func Prepare(root string, ex catalog.Exercise) (string, error) {
	dir := filepath.Join(root, "rendu", ex.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}

	if ex.Header != "" {
		headerPath := filepath.Join(dir, ex.Header)
		if _, err := os.Stat(headerPath); os.IsNotExist(err) {
			if err := os.WriteFile(headerPath, []byte(ex.HeaderContent), 0o644); err != nil {
				return "", fmt.Errorf("writing %s: %w", ex.Header, err)
			}
		}
	}

	path := filepath.Join(dir, ex.ExpectedFile)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := os.WriteFile(path, []byte(stub(ex)), 0o644); err != nil {
		return "", fmt.Errorf("writing %s: %w", ex.ExpectedFile, err)
	}
	return path, nil
}

// stub is the starting content of a fresh answer file: enough to compile
// against, never enough to be a hint.
func stub(ex catalog.Exercise) string {
	header := fmt.Sprintf("/* %s */\n\n", ex.Name)
	if ex.Kind == catalog.KindProgram {
		return header + "int main(int argc, char **argv)\n{\n\t(void)argc;\n\t(void)argv;\n\treturn (0);\n}\n"
	}
	if ex.Prototype == "" {
		return header
	}
	// Turn the declaration into an empty definition the candidate fills in.
	// The prototype itself is kept as a comment immediately above it, so the
	// exact signature the candidate was given still appears in the file.
	body := strings.TrimSuffix(ex.Prototype, ";")
	return header + "/* " + ex.Prototype + " */\n" + body + "\n{\n}\n"
}

// EditorCommand builds the command that opens path in the candidate's editor.
func EditorCommand(path string) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd
}
