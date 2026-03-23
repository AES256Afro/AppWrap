package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/theencryptedafro/appwrap/internal/service"
)

// appSelectedMsg is sent when the user picks an app from the list.
type appSelectedMsg struct {
	exePath string
}

// appsLoadedMsg is sent when the background scan finishes.
type appsLoadedMsg struct {
	apps []service.InstalledApp
	err  error
}

type appPickerView struct {
	svc    *service.AppService
	keys   KeyMap
	width  int
	height int

	loading  bool
	spinner  spinner.Model
	filter   textinput.Model
	apps     []service.InstalledApp // all apps
	filtered []service.InstalledApp // after filter
	cursor   int
	err      error
}

func newAppPickerView(svc *service.AppService) *appPickerView {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.Width = 50

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	return &appPickerView{
		svc:     svc,
		keys:    DefaultKeyMap(),
		loading: true,
		spinner: sp,
		filter:  ti,
	}
}

func (v *appPickerView) Title() string { return "Browse Installed Apps" }

func (v *appPickerView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.filter.Width = w - 10
	if v.filter.Width < 20 {
		v.filter.Width = 20
	}
}

func (v *appPickerView) Init() tea.Cmd {
	return tea.Batch(
		v.spinner.Tick,
		v.loadApps(),
	)
}

func (v *appPickerView) loadApps() tea.Cmd {
	return func() tea.Msg {
		apps, err := v.svc.ListInstalledApps()
		return appsLoadedMsg{apps: apps, err: err}
	}
}

func (v *appPickerView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case appsLoadedMsg:
		v.loading = false
		v.apps = msg.apps
		v.err = msg.err
		v.applyFilter()
		return v, nil

	case spinner.TickMsg:
		if v.loading {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			return v, cmd
		}
		return v, nil

	case tea.KeyMsg:
		if v.loading {
			if key.Matches(msg, v.keys.Back) {
				return v, func() tea.Msg { return PopViewMsg{} }
			}
			return v, nil
		}

		switch {
		case key.Matches(msg, v.keys.Back):
			return v, func() tea.Msg { return PopViewMsg{} }

		case key.Matches(msg, v.keys.Enter):
			if len(v.filtered) > 0 && v.cursor < len(v.filtered) {
				app := v.filtered[v.cursor]
				if app.ExePath != "" {
					return v, func() tea.Msg {
						return appSelectedMsg{exePath: app.ExePath}
					}
				}
			}
			return v, nil

		case msg.String() == "up" || msg.String() == "k":
			if v.cursor > 0 {
				v.cursor--
			}
			return v, nil

		case msg.String() == "down" || msg.String() == "j":
			if v.cursor < len(v.filtered)-1 {
				v.cursor++
			}
			return v, nil

		case msg.String() == "pgup":
			v.cursor -= 10
			if v.cursor < 0 {
				v.cursor = 0
			}
			return v, nil

		case msg.String() == "pgdown":
			v.cursor += 10
			if v.cursor >= len(v.filtered) {
				v.cursor = len(v.filtered) - 1
			}
			if v.cursor < 0 {
				v.cursor = 0
			}
			return v, nil

		default:
			// Update filter input
			oldVal := v.filter.Value()
			var cmd tea.Cmd
			v.filter, cmd = v.filter.Update(msg)
			if v.filter.Value() != oldVal {
				v.applyFilter()
				v.cursor = 0
			}
			return v, cmd
		}
	}

	return v, nil
}

func (v *appPickerView) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(v.filter.Value()))
	if q == "" {
		v.filtered = v.apps
		return
	}

	var result []service.InstalledApp
	for _, app := range v.apps {
		name := strings.ToLower(app.Name)
		pub := strings.ToLower(app.Publisher)
		if strings.Contains(name, q) || strings.Contains(pub, q) {
			result = append(result, app)
		}
	}
	v.filtered = result
}

