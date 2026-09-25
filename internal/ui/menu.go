package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateMenu(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "1":
		// Exam mode needs a gradable exercise at every level. Until the
		// remaining content lands, say so plainly rather than failing.
		if !m.examAvailable() {
			m.status = m.examUnavailableReason()
			return m, nil
		}
		return m.startExam()
	case "2":
		m.practice = m.deps.Catalog.All()
		m.cursor = 0
		m.screen = screenPractice
		return m, nil
	case "3":
		m.screen = screenAbout
		return m, nil
	case "c":
		if m.exam != nil && !m.exam.Over(time.Now()) {
			m.screen = screenExam
			return m.loadCurrent()
		}
		return m, nil
	}
	return m, nil
}

func (m Model) viewMenu() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Exam Rank 02 — practice") + "\n\n")

	total, passed := 0, 0
	for _, ex := range m.deps.Catalog.All() {
		total++
		if st, ok := m.deps.State.Exercises[ex.Name]; ok && st.Passes > 0 {
			passed++
		}
	}
	b.WriteString(styleDim.Render(fmt.Sprintf("%d of %d exercises passed at least once", passed, total)) + "\n\n")

	if m.examAvailable() {
		b.WriteString(styleKey.Render("1") + "  Exam        three hours, one exercise per level, 100 points to pass\n")
	} else {
		b.WriteString(styleDim.Render("1  Exam        ") +
			styleWarn.Render(m.examUnavailableReason()) + "\n")
	}
	b.WriteString(styleKey.Render("2") + "  Practice    pick any exercise, no clock, solutions available\n")
	if m.exam != nil && !m.exam.Over(m.now) {
		b.WriteString(styleKey.Render("c") + "  Resume      " +
			styleWarn.Render(fmt.Sprintf("exam in progress, %s left", clock(m.exam.Remaining(m.now)))) + "\n")
	}
	b.WriteString(styleKey.Render("3") + "  About\n")
	b.WriteString(styleKey.Render("q") + "  Quit\n")

	if m.err != nil {
		b.WriteString("\n" + styleKO.Render("error: "+m.err.Error()) + "\n")
	}
	return b.String()
}

func (m Model) viewAbout() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("About") + "\n\n")
	b.WriteString("A practice tool for the 42 Common Core Exam Rank 02.\n\n")
	b.WriteString("Exercise subjects and the reference solutions used for grading come from\n")
	b.WriteString(styleDim.Render("github.com/alexhiguera/Exam_Rank_02_42_School") + "\n")
	b.WriteString("MIT licensed, Copyright (c) 2026 Alex Higuera.\n\n")
	b.WriteString(styleWarn.Render("This is practice. The real exam is graded by the Moulinette,") + "\n")
	b.WriteString(styleWarn.Render("in a closed environment, and its rules are the ones that count.") + "\n\n")
	b.WriteString(styleDim.Render("any key to go back"))
	return b.String()
}
