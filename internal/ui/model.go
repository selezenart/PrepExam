// Package ui is the terminal interface: a menu, a timed exam, and untimed
// practice, all driven by the grader underneath.
package ui

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"exam02/internal/catalog"
	"exam02/internal/grader"
	"exam02/internal/session"
	"exam02/internal/store"
)

// screen is which view is on show.
type screen int

const (
	screenMenu screen = iota
	screenExam
	screenPractice
	screenExercise
	screenResult
	screenAbout
)

// Deps is everything the interface needs from the rest of the application,
// passed in rather than constructed here so tests can substitute their own.
type Deps struct {
	Catalog  *catalog.Catalog
	Grader   *grader.Grader
	Config   store.Config
	State    *store.State
	StateDir string
	Root     string // where rendu/ is created

	// ResumeExam is an exam recovered from saved state, or nil when there is
	// none in flight.
	ResumeExam *session.Exam
}

// Model is the Bubble Tea model for the whole application.
type Model struct {
	deps Deps

	screen screen

	exam     *session.Exam
	practice []catalog.Exercise
	cursor   int
	// unsureOnly narrows the practice list to exercises marked unsure.
	unsureOnly bool

	current catalog.Exercise
	srcPath string
	verdict *grader.Verdict
	// gradedPath is the file the verdict on screen was for. In an exam a new
	// exercise is drawn straight after grading, so it can differ from srcPath.
	gradedPath string
	grading    bool
	revealed   bool

	status         string
	confirmRestart bool   // the next 1 on the menu abandons the exam in flight
	notice         string // a one-off message shown on the exercise screen
	err            error
	now            time.Time
	width          int
	height         int
}

// tickMsg drives the countdown.
type tickMsg time.Time

// gradedMsg carries a finished grading run back to the interface.
type gradedMsg struct {
	verdict grader.Verdict
	err     error
}

// editorFinishedMsg arrives when the candidate leaves their editor.
type editorFinishedMsg struct{ err error }

// New builds the initial model, showing the menu.
func New(deps Deps) Model {
	m := Model{deps: deps, screen: screenMenu, now: time.Now(), exam: deps.ResumeExam}
	// Decode may have swapped the saved exercise for another because the
	// catalog changed. The candidate's rendu/ file was written for the old
	// one, so they must be told rather than left to discover it.
	if s := m.exam; s != nil && s.Substitution() != nil {
		sub := s.Substitution()
		m.notice = fmt.Sprintf("%s is no longer in the catalog; your exercise is now %s", sub.From, sub.To)
	}
	// Each launch directory gets its own rendu/. Starting from a different
	// one than last time silently hands the candidate fresh stubs, so say so.
	if st := deps.State; st != nil {
		if st.Workspace != "" && st.Workspace != deps.Root {
			m.status = fmt.Sprintf("last time your files were in %s; this time they are in %s",
				filepath.Join(st.Workspace, "rendu"), filepath.Join(deps.Root, "rendu"))
		}
		st.Workspace = deps.Root
	}
	return m
}

// Init starts the clock.
func (m Model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Update routes a message to the active screen.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		if m.screen == screenExam && m.exam != nil && m.exam.Over(m.now) {
			m.screen = screenResult
			m.persistExam()
		}
		return m, tick()

	case gradedMsg:
		return m.applyVerdict(msg)

	case editorFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.grading {
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil // ignore input while a compile is in flight
	}
	switch m.screen {
	case screenMenu:
		return m.updateMenu(msg)
	case screenPractice:
		return m.updatePractice(msg)
	case screenExam, screenExercise:
		return m.updateExercise(msg)
	case screenResult:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		m.screen = screenMenu
		return m, nil
	case screenAbout:
		m.screen = screenMenu
		return m, nil
	}
	return m, nil
}

// View renders the active screen.
func (m Model) View() string {
	switch m.screen {
	case screenMenu:
		return m.viewMenu()
	case screenPractice:
		return m.viewPractice()
	case screenExam, screenExercise:
		return m.viewExercise()
	case screenResult:
		return m.viewResult()
	case screenAbout:
		return m.viewAbout()
	}
	return ""
}

