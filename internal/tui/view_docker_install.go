package tui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/theencryptedafro/appwrap/internal/service"
)

// Install methods the user can choose from.
type installMethod struct {
	key   string
	label string
	desc  string
	cmd   string   // executable
	args  []string // arguments
}

var installMethods = []installMethod{
	{
		key:   "1",
		label: "winget (Recommended)",
		desc:  "Install Docker Desktop via Windows Package Manager",
		cmd:   "winget",
		args:  []string{"install", "-e", "--id", "Docker.DockerDesktop", "--accept-source-agreements", "--accept-package-agreements"},
	},
	{
		key:   "2",
		label: "Chocolatey",
		desc:  "Install Docker Desktop via Chocolatey",
		cmd:   "choco",
		args:  []string{"install", "docker-desktop", "-y"},
	},
	{
		key:   "3",
		label: "Scoop",
		desc:  "Install Docker Desktop via Scoop",
		cmd:   "scoop",
		args:  []string{"install", "docker"},
	},
}

type dockerInstallStep int

const (
	diStepSelect  dockerInstallStep = iota // Choose install method
	diStepRunning                          // Installing...
	diStepDone                             // Finished (success or error)
)

type dockerInstallView struct {
	svc      *service.AppService
	width    int
	height   int
	step     dockerInstallStep
	cursor   int
	spinner  spinner.Model
	logs     []string
	err      error
	success  bool
	keys     KeyMap
	// Which package managers are available
	available map[int]bool
}

// Messages
type dockerInstallLogMsg string
type dockerInstallDoneMsg struct{ err error }

func newDockerInstallView(svc *service.AppService) *dockerInstallView {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(ColorDockerBlue)

	// Check which package managers exist
	available := make(map[int]bool)
	for i, m := range installMethods {
		if _, err := exec.LookPath(m.cmd); err == nil {
			available[i] = true
		}
	}

	return &dockerInstallView{
		svc:       svc,
		step:      diStepSelect,
		spinner:   sp,
		keys:      DefaultKeyMap(),
		available: available,
	}
}

func (d *dockerInstallView) Title() string { return "Install Docker" }

func (d *dockerInstallView) SetSize(w, h int) {
	d.width = w
	d.height = h
}

func (d *dockerInstallView) Init() tea.Cmd { return nil }

func (d *dockerInstallView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch d.step {
		case diStepSelect:
			switch {
			case key.Matches(msg, d.keys.Back):
				return d, func() tea.Msg { return PopViewMsg{} }
			case key.Matches(msg, d.keys.Quit):
				return d, tea.Quit
			case msg.String() == "up" || msg.String() == "k":
				d.moveCursorUp()
			case msg.String() == "down" || msg.String() == "j":
				d.moveCursorDown()
			case key.Matches(msg, d.keys.Enter):
				return d, d.startInstall(d.cursor)
			case msg.String() == "1":
				return d, d.startInstall(0)
			case msg.String() == "2":
				return d, d.startInstall(1)
			case msg.String() == "3":
				return d, d.startInstall(2)
			}
		case diStepRunning:
			// Can't cancel easily, just let it run
		case diStepDone:
			switch {
			case key.Matches(msg, d.keys.Back), key.Matches(msg, d.keys.Enter):
				return d, func() tea.Msg { return PopViewMsg{} }
			case key.Matches(msg, d.keys.Quit):
				return d, tea.Quit
			}
		}

	case spinner.TickMsg:
		if d.step == diStepRunning {
			var cmd tea.Cmd
			d.spinner, cmd = d.spinner.Update(msg)
			return d, cmd
		}

	case dockerInstallLogMsg:
		d.logs = append(d.logs, string(msg))
		return d, nil

	case dockerInstallDoneMsg:
		d.step = diStepDone
		if msg.err != nil {
			d.err = msg.err
			d.logs = append(d.logs, fmt.Sprintf("ERROR: %v", msg.err))
		} else {
			d.success = true
			d.logs = append(d.logs, "Docker Desktop installed successfully!")
			d.logs = append(d.logs, "")
			d.logs = append(d.logs, "NOTE: You may need to:")
			d.logs = append(d.logs, "  1. Start Docker Desktop from the Start Menu")
			d.logs = append(d.logs, "  2. Restart your terminal")
			d.logs = append(d.logs, "  3. Restart your computer (if WSL2 was just installed)")
		}
		return d, nil
	}

	return d, nil
}

func (d *dockerInstallView) moveCursorUp() {
	for {
		if d.cursor <= 0 {
			return
		}
		d.cursor--
		if d.available[d.cursor] {
			return
		}
	}
}

