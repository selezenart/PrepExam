package ui

import (
	"fmt"
	"strings"
)

func (m Model) viewResult() string {
	var b strings.Builder
	if m.exam == nil {
		return "no exam to report on"
	}

	points, mark := m.exam.Points(), m.deps.Config.PassMark
	headline := styleKO.Render("FAILED")
	if m.exam.Passed() {
		headline = styleOK.Render("PASSED")
	}
	b.WriteString(styleTitle.Render("Exam over") + "   " + headline + "\n\n")
	b.WriteString(fmt.Sprintf("%d / %d points\n", points, mark))
	b.WriteString(fmt.Sprintf("%s remaining on the clock\n\n", clock(m.exam.Remaining(m.now))))

	attempts := m.exam.Attempts()
	if len(attempts) == 0 {
		b.WriteString(styleDim.Render("nothing was submitted") + "\n")
	} else {
		b.WriteString(styleDim.Render("attempts") + "\n")
		for _, a := range attempts {
			outcome := styleKO.Render("KO")
			if a.Passed {
				outcome = styleOK.Render("OK")
			}
			b.WriteString(fmt.Sprintf("  %s  level %d  %s\n", outcome, a.Level, a.Exercise))
		}
	}

	if !m.exam.Passed() {
		b.WriteString("\n" + styleDim.Render(
			"The pass mark is every level cleared. Practice mode has no clock.") + "\n")
	}
	b.WriteString("\n" + styleDim.Render(
		styleKey.Render("q")+" quit   any other key returns to the menu"))
	return b.String()
}