// refreshPractice rebuilds the practice list: every exercise grouped by level
// so the list reads in exam order, or only the ones marked unsure when the
// filter is on. All is sorted by name and the sort is stable, so names stay
// ordered within a level. The cursor is kept in range, since unmarking under
// the filter removes the row it was on.
func (m Model) refreshPractice() Model {
	all := m.deps.Catalog.All()
	sort.SliceStable(all, func(i, j int) bool { return all[i].Level < all[j].Level })
	m.practice = all[:0:0]
	for _, ex := range all {
		if !m.unsureOnly || m.deps.State.IsUnsure(ex.Name) {
			m.practice = append(m.practice, ex)
		}
	}
	if m.cursor >= len(m.practice) {
		m.cursor = len(m.practice) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return m
}

// toggleUnsure flips the unsure mark on name and saves it straight away, so a
// mark made mid-exam survives the program being closed.
func (m Model) toggleUnsure(name string) Model {
	m.deps.State.ToggleUnsure(name)
	if err := m.deps.State.Save(m.deps.StateDir); err != nil {
		m.err = err
	}
	return m
}

// examAvailable reports whether every level has at least one gradable
// exercise. An exam draws one from each, so a single empty level makes the
// mode impossible.
func (m Model) examAvailable() bool {
	for level := 1; level <= 4; level++ {
		if len(m.deps.Catalog.GradableByLevel(level)) == 0 {
			return false
		}
	}
	return true
}

// examUnavailableReason explains, in the user's terms, why exam mode is not
// offered yet and what is available instead.
func (m Model) examUnavailableReason() string {
	var ready []string
	for level := 1; level <= 4; level++ {
		if len(m.deps.Catalog.GradableByLevel(level)) > 0 {
			ready = append(ready, fmt.Sprintf("%d", level))
		}
	}
	if len(ready) == 0 {
		return "no exercises are gradable yet"
	}
	return "needs all four levels; only " + strings.Join(ready, ", ") +
		" ready so far — practice mode works now"
}

// examPools builds the draw pools for an exam from the gradable exercises.
func (m Model) examPools() map[int][]string {
	pools := map[int][]string{}
	for level := 1; level <= 4; level++ {
		for _, ex := range m.deps.Catalog.GradableByLevel(level) {
			pools[level] = append(pools[level], ex.Name)
		}
	}
	return pools
}

// startExam draws the first exercise and switches to the exam screen.
func (m Model) startExam() (Model, tea.Cmd) {
	exam, err := session.NewExam(m.deps.Config, m.examPools(), rand.New(rand.NewSource(time.Now().UnixNano())), time.Now())
	if err != nil {
		m.err = err
		return m, nil
	}
	exam.Root = m.deps.Root
	m.exam = exam
	m.notice = ""
	m.screen = screenExam
	next, cmd := m.loadCurrent()
	next.persistExam()
	return next, cmd
}

// loadCurrent prepares the workspace for whichever exercise is now current.
func (m Model) loadCurrent() (Model, tea.Cmd) {
	name := m.current.Name
	if m.screen == screenExam && m.exam != nil {
		name = m.exam.Current()
	}
	ex, ok := m.deps.Catalog.ByName(name)
	if !ok {
		m.err = fmt.Errorf("exercise %q is not in the catalog", name)
		return m, nil
	}
	m.current = ex
	m.verdict = nil
	m.revealed = false
	m.err = nil
	path, err := prepareWorkspace(m.workspaceRoot(), ex)
	if err != nil {
		m.err = err
		return m, nil
	}
	m.srcPath = path
	// Record the practice directory the moment rendu/ is used, so the next
	// launch can point out a different one even if this run never quits
	// cleanly. An exam's own directory is recorded with the exam instead.
	if st := m.deps.State; st != nil && m.screen != screenExam {
		st.Workspace = m.deps.Root
		if err := st.Save(m.deps.StateDir); err != nil {
			m.err = err
		}
	}
	return m, nil
}

// workspaceRoot is where rendu/ lives for the exercise on screen: the
// directory an exam was started in, or the launch directory for practice.
func (m Model) workspaceRoot() string {
	if m.screen == screenExam && m.exam != nil && m.exam.Root != "" {
		return m.exam.Root
	}
	return m.deps.Root
}

// persistExam writes the in-flight exam into saved state. A finished exam is
// cleared instead, so the menu does not offer to resume a run that is over.
func (m *Model) persistExam() {
	if m.exam == nil || m.exam.Over(time.Now()) {
		m.deps.State.Exam = nil
	} else {
		raw, err := session.Encode(m.exam)
		if err != nil {
			m.err = err
			return
		}
		m.deps.State.Exam = raw
	}
	if err := m.deps.State.Save(m.deps.StateDir); err != nil {
		m.err = err
	}
}
