package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updatePractice(msg tea.KeyMsg) (Model, tea.Cmd) {
	// A message on the list answers the key before; the next key moves on.
	m.status = ""
	m.err = nil
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

	case "m":
		if len(m.practice) == 0 {
			return m, nil
		}
		m = m.toggleUnsure(m.practice[m.cursor].Name)
		return m.refreshPractice(), nil

	case "f":
		m.unsureOnly = !m.unsureOnly
		m.cursor = 0
		return m.refreshPractice(), nil

	case "u":
		// Retry whichever marked exercise has waited longest. Drawn from the
		// whole catalog, so it works with the filter on or off.
		var names []string
		for _, ex := range m.deps.Catalog.All() {
			names = append(names, ex.Name)
		}
		ex, ok := m.deps.Catalog.ByName(m.deps.State.OldestUnsure(names))
		if !ok {
			m.status = "nothing is marked unsure: press m on an exercise to mark it"
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
	title := "Practice"
	if m.unsureOnly {
		title = "Practice — marked unsure"
	}
	position := 0
	if len(m.practice) > 0 {
		position = m.cursor + 1
	}
	b.WriteString(styleTitle.Render(title) + "  " +
		styleDim.Render(fmt.Sprintf("%d/%d", position, len(m.practice))) + "\n\n")

	if len(m.practice) == 0 && m.unsureOnly {
		b.WriteString(styleDim.Render("no exercises marked unsure — press f to show all, then m on one to mark it") + "\n")
	}

	first, last := m.practiceWindow()
	for i := first; i < last; i++ {
		ex := m.practice[i]
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
		unsure := "  "
		if m.deps.State.IsUnsure(ex.Name) {
			unsure = styleWarn.Render("? ")
		}
		b.WriteString(marker + unsure + subjectTitle(ex) + "  " + record + "\n")
	}

	if m.status != "" {
		b.WriteString("\n" + styleWarn.Render(m.status) + "\n")
	}
	if m.err != nil {
		b.WriteString("\n" + styleKO.Render("error: "+m.err.Error()) + "\n")
	}

	filter := " only unsure"
	if m.unsureOnly {
		filter = " show all"
	}
	b.WriteString("\n" + styleDim.Render(
		styleKey.Render("enter")+" open   "+
			styleKey.Render("m")+" mark   "+
			styleKey.Render("f")+filter+"   "+
			styleKey.Render("u")+" retry unsure   "+
			styleKey.Render("w")+" drill weakest   "+
			styleKey.Render("q")+" back"))
	return b.String()
}

// practiceWindow is the slice of the list that fits the terminal, positioned
// so the cursor stays on screen. The title and key hints take four lines, and
// a status or error line two more each.
func (m Model) practiceWindow() (first, last int) {
	rows := m.height - 4
	if m.status != "" {
		rows -= 2
	}
	if m.err != nil {
		rows -= 2
	}
	if m.height == 0 || rows >= len(m.practice) {
		return 0, len(m.practice)
	}
	if rows < 1 {
		rows = 1
	}
	first = m.cursor - rows/2
	if first < 0 {
		first = 0
	}
	if first+rows > len(m.practice) {
		first = len(m.practice) - rows
	}
	return first, first + rows
}
