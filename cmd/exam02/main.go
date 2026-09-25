// Command exam02 is a practice tool for the 42 Common Core Exam Rank 02.
//
// Exercise subjects and the reference solutions used for grading come from
// github.com/alexhiguera/Exam_Rank_02_42_School, MIT licensed,
// Copyright (c) 2026 Alex Higuera.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"exam02/internal/catalog"
	"exam02/internal/grader"
	"exam02/internal/session"
	"exam02/internal/store"
	"exam02/internal/ui"
)

// version is stamped at build time by the release target.
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	root := flag.String("dir", ".", "directory to create rendu/ in")
	flag.Parse()

	if *showVersion {
		fmt.Printf("exam02 %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}

	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, "exam02:", err)
		os.Exit(1)
	}
}

func run(root string) error {
	// Fail here rather than at the first grading attempt: being told there is
	// no compiler before starting a three hour exam is worth a lot more than
	// being told forty minutes in.
	if err := grader.CheckToolchain(); err != nil {
		return fmt.Errorf("%w\n\n%s", err, toolchainAdvice())
	}

	c, err := catalog.Embedded()
	if err != nil {
		return fmt.Errorf("loading exercises: %w", err)
	}

	dir, err := store.Dir()
	if err != nil {
		return err
	}
	cfg, err := store.LoadConfig(dir)
	if err != nil {
		return err
	}
	state, err := store.Load(dir)
	if err != nil {
		return err
	}

	model := ui.New(ui.Deps{
		Catalog:    c,
		Grader:     grader.New(),
		Config:     cfg,
		State:      state,
		StateDir:   dir,
		Root:       root,
		ResumeExam: resumableExam(state, c),
	})

	program := tea.NewProgram(teaModel{model}, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return err
	}
	return state.Save(dir)
}

// teaModel adapts ui.Model, whose Update returns a concrete type for
// testability, to the tea.Model interface.
type teaModel struct{ m ui.Model }

func (t teaModel) Init() tea.Cmd { return t.m.Init() }

func (t teaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := t.m.Update(msg)
	return teaModel{next}, cmd
}

func (t teaModel) View() string { return t.m.View() }

// resumableExam recovers an exam left in flight by an earlier run, or returns
// nil when there is none. A saved exam that no longer decodes is discarded
// rather than reported: it is not worth blocking the application over, and the
// candidate can simply start a new run.
func resumableExam(state *store.State, c *catalog.Catalog) *session.Exam {
	if len(state.Exam) == 0 {
		return nil
	}
	pools := map[int][]string{}
	for level := 1; level <= 4; level++ {
		for _, ex := range c.GradableByLevel(level) {
			pools[level] = append(pools[level], ex.Name)
		}
	}
	exam, err := session.Decode(state.Exam, pools, rand.New(rand.NewSource(time.Now().UnixNano())))
	if err != nil || exam.Over(time.Now()) {
		return nil
	}
	return exam
}

// toolchainAdvice tells the user how to get a C compiler on their platform.
func toolchainAdvice() string {
	switch runtime.GOOS {
	case "darwin":
		return "Install the Xcode command line tools:\n    xcode-select --install"
	case "linux":
		return "Install a C toolchain, for example:\n" +
			"    sudo apt install build-essential   (Debian, Ubuntu)\n" +
			"    sudo dnf install gcc binutils      (Fedora)"
	default:
		return "Install a C compiler (cc) and binutils (nm)."
	}
}
