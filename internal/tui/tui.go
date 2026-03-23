package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/theencryptedafro/appwrap/internal/service"
)

// Run starts the TUI application.
func Run(svc *service.AppService) error {
	m := newApp(svc)
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.program = p
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
