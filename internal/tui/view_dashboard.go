package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/theencryptedafro/appwrap/internal/service"
)

// dashboardAction defines one action tile on the dashboard.
type dashboardAction struct {
	key   string
	label string
	desc  string
}

var dashboardActions = []dashboardAction{
	{key: "S", label: "Scan App", desc: "Discover dependencies"},
	{key: "B", label: "Build Image", desc: "Create Docker image"},
	{key: "R", label: "Run Container", desc: "Start application"},
	{key: "I", label: "Inspect Binary", desc: "PE analysis"},
	{key: "K", label: "Generate Keys", desc: "Age encryption keys"},
	{key: "P", label: "Profiles", desc: "Manage profiles"},
	{key: "C", label: "Containers", desc: "Manage containers"},
	{key: "D", label: "Setup / Deps", desc: "Check & install all dependencies"},
}

// Extra actions shown when Docker is disconnected.
var dockerActions = []dashboardAction{
	{key: "T", label: "Retry Connection", desc: "Check if Docker is now available"},
}

// dockerRetryMsg is sent after a retry connection check.
type dockerRetryMsg struct{ connected bool }

type dashboardView struct {
	svc             *service.AppService
	width           int
	height          int
	cursor          int
	keys            KeyMap
	profiles        []service.ProfileSummary
	dockerConnected bool
	retrying        bool
}

func newDashboardView(svc *service.AppService, width, height int) *dashboardView {
	profiles, _ := svc.ListProfiles("")
	return &dashboardView{
		svc:             svc,
		width:           width,
		height:          height,
		keys:            DefaultKeyMap(),
		profiles:        profiles,
		dockerConnected: svc.DockerAvailable(),
	}
}

// totalActions returns how many menu items are visible.
func (d *dashboardView) totalActions() int {
	n := len(dashboardActions)
	if !d.dockerConnected {
		n += len(dockerActions)
	}
	return n
}

func (d *dashboardView) Title() string { return "Dashboard" }

func (d *dashboardView) SetSize(w, h int) {
	d.width = w
	d.height = h
}

func (d *dashboardView) Init() tea.Cmd { return nil }

func (d *dashboardView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case dockerRetryMsg:
		d.retrying = false
		d.dockerConnected = msg.connected
		return d, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keys.Quit):
			return d, tea.Quit
		case key.Matches(msg, d.keys.NavScan):
			return d, d.pushView(newScanView(d.svc))
		case key.Matches(msg, d.keys.NavBuild):
			return d, d.pushView(newBuildView(d.svc))
		case key.Matches(msg, d.keys.NavRun):
			return d, d.pushView(newRunView(d.svc))
		case key.Matches(msg, d.keys.NavInspect):
			return d, d.pushView(newInspectView(d.svc))
		case key.Matches(msg, d.keys.NavKeygen):
			return d, d.pushView(newKeygenView(d.svc))
		case key.Matches(msg, d.keys.NavProfiles):
			return d, d.pushView(newProfilesView(d.svc))
		case key.Matches(msg, d.keys.NavContainers):
			return d, d.pushView(newContainersView(d.svc))
		case msg.String() == "d" || msg.String() == "D":
			return d, d.pushView(newSetupView(d.svc))
		case msg.String() == "t" || msg.String() == "T":
			if !d.dockerConnected && !d.retrying {
				d.retrying = true
				return d, d.retryDockerConnection()
			}
		case key.Matches(msg, d.keys.Enter):
			return d, d.activateCursor()
		case msg.String() == "up" || msg.String() == "k":
			if d.cursor > 0 {
				d.cursor--
			}
		case msg.String() == "down" || msg.String() == "j":
			if d.cursor < d.totalActions()-1 {
				d.cursor++
			}
		}
	}
	return d, nil
}

func (d *dashboardView) retryDockerConnection() tea.Cmd {
	return func() tea.Msg {
		connected := d.svc.DockerAvailable()
		return dockerRetryMsg{connected: connected}
	}
}

func (d *dashboardView) activateCursor() tea.Cmd {
	switch d.cursor {
	case 0:
		return d.pushView(newScanView(d.svc))
	case 1:
		return d.pushView(newBuildView(d.svc))
	case 2:
		return d.pushView(newRunView(d.svc))
	case 3:
		return d.pushView(newInspectView(d.svc))
	case 4:
		return d.pushView(newKeygenView(d.svc))
	case 5:
		return d.pushView(newProfilesView(d.svc))
	case 6:
		return d.pushView(newContainersView(d.svc))
	case 7:
		return d.pushView(newSetupView(d.svc))
	case 8:
		// Retry connection (only visible when Docker disconnected)
		if !d.dockerConnected && !d.retrying {
			d.retrying = true
			return d.retryDockerConnection()
		}
	}
	return nil
}

func (d *dashboardView) pushView(v View) tea.Cmd {
	return func() tea.Msg { return PushViewMsg{View: v} }
}

func (d *dashboardView) View() string {
	var b strings.Builder

	// Logo.
	b.WriteString(Logo())
	b.WriteString("\n\n")

	// Docker status.
	if d.dockerConnected {
		b.WriteString(SuccessStyle.Render("  Docker: connected"))
	} else {
		status := ErrorStyle.Render("  Docker: disconnected")
		if d.retrying {
			status = WarningStyle.Render("  Docker: checking...")
		}
		b.WriteString(status)
	}
	b.WriteString("\n\n")

	// Quick actions.
	b.WriteString(HeadingStyle.Render("  Quick Actions"))
	b.WriteString("\n\n")

	for i, a := range dashboardActions {
		keyTag := KeyStyle.Render(fmt.Sprintf("[%s]", a.key))
		label := a.label
		desc := SubtitleStyle.Render(a.desc)

		line := fmt.Sprintf("  %s %s  %s", keyTag, label, desc)
		if i == d.cursor {
			line = SelectedRowStyle.Width(d.width - 4).Render(
				fmt.Sprintf(" [%s] %-18s %s", a.key, a.label, a.desc),
			)
			line = "  " + line
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Docker install/retry actions (only when disconnected).
	if !d.dockerConnected {
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Docker Setup"))
		b.WriteString("\n\n")

		for i, a := range dockerActions {
			idx := len(dashboardActions) + i
			keyTag := KeyStyle.Render(fmt.Sprintf("[%s]", a.key))
			label := a.label
			desc := SubtitleStyle.Render(a.desc)

			line := fmt.Sprintf("  %s %s  %s", keyTag, label, desc)
			if idx == d.cursor {
				line = SelectedRowStyle.Width(d.width - 4).Render(
					fmt.Sprintf(" [%s] %-18s %s", a.key, a.label, a.desc),
				)
				line = "  " + line
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Recent profiles.
	if len(d.profiles) > 0 {
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Recent Profiles"))
		b.WriteString("\n\n")
		limit := 5
		if len(d.profiles) < limit {
			limit = len(d.profiles)
		}
		for _, p := range d.profiles[:limit] {
			name := lipgloss.NewStyle().Foreground(ColorDockerBlue).Render(p.AppName)
			strategy := SubtitleStyle.Render(p.Strategy)
			arch := SubtitleStyle.Render(p.Arch)
			b.WriteString(fmt.Sprintf("    %s  %s  %s  (%s)\n", name, strategy, arch, p.Name))
		}
	}

	return lipgloss.NewStyle().
		Width(d.width).
		Height(d.height).
		Render(b.String())
}
