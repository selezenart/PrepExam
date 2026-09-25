package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"exam02/internal/catalog"
	"exam02/internal/grader"
	"exam02/internal/workspace"
)

// prepareWorkspace is a variable so tests can avoid touching the filesystem.
var prepareWorkspace = workspace.Prepare

func (m Model) updateExercise(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "q", "esc":
		// Leaving an exam does not end it: the clock keeps running and the
		// menu offers to resume, as it would after closing the program.
		if m.screen == screenExam {
			m.screen = screenMenu
			return m, nil
		}
		m.screen = screenPractice
		return m, nil

	case "g":
		m.grading = true
		m.status = "compiling…"
		m.err = nil
		return m, m.gradeCmd()

	case "e":
		cmd := workspace.EditorCommand(m.srcPath)
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return editorFinishedMsg{err: err}
		})

	case "r":
		if m.screen == screenExercise {
			m.revealed = !m.revealed
		}
		return m, nil

	case "x":
		if m.screen == screenExercise && m.current.Spanish != "" {
			m.spanish = !m.spanish
		}
		return m, nil
	}
	return m, nil
}

// gradeCmd runs the grader off the interface goroutine so the clock keeps
// ticking while a compile is in flight.
func (m Model) gradeCmd() tea.Cmd {
	ex, path, g := m.current, m.srcPath, m.deps.Grader
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		v, err := g.Grade(ctx, ex, path)
		return gradedMsg{verdict: v, err: err}
	}
}

// applyVerdict records the outcome and moves the exam on.
func (m Model) applyVerdict(msg gradedMsg) (Model, tea.Cmd) {
	m.grading = false
	m.status = ""
	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}
	v := msg.verdict
	m.verdict = &v

	m.deps.State.Record(m.current.Name, string(v.Status), v.Passed())
	if err := m.deps.State.Save(m.deps.StateDir); err != nil {
		m.err = err
	}

	if m.screen == screenExam && m.exam != nil {
		m.exam.Record(v.Passed(), time.Now())
		m.notice = ""
		m.persistExam()
		if m.exam.Over(time.Now()) {
			m.screen = screenResult
			return m, nil
		}
		// Either way a new exercise is drawn: the next level on a pass, a
		// different one from the same level on a fail. The verdict for the
		// one just graded stays on screen so the candidate sees why.
		graded := m.current.Name
		next, cmd := m.loadCurrent()
		next.verdict = &v
		next.notice = fmt.Sprintf("last graded: %s", graded)
		return next, cmd
	}
	return m, nil
}

func (m Model) viewExercise() string {
	var b strings.Builder

	if m.screen == screenExam && m.exam != nil {
		remaining := m.exam.Remaining(m.now)
		b.WriteString(styleTitle.Render(fmt.Sprintf(
			"Level %d   %d / %d points   %s left",
			m.exam.Level(), m.exam.Points(), m.deps.Config.PassMark, clock(remaining))))
		if remaining < 10*time.Minute {
			b.WriteString("  " + styleWarn.Render("!"))
		}
		b.WriteString("\n\n")
	} else {
		b.WriteString(styleTitle.Render(fmt.Sprintf("%s   level %d", m.current.Name, m.current.Level)) + "\n\n")
	}

	if m.notice != "" {
		b.WriteString(styleWarn.Render(m.notice) + "\n\n")
	}

	b.WriteString(styleBox.Render(strings.TrimSpace(m.current.Subject)) + "\n\n")
	b.WriteString(styleDim.Render("write your answer in  ") + m.srcPath + "\n")
	if len(m.current.AllowedFunctions) > 0 {
		b.WriteString(styleDim.Render("allowed: ") + strings.Join(m.current.AllowedFunctions, ", ") + "\n")
	} else {
		b.WriteString(styleDim.Render("allowed: nothing") + "\n")
	}
	b.WriteString("\n")

	switch {
	case m.grading:
		b.WriteString(m.status + "\n")
	case m.verdict != nil:
		b.WriteString(renderVerdict(*m.verdict) + "\n")
	}
	if m.err != nil {
		b.WriteString(styleKO.Render("error: "+m.err.Error()) + "\n")
	}

	if m.revealed {
		b.WriteString("\n" + styleWarn.Render("reference solution") + "\n")
		b.WriteString(styleBox.Render(strings.TrimSpace(m.current.Reference)) + "\n")
	}
	if m.spanish {
		b.WriteString("\n" + styleBox.Render(strings.TrimSpace(m.current.Spanish)) + "\n")
	}

	b.WriteString("\n" + m.keyHints())
	return b.String()
}

func (m Model) keyHints() string {
	hints := []string{
		styleKey.Render("g") + " grade",
		styleKey.Render("e") + " edit",
	}
	if m.screen == screenExercise {
		hints = append(hints, styleKey.Render("r")+" solution")
		if m.current.Spanish != "" {
			hints = append(hints, styleKey.Render("x")+" explicación")
		}
	}
	hints = append(hints, styleKey.Render("q")+" back")
	return styleDim.Render(strings.Join(hints, "   "))
}

func renderVerdict(v grader.Verdict) string {
	if v.Passed() {
		return styleOK.Render("OK") + "  " + v.Summary
	}
	out := styleKO.Render("KO") + "  " + v.Summary
	if v.Detail != "" {
		out += "\n\n" + styleBox.Render(strings.TrimSpace(v.Detail))
	}
	return out
}

// clock renders a duration as h:mm:ss.
func clock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Seconds())
	return fmt.Sprintf("%d:%02d:%02d", total/3600, (total/60)%60, total%60)
}

// subjectTitle is used by the practice list.
func subjectTitle(ex catalog.Exercise) string {
	return fmt.Sprintf("%-22s level %d", ex.Name, ex.Level)
}
