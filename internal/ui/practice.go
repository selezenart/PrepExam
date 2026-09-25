package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updatePractice(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "q", "esc":
		m.screen = screenMenu
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		if m.cursor < len(m.practice)-1 {
			m.cursor++
		}
		return m, nil

	case "enter":
		if len(m.practice) == 0 {
			return m, nil
		}
		m.current = m.practice[m.cursor]
		m.screen = screenExercise
		return m.loadCurrent()

	case "w":
		// Drill whichever exercise the record says is weakest.
		names := make([]string, 0, len(m.practice))
		for _, ex := range m.practice {
			names = append(names, ex.Name)
		}
		weakest := m.deps.State.Weakest(names)
		ex, ok := m.deps.Catalog.ByName(weakest)
		if !ok {
			return m, nil
		}
		m.current = ex
		m.screen = screenExercise
		return m.loadCurrent()
	}
	return m, nil
}

func (m Model) viewPractice() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Practice") + "\n\n")

	for i, ex := range m.practice {
		marker := "  "
		if i == m.cursor {
			marker = styleKey.Render("> ")
		}
		record := styleDim.Render("     —")
		if st, ok := m.deps.State.Exercises[ex.Name]; ok && st.Attempts > 0 {
			text := fmt.Sprintf("%5s", fmt.Sprintf("%d/%d", st.Passes, st.Attempts))
			if st.Passes > 0 {
				record = styleOK.Render(text)
			} else {
				record = styleKO.Render(text)
			}
		}
		b.WriteString(marker + subjectTitle(ex) + "  " + record + "\n")
	}

	b.WriteString("\n" + styleDim.Render(
		styleKey.Render("enter")+" open   "+
			styleKey.Render("w")+" drill weakest   "+
			styleKey.Render("q")+" back"))
	return b.String()
}
