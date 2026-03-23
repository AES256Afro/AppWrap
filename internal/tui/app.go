package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/theencryptedafro/appwrap/internal/service"
)

// ---------- custom messages ----------

// PushViewMsg tells the app to push a new view onto the stack.
type PushViewMsg struct{ View View }

// PopViewMsg tells the app to pop the current view and go back.
type PopViewMsg struct{}

// EventMsg wraps a service.Event for delivery into the bubbletea loop.
type EventMsg service.Event

// ---------- View interface ----------

// View is the abstraction every screen implements.
type View interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (View, tea.Cmd)
	View() string
	Title() string
	// SetSize is called when the terminal is resized.
	SetSize(width, height int)
}

// ---------- app model ----------

type appModel struct {
	svc     *service.AppService
	views   []View
	width   int
	height  int
	keys    KeyMap
	program *tea.Program
}

func newApp(svc *service.AppService) appModel {
	return appModel{
		svc:  svc,
		keys: DefaultKeyMap(),
	}
}

// SetProgram stores the program reference so views can send messages from goroutines.
func (m *appModel) SetProgram(p *tea.Program) {
	m.program = p
}

func (m appModel) Init() tea.Cmd {
	// Will be replaced once the program starts; we lazily push the dashboard in Update
	// on the first WindowSizeMsg.
	return nil
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Push the dashboard on first resize (when stack is empty).
		if len(m.views) == 0 {
			dash := newDashboardView(m.svc, m.width, m.contentHeight())
			m.views = append(m.views, dash)
			return m, dash.Init()
		}
		for _, v := range m.views {
			v.SetSize(m.width, m.contentHeight())
		}
		return m, nil

	case PushViewMsg:
		msg.View.SetSize(m.width, m.contentHeight())
		m.views = append(m.views, msg.View)
		return m, msg.View.Init()

	case PopViewMsg:
		if len(m.views) > 1 {
			m.views = m.views[:len(m.views)-1]
		}
		return m, nil

	case tea.KeyMsg:
		// Global quit always works.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	// Delegate to the top view.
	if len(m.views) > 0 {
		top := m.views[len(m.views)-1]
		updated, cmd := top.Update(msg)
		m.views[len(m.views)-1] = updated
		return m, cmd
	}
	return m, nil
}

func (m appModel) View() string {
	if len(m.views) == 0 {
		return "Loading..."
	}

	top := m.views[len(m.views)-1]

	// Header: breadcrumb / title bar.
	header := m.renderHeader(top.Title())

	// Body: the current view.
	body := top.View()

	// Footer: key hints.
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m appModel) renderHeader(title string) string {
	titleRendered := TitleStyle.Render(" AppWrap ")
	separator := SubtitleStyle.Render(" > ")
	breadcrumb := titleRendered + separator + HeadingStyle.Render(title)

	dockerStatus := SuccessStyle.Render(" Docker: connected ")
	if !m.svc.DockerAvailable() {
		dockerStatus = ErrorStyle.Render(" Docker: disconnected ")
	}

	gap := m.width - lipgloss.Width(breadcrumb) - lipgloss.Width(dockerStatus)
	if gap < 0 {
		gap = 0
	}
	padding := lipgloss.NewStyle().Width(gap).Render("")

	bar := StatusBarStyle.Width(m.width).Render(breadcrumb + padding + dockerStatus)
	return bar
}

func (m appModel) renderFooter() string {
	var hints string
	if len(m.views) > 1 {
		hints = KeyStyle.Render("esc") + KeyHintStyle.Render(" back  ")
	}
	hints += KeyStyle.Render("q") + KeyHintStyle.Render(" quit  ")
	hints += KeyStyle.Render("?") + KeyHintStyle.Render(" help")
	return StatusBarStyle.Width(m.width).Render(hints)
}

// contentHeight returns the vertical space available for the view body.
// Subtracts 2 for header and footer lines.
func (m appModel) contentHeight() int {
	h := m.height - 2
	if h < 1 {
		h = 1
	}
	return h
}
