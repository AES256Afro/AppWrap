package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/theencryptedafro/appwrap/internal/service"
)

// ---------- dependency definitions ----------

type depStatus int

const (
	depUnknown    depStatus = iota
	depChecking             // currently probing
	depInstalled            // found and working
	depMissing              // not found
	depInstalling           // install in progress
	depFailed               // install failed
)

type dependency struct {
	id          string
	name        string
	description string
	required    bool   // true = mandatory, false = optional (feature-gated)
	checkCmd    string // binary to LookPath
	checkArgs   []string
	installCmd  string
	installArgs []string
	status      depStatus
	version     string
	errMsg      string
}

func defaultDeps() []*dependency {
	return []*dependency{
		{
			id:          "wsl",
			name:        "WSL2",
			description: "Windows Subsystem for Linux (required by Docker Desktop)",
			required:    true,
			checkCmd:    "wsl",
			checkArgs:   []string{"--status"},
			installCmd:  "wsl",
			installArgs: []string{"--install", "--no-distribution"},
		},
		{
			id:          "docker",
			name:        "Docker Desktop",
			description: "Container runtime for building and running apps",
			required:    true,
			checkCmd:    "docker",
			checkArgs:   []string{"info"},
			installCmd:  "winget",
			installArgs: []string{"install", "-e", "--id", "Docker.DockerDesktop", "--accept-source-agreements", "--accept-package-agreements"},
		},
		{
			id:          "age",
			name:        "Age Encryption",
			description: "File encryption for secure containers (optional)",
			required:    false,
			checkCmd:    "age",
			checkArgs:   []string{"--version"},
			installCmd:  "winget",
			installArgs: []string{"install", "-e", "--id", "FiloSottile.age", "--accept-source-agreements", "--accept-package-agreements"},
		},
		{
			id:          "wireguard",
			name:        "WireGuard",
			description: "VPN tunnel for container network privacy (optional)",
			required:    false,
			checkCmd:    "wg",
			checkArgs:   []string{"--version"},
			installCmd:  "winget",
			installArgs: []string{"install", "-e", "--id", "WireGuard.WireGuard", "--accept-source-agreements", "--accept-package-agreements"},
		},
	}
}

// ---------- messages ----------

type depCheckResultMsg struct {
	id      string
	status  depStatus
	version string
	errMsg  string
}

type depInstallResultMsg struct {
	id     string
	status depStatus
	errMsg string
}

type allChecksCompleteMsg struct{}

// ---------- setup view ----------

type setupView struct {
	svc         *service.AppService
	width       int
	height      int
	deps        []*dependency
	cursor      int
	spinner     spinner.Model
	keys        KeyMap
	checking    bool
	hasWinget   bool
	allReady    bool
	installLogs map[string][]string
	mu          sync.Mutex
}

func newSetupView(svc *service.AppService) *setupView {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(ColorDockerBlue)

	_, wingetErr := exec.LookPath("winget")

	return &setupView{
		svc:         svc,
		deps:        defaultDeps(),
		spinner:     sp,
		keys:        DefaultKeyMap(),
		hasWinget:   wingetErr == nil,
		installLogs: make(map[string][]string),
	}
}

func (v *setupView) Title() string { return "Setup" }

func (v *setupView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *setupView) Init() tea.Cmd {
	// Start checking all deps immediately
	v.checking = true
	for _, d := range v.deps {
		d.status = depChecking
	}
	return tea.Batch(v.spinner.Tick, v.checkAllDeps())
}

