package ui

import "github.com/charmbracelet/lipgloss"

// Colours are chosen from the 256-colour palette so the interface looks the
// same in the terminals 42 machines actually run.
var (
	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("87"))
	styleDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleOK    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("47"))
	styleKO    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))
	styleWarn  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleKey   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
)