func (v *appPickerView) View() string {
	var b strings.Builder

	b.WriteString("\n")

	if v.loading {
		b.WriteString(fmt.Sprintf("  %s Scanning installed applications...\n", v.spinner.View()))
		b.WriteString(SubtitleStyle.Render("  Checking registry, Start Menu, and more...\n"))
		return lipgloss.NewStyle().Width(v.width).Height(v.height).Render(b.String())
	}

	if v.err != nil {
		b.WriteString(ErrorStyle.Render("  Error discovering apps: "+v.err.Error()) + "\n")
		b.WriteString(KeyHintStyle.Render("  esc: back"))
		return lipgloss.NewStyle().Width(v.width).Height(v.height).Render(b.String())
	}

	// Filter input
	b.WriteString("  " + InfoStyle.Render("Filter: ") + v.filter.View() + "\n")
	b.WriteString(SubtitleStyle.Render(fmt.Sprintf("  %d apps found (%d shown)", len(v.apps), len(v.filtered))) + "\n\n")

	// Table header
	header := fmt.Sprintf("  %-35s %-22s %-12s %-6s", "Name", "Publisher", "Version", "Source")
	b.WriteString(TableHeaderStyle.Render(header) + "\n")

	if len(v.filtered) == 0 {
		b.WriteString(SubtitleStyle.Render("  No matching applications found.\n"))
	} else {
		// Visible window
		maxRows := v.height - 10
		if maxRows < 5 {
			maxRows = 5
		}
		start := 0
		if v.cursor >= maxRows {
			start = v.cursor - maxRows + 1
		}
		end := start + maxRows
		if end > len(v.filtered) {
			end = len(v.filtered)
		}

		for i := start; i < end; i++ {
			app := v.filtered[i]
			name := truncate(app.Name, 34)
			pub := truncate(app.Publisher, 21)
			ver := truncate(app.Version, 11)
			src := truncate(app.Source, 5)

			line := fmt.Sprintf("  %-35s %-22s %-12s %-6s", name, pub, ver, src)

			if i == v.cursor {
				// Show exe path indicator
				indicator := ""
				if app.ExePath != "" {
					indicator = SuccessStyle.Render(" *")
				} else {
					indicator = WarningStyle.Render(" ?")
				}
				line = SelectedRowStyle.Width(v.width - 6).Render(
					fmt.Sprintf(" %-35s %-22s %-12s %-6s", name, pub, ver, src),
				) + indicator
				line = "  " + line
			}

			b.WriteString(line + "\n")
		}

		// Scroll indicator
		if len(v.filtered) > maxRows {
			b.WriteString(SubtitleStyle.Render(fmt.Sprintf("  [%d/%d] ", v.cursor+1, len(v.filtered))))
			b.WriteString(SubtitleStyle.Render("pgup/pgdn to scroll\n"))
		}
	}

	// Selected app details
	if len(v.filtered) > 0 && v.cursor < len(v.filtered) {
		app := v.filtered[v.cursor]
		b.WriteString("\n")
		b.WriteString(SubtitleStyle.Render("  ───────────────────────────────────\n"))
		if app.ExePath != "" {
			b.WriteString(fmt.Sprintf("  Exe:  %s\n", InfoStyle.Render(app.ExePath)))
		} else {
			b.WriteString(WarningStyle.Render("  No .exe path found — install path only\n"))
			if app.InstallPath != "" {
				b.WriteString(fmt.Sprintf("  Dir:  %s\n", SubtitleStyle.Render(app.InstallPath)))
			}
		}
	}

	b.WriteString("\n")
	hints := KeyStyle.Render("enter") + KeyHintStyle.Render(" select  ")
	hints += KeyStyle.Render("up/down") + KeyHintStyle.Render(" navigate  ")
	hints += KeyStyle.Render("esc") + KeyHintStyle.Render(" back")
	b.WriteString(hints)

	return lipgloss.NewStyle().Width(v.width).Height(v.height).Render(b.String())
}

// truncate is defined in view_setup.go