func (v *setupView) checkAllDeps() tea.Cmd {
	var cmds []tea.Cmd
	for _, d := range v.deps {
		dep := d // capture
		cmds = append(cmds, func() tea.Msg {
			// Check if binary exists
			path, err := exec.LookPath(dep.checkCmd)
			if err != nil {
				return depCheckResultMsg{id: dep.id, status: depMissing}
			}

			// Try running the check command for version info
			version := ""
			if len(dep.checkArgs) > 0 {
				out, err := exec.Command(dep.checkCmd, dep.checkArgs...).CombinedOutput()
				if err != nil {
					// Binary exists but check command failed
					// For docker this means daemon isn't running
					if dep.id == "docker" {
						return depCheckResultMsg{
							id:     dep.id,
							status: depMissing,
							errMsg: "Docker is installed but not running. Start Docker Desktop.",
						}
					}
					// For WSL, --status might fail if not installed
					if dep.id == "wsl" {
						return depCheckResultMsg{id: dep.id, status: depMissing}
					}
				}
				// Extract first line as version
				lines := strings.Split(strings.TrimSpace(string(out)), "\n")
				if len(lines) > 0 {
					version = strings.TrimSpace(lines[0])
					// Truncate long version strings
					if len(version) > 60 {
						version = version[:60] + "..."
					}
				}
			}

			_ = path
			return depCheckResultMsg{id: dep.id, status: depInstalled, version: version}
		})
	}
	return tea.Batch(cmds...)
}

func (v *setupView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {

	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd

	case depCheckResultMsg:
		for _, d := range v.deps {
			if d.id == msg.id {
				d.status = msg.status
				d.version = msg.version
				d.errMsg = msg.errMsg
				break
			}
		}
		// Check if all checks are done
		allDone := true
		for _, d := range v.deps {
			if d.status == depChecking {
				allDone = false
				break
			}
		}
		if allDone {
			v.checking = false
			v.updateAllReady()
		}
		return v, nil

	case depInstallResultMsg:
		for _, d := range v.deps {
			if d.id == msg.id {
				d.status = msg.status
				d.errMsg = msg.errMsg
				break
			}
		}
		v.updateAllReady()
		return v, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keys.Back):
			return v, func() tea.Msg { return PopViewMsg{} }
		case key.Matches(msg, v.keys.Quit):
			return v, tea.Quit
		case msg.String() == "up" || msg.String() == "k":
			if v.cursor > 0 {
				v.cursor--
			}
		case msg.String() == "down" || msg.String() == "j":
			if v.cursor < len(v.deps)-1 {
				v.cursor++
			}
		case key.Matches(msg, v.keys.Enter):
			return v, v.installSelected()
		case msg.String() == "i":
			return v, v.installSelected()
		case msg.String() == "a":
			return v, v.installAllMissing()
		case msg.String() == "r":
			// Re-check all
			v.checking = true
			for _, d := range v.deps {
				d.status = depChecking
				d.errMsg = ""
			}
			return v, tea.Batch(v.spinner.Tick, v.checkAllDeps())
		}
	}
	return v, nil
}

func (v *setupView) installSelected() tea.Cmd {
	if v.cursor < 0 || v.cursor >= len(v.deps) {
		return nil
	}
	dep := v.deps[v.cursor]
	if dep.status != depMissing && dep.status != depFailed {
		return nil
	}
	return v.installDep(dep)
}