func (d *dockerInstallView) moveCursorDown() {
	for {
		if d.cursor >= len(installMethods)-1 {
			return
		}
		d.cursor++
		if d.available[d.cursor] {
			return
		}
	}
}

func (d *dockerInstallView) startInstall(idx int) tea.Cmd {
	if idx < 0 || idx >= len(installMethods) {
		return nil
	}
	if !d.available[idx] {
		d.logs = append(d.logs, fmt.Sprintf("%s is not installed on this system", installMethods[idx].cmd))
		return nil
	}

	method := installMethods[idx]
	d.step = diStepRunning
	d.logs = []string{
		fmt.Sprintf("Installing Docker Desktop via %s...", method.cmd),
		fmt.Sprintf("Running: %s %s", method.cmd, strings.Join(method.args, " ")),
		"",
	}

	return tea.Batch(d.spinner.Tick, d.runInstall(method))
}

func (d *dockerInstallView) runInstall(method installMethod) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command(method.cmd, method.args...)
		output, err := cmd.CombinedOutput()

		// Send log lines
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				// We can't easily send intermediate messages from here,
				// so we'll bundle the output into the done message.
				_ = line
			}
		}

		if err != nil {
			return dockerInstallDoneMsg{err: fmt.Errorf("%s: %w\n%s", method.cmd, err, string(output))}
		}
		return dockerInstallDoneMsg{err: nil}
	}
}

func (d *dockerInstallView) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(HeadingStyle.Render("  Install Docker"))
	b.WriteString("\n\n")

	switch d.step {
	case diStepSelect:
		b.WriteString("  Docker is required to build and run containers.\n")
		b.WriteString("  Select an installation method:\n\n")

		for i, m := range installMethods {
			avail := d.available[i]

			keyTag := KeyStyle.Render(fmt.Sprintf("[%s]", m.key))
			label := m.label
			desc := SubtitleStyle.Render(m.desc)

			if !avail {
				// Dim unavailable options
				keyTag = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("[%s]", m.key))
				label = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(m.label)
				desc = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(m.desc + " (not found)")
			}

			line := fmt.Sprintf("  %s %s  %s", keyTag, label, desc)
			if i == d.cursor && avail {
				line = "  " + SelectedRowStyle.Width(d.width - 6).Render(
					fmt.Sprintf(" [%s] %-28s %s", m.key, m.label, m.desc),
				)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Show none-available warning
		anyAvailable := false
		for _, a := range d.available {
			if a {
				anyAvailable = true
				break
			}
		}
		if !anyAvailable {
			b.WriteString("\n")
			b.WriteString(WarningStyle.Render("  No package managers found!"))
			b.WriteString("\n")
			b.WriteString("  Install one of: winget, chocolatey, or scoop first.\n")
			b.WriteString("  Or download Docker Desktop manually from:\n")
			b.WriteString(lipgloss.NewStyle().Foreground(ColorDockerBlue).Render("  https://www.docker.com/products/docker-desktop/"))
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(SubtitleStyle.Render("  Or download manually: "))
		b.WriteString(lipgloss.NewStyle().Foreground(ColorDockerBlue).Render("https://docker.com/products/docker-desktop/"))
		b.WriteString("\n")

	case diStepRunning:
		b.WriteString(fmt.Sprintf("  %s Installing Docker Desktop...\n\n", d.spinner.View()))
		b.WriteString(d.renderLogs())

	case diStepDone:
		if d.success {
			b.WriteString(SuccessStyle.Render("  Docker Desktop installed successfully!"))
		} else {
			b.WriteString(ErrorStyle.Render("  Installation failed"))
		}
		b.WriteString("\n\n")
		b.WriteString(d.renderLogs())
		b.WriteString("\n")
		b.WriteString(SubtitleStyle.Render("  Press Enter or Esc to go back"))
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(d.width).
		Height(d.height).
		Render(b.String())
}

func (d *dockerInstallView) renderLogs() string {
	var b strings.Builder
	logStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		Width(d.width - 6)

	// Show last N lines that fit
	maxLines := d.height - 12
	if maxLines < 5 {
		maxLines = 5
	}
	start := 0
	if len(d.logs) > maxLines {
		start = len(d.logs) - maxLines
	}

	var lines []string
	for _, l := range d.logs[start:] {
		lines = append(lines, l)
	}
	b.WriteString("  " + logStyle.Render(strings.Join(lines, "\n")))
	b.WriteString("\n")
	return b.String()
}
