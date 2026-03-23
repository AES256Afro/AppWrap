package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all global key bindings for the TUI.
type KeyMap struct {
	Quit      key.Binding
	Back      key.Binding
	NextFocus key.Binding
	PrevFocus key.Binding
	Enter     key.Binding
	Help      key.Binding

	// Dashboard quick-nav shortcuts.
	NavScan       key.Binding
	NavBuild      key.Binding
	NavRun        key.Binding
	NavInspect    key.Binding
	NavKeygen     key.Binding
	NavProfiles   key.Binding
	NavContainers key.Binding
}

// DefaultKeyMap returns the standard key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		NextFocus: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next"),
		),
		PrevFocus: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),

		NavScan: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "scan"),
		),
		NavBuild: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "build"),
		),
		NavRun: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "run"),
		),
		NavInspect: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "inspect"),
		),
		NavKeygen: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "keygen"),
		),
		NavProfiles: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "profiles"),
		),
		NavContainers: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "containers"),
		),
	}
}