func (v *setupView) installAllMissing() tea.Cmd {
	var cmds []tea.Cmd
	for _, d := range v.deps {
		if d.status == depMissing || d.status == depFailed {
			cmds = append(cmds, v.installDep(d))
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (v *setupView) installDep(dep *dependency) tea.Cmd {
	if !v.hasWinget && dep.installCmd == "winget" {
		dep.status = depFailed
		dep.errMsg = "winget not available. Install manually."
		return nil
	}

	dep.status = depInstalling
	d := dep // capture
	return tea.Batch(v.spinner.Tick, func() tea.Msg {
		cmd := exec.Command(d.installCmd, d.installArgs...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return depInstallResultMsg{
				id:     d.id,
				status: depFailed,
				errMsg: fmt.Sprintf("%v: %s", err, truncate(string(out), 200)),
			}
		}
		return depInstallResultMsg{id: d.id, status: depInstalled}
	})
}

func (v *setupView) updateAllReady() {
	v.allReady = true
	for _, d := range v.deps {
		if d.required && d.status != depInstalled {
			v.allReady = false
			break
		}
	}
}

func (v *setupView) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(HeadingStyle.Render("  Environment Setup"))
	b.WriteString("\n")
	b.WriteString(SubtitleStyle.Render("  AppWrap checks and installs everything you need.\n"))
	b.WriteString("\n")

	// Winget status
	if !v.hasWinget {
		b.WriteString(WarningStyle.Render("  ⚠ winget not found — auto-install unavailable. Install dependencies manually."))
		b.WriteString("\n\n")
	}

	// Dependency table
	for i, d := range v.deps {
		cursor := "  "
		if i == v.cursor {
			cursor = "> "
		}

		// Status icon
		var icon, statusText string
		switch d.status {
		case depUnknown, depChecking:
			icon = v.spinner.View()
			statusText = "checking..."
		case depInstalled:
			icon = SuccessStyle.Render("✓")
			statusText = SuccessStyle.Render("installed")
			if d.version != "" {
				statusText += SubtitleStyle.Render(fmt.Sprintf(" (%s)", d.version))
			}
		case depMissing:
			icon = ErrorStyle.Render("✗")
			statusText = ErrorStyle.Render("missing")
			if d.errMsg != "" {
				statusText += SubtitleStyle.Render(fmt.Sprintf(" — %s", d.errMsg))
			}
		case depInstalling:
			icon = v.spinner.View()
			statusText = WarningStyle.Render("installing...")
		case depFailed:
			icon = ErrorStyle.Render("✗")
			statusText = ErrorStyle.Render("failed")
			if d.errMsg != "" {
				statusText += SubtitleStyle.Render(fmt.Sprintf(" — %s", d.errMsg))
			}
		}

		// Required tag
		reqTag := SubtitleStyle.Render("optional")
		if d.required {
			reqTag = lipgloss.NewStyle().Foreground(ColorDockerBlue).Render("required")
		}

		name := d.name
		if i == v.cursor {
			name = lipgloss.NewStyle().Bold(true).Foreground(ColorDockerBlue).Render(d.name)
		}

		line := fmt.Sprintf("%s %s %-22s %-10s %s", cursor, icon, name, reqTag, statusText)

		if i == v.cursor {
			// Highlight row
			row := SelectedRowStyle.Width(v.width - 4).Render(
				fmt.Sprintf(" %s %-22s %-10s %s", icon, d.name, reqTag, statusText),
			)
			line = cursor + row
		}

		b.WriteString(line)
		b.WriteString("\n")

		// Show description for selected item
		if i == v.cursor {
			b.WriteString(SubtitleStyle.Render(fmt.Sprintf("     %s", d.description)))
			b.WriteString("\n")
			if (d.status == depMissing || d.status == depFailed) && v.hasWinget {
				installHint := fmt.Sprintf("     Install: %s %s", d.installCmd, strings.Join(d.installArgs, " "))
				b.WriteString(SubtitleStyle.Render(installHint))
				b.WriteString("\n")
			}
		}
	}

	// Summary
	b.WriteString("\n")
	installed := 0
	missing := 0
	requiredMissing := 0
	for _, d := range v.deps {
		if d.status == depInstalled {
			installed++
		}
		if d.status == depMissing || d.status == depFailed {
			missing++
			if d.required {
				requiredMissing++
			}
		}
	}

	if v.allReady {
		b.WriteString(SuccessStyle.Render("  ✓ All required dependencies are installed. AppWrap is ready!"))
		b.WriteString("\n")
	} else if !v.checking {
		b.WriteString(fmt.Sprintf("  %s installed, %s missing",
			SuccessStyle.Render(fmt.Sprintf("%d", installed)),
			ErrorStyle.Render(fmt.Sprintf("%d", missing)),
		))
		if requiredMissing > 0 {
			b.WriteString(ErrorStyle.Render(fmt.Sprintf(" (%d required)", requiredMissing)))
		}
		b.WriteString("\n")
	}

	// Actions
	b.WriteString("\n")
	b.WriteString(HeadingStyle.Render("  Actions"))
	b.WriteString("\n\n")

	actions := []struct {
		key  string
		desc string
	}{
		{"Enter/i", "Install selected dependency"},
		{"a", "Install all missing"},
		{"r", "Re-check all"},
		{"esc", "Back to dashboard"},
	}
	for _, a := range actions {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			KeyStyle.Render(fmt.Sprintf("[%s]", a.key)),
			SubtitleStyle.Render(a.desc),
		))
	}

	return lipgloss.NewStyle().
		Width(v.width).
		Height(v.height).
		Render(b.String())
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
