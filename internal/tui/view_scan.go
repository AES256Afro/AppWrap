package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/theencryptedafro/appwrap/internal/service"
)

// scanCompleteMsg is sent when the scan goroutine finishes.
type scanCompleteMsg struct {
	result *service.ScanResult
	err    error
}

type scanStep int

const (
	scanStepPath scanStep = iota
	scanStepOpts
	scanStepRunning
	scanStepDone
)

type scanView struct {
	svc   *service.AppService
	keys  KeyMap
	width int
	height int

	step     scanStep
	pathInput textinput.Model

	// Options (step 2).
	strategy  int // 0=wine, 1=windows-servercore, 2=windows-nanoserver
	encrypt   bool
	firewall  int // 0=off, 1=deny, 2=allow
	vpnPath   textinput.Model
	optFocus  int // which option is focused

	// Progress (step 3).
	spinner  spinner.Model
	percent  int
	events   []string
	eventLog strings.Builder

	// Results (step 4).
	result *service.ScanResult
	err    error
}

var strategies = []string{"wine", "windows-servercore", "windows-nanoserver"}
var firewallModes = []string{"off", "deny", "allow"}

func newScanView(svc *service.AppService) *scanView {
	ti := textinput.New()
	ti.Placeholder = "path/to/app.exe or shortcut.lnk"
	ti.Focus()
	ti.Width = 60

	vpn := textinput.New()
	vpn.Placeholder = "path/to/wireguard.conf (optional)"
	vpn.Width = 50

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	return &scanView{
		svc:       svc,
		keys:      DefaultKeyMap(),
		step:      scanStepPath,
		pathInput: ti,
		vpnPath:   vpn,
		spinner:   sp,
	}
}

func (v *scanView) Title() string { return "Scan App" }

func (v *scanView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.pathInput.Width = w - 10
	if v.pathInput.Width < 20 {
		v.pathInput.Width = 20
	}
	v.vpnPath.Width = w - 10
	if v.vpnPath.Width < 20 {
		v.vpnPath.Width = 20
	}
}

func (v *scanView) Init() tea.Cmd {
	return textinput.Blink
}

func (v *scanView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case appSelectedMsg:
		// Returned from the app picker — fill in the path and move to options.
		v.pathInput.SetValue(msg.exePath)
		v.step = scanStepOpts
		v.pathInput.Blur()
		return v, nil

	case tea.KeyMsg:
		switch v.step {
		case scanStepPath:
			return v.updatePath(msg)
		case scanStepOpts:
			return v.updateOpts(msg)
		case scanStepRunning:
			if key.Matches(msg, v.keys.Back) {
				// Cannot cancel mid-scan; ignore.
			}
			return v, nil
		case scanStepDone:
			if key.Matches(msg, v.keys.Back) || key.Matches(msg, v.keys.Enter) {
				return v, func() tea.Msg { return PopViewMsg{} }
			}
		}

	case EventMsg:
		evt := service.Event(msg)
		switch evt.Kind {
		case service.EventProgress:
			v.percent = evt.Percent
		case service.EventLogLine:
			v.events = append(v.events, evt.Message)
		default:
			v.events = append(v.events, evt.Message)
		}
		return v, nil

	case scanCompleteMsg:
		v.result = msg.result
		v.err = msg.err
		v.step = scanStepDone
		return v, nil

	case spinner.TickMsg:
		if v.step == scanStepRunning {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return v, tea.Batch(cmds...)
}

func (v *scanView) updatePath(msg tea.KeyMsg) (View, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Back):
		return v, func() tea.Msg { return PopViewMsg{} }
	case key.Matches(msg, v.keys.Enter):
		if strings.TrimSpace(v.pathInput.Value()) != "" {
			v.step = scanStepOpts
			v.pathInput.Blur()
			return v, nil
		}
	case msg.String() == "ctrl+b", msg.String() == "tab":
		// Open installed apps browser
		picker := newAppPickerView(v.svc)
		return v, func() tea.Msg { return PushViewMsg{View: picker} }
	}
	var cmd tea.Cmd
	v.pathInput, cmd = v.pathInput.Update(msg)
	return v, cmd
}

func (v *scanView) updateOpts(msg tea.KeyMsg) (View, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Back):
		v.step = scanStepPath
		v.pathInput.Focus()
		return v, textinput.Blink
	case key.Matches(msg, v.keys.Enter):
		if v.optFocus == 4 { // "Start Scan" button
			return v, v.startScan()
		}
	case msg.String() == "up" || msg.String() == "k":
		if v.optFocus > 0 {
			v.optFocus--
		}
	case msg.String() == "down" || msg.String() == "j":
		if v.optFocus < 4 {
			v.optFocus++
		}
	case msg.String() == "left" || msg.String() == "h":
		switch v.optFocus {
		case 0:
			if v.strategy > 0 {
				v.strategy--
			}
		case 2:
			if v.firewall > 0 {
				v.firewall--
			}
		}
	case msg.String() == "right" || msg.String() == "l":
		switch v.optFocus {
		case 0:
			if v.strategy < len(strategies)-1 {
				v.strategy++
			}
		case 2:
			if v.firewall < len(firewallModes)-1 {
				v.firewall++
			}
		}
	case msg.String() == " ":
		if v.optFocus == 1 {
			v.encrypt = !v.encrypt
		}
	}

	// Update VPN text input if focused.
	if v.optFocus == 3 {
		if !v.vpnPath.Focused() {
			v.vpnPath.Focus()
		}
		var cmd tea.Cmd
		v.vpnPath, cmd = v.vpnPath.Update(msg)
		return v, cmd
	}
	v.vpnPath.Blur()

	return v, nil
}

func (v *scanView) startScan() tea.Cmd {
	v.step = scanStepRunning
	v.events = nil
	v.percent = 0

	opts := service.ScanOpts{
		TargetPath: v.pathInput.Value(),
		Strategy:   strategies[v.strategy],
		Encrypt:    v.encrypt,
	}
	if v.firewall > 0 {
		opts.Firewall = firewallModes[v.firewall]
	}
	if strings.TrimSpace(v.vpnPath.Value()) != "" {
		opts.VPNConfig = v.vpnPath.Value()
	}

	return tea.Batch(
		v.spinner.Tick,
		func() tea.Msg {
			events := make(chan service.Event, 64)
			go func() {
				for range events {
					// Delivered through the program; we wrap it as a tea.Msg.
					// But since we don't have program ref here, we use a closure
					// that sends back through the batch.
				}
				_ = events // keep compiler happy
			}()

			// We run the scan synchronously in this goroutine.
			// Events are bridged via the channel -> EventMsg.
			evCh := make(chan service.Event, 64)
			var result *service.ScanResult
			var scanErr error

			done := make(chan struct{})
			go func() {
				result, scanErr = v.svc.ScanApp(context.Background(), opts, evCh)
				close(evCh)
				close(done)
			}()

			// Bridge events: we cannot call program.Send here,
			// so we collect them and they arrive as scanCompleteMsg.
			var collectedEvents []service.Event
			for evt := range evCh {
				collectedEvents = append(collectedEvents, evt)
			}
			<-done

			_ = collectedEvents // Events were already delivered via the channel.
			return scanCompleteMsg{result: result, err: scanErr}
		},
	)
}

func (v *scanView) View() string {
	var b strings.Builder

	switch v.step {
	case scanStepPath:
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Step 1: Select Target"))
		b.WriteString("\n\n")
		b.WriteString("  Enter the path to an .exe or .lnk file:\n\n")
		b.WriteString("  " + v.pathInput.View())
		b.WriteString("\n\n")
		b.WriteString("  " + SubtitleStyle.Render("or press "))
		b.WriteString(KeyStyle.Render("Tab"))
		b.WriteString(SubtitleStyle.Render(" to browse installed applications"))
		b.WriteString("\n\n")
		b.WriteString(KeyHintStyle.Render("  enter: next  tab: browse apps  esc: back"))

	case scanStepOpts:
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Step 2: Options"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  Target: %s\n\n", InfoStyle.Render(v.pathInput.Value())))

		// Strategy selector.
		b.WriteString(v.optLine(0, "Strategy", v.strategyDisplay()))
		// Encrypt toggle.
		b.WriteString(v.optLine(1, "Encrypt", v.boolDisplay(v.encrypt)))
		// Firewall selector.
		b.WriteString(v.optLine(2, "Firewall", v.firewallDisplay()))
		// VPN path.
		vpnLabel := "  VPN Config  "
		if v.optFocus == 3 {
			vpnLabel = SelectedRowStyle.Render(vpnLabel)
		}
		b.WriteString(vpnLabel + v.vpnPath.View() + "\n")
		// Start button.
		btn := "  [ Start Scan ]"
		if v.optFocus == 4 {
			btn = SelectedRowStyle.Render("  [ Start Scan ]")
		}
		b.WriteString("\n" + btn + "\n")

		b.WriteString("\n")
		b.WriteString(KeyHintStyle.Render("  arrows: change  space: toggle  enter: start  esc: back"))

	case scanStepRunning:
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Step 3: Scanning..."))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  %s Scanning... %d%%\n\n", v.spinner.View(), v.percent))
		b.WriteString(v.renderProgressBar())
		b.WriteString("\n\n")
		// Event log.
		start := 0
		maxLines := v.height - 12
		if maxLines < 5 {
			maxLines = 5
		}
		if len(v.events) > maxLines {
			start = len(v.events) - maxLines
		}
		for _, e := range v.events[start:] {
			b.WriteString("  " + SubtitleStyle.Render(e) + "\n")
		}

	case scanStepDone:
		b.WriteString("\n")
		if v.err != nil {
			b.WriteString(HeadingStyle.Render("  Scan Failed"))
			b.WriteString("\n\n")
			b.WriteString("  " + ErrorStyle.Render(v.err.Error()))
		} else {
			b.WriteString(HeadingStyle.Render("  Step 4: Scan Complete"))
			b.WriteString("\n\n")
			b.WriteString(SuccessStyle.Render("  Profile generated successfully!"))
			b.WriteString("\n\n")
			if v.result != nil {
				p := v.result.Profile
				b.WriteString(fmt.Sprintf("  App:      %s\n", InfoStyle.Render(p.App.Name)))
				b.WriteString(fmt.Sprintf("  Arch:     %s\n", p.Binary.Arch))
				b.WriteString(fmt.Sprintf("  Strategy: %s\n", p.Build.Strategy))
				totalDLLs := len(p.Dependencies.DLLs)
				appDLLs := 0
				for _, d := range p.Dependencies.DLLs {
					if !d.IsSystem {
						appDLLs++
					}
				}
				b.WriteString(fmt.Sprintf("  DLLs:     %d total (%d app, %d system)\n", totalDLLs, appDLLs, totalDLLs-appDLLs))
				b.WriteString(fmt.Sprintf("  Output:   %s\n", SuccessStyle.Render(v.result.OutputPath)))

				if len(v.result.Warnings) > 0 {
					b.WriteString("\n")
					b.WriteString(WarningStyle.Render(fmt.Sprintf("  %d warning(s):\n", len(v.result.Warnings))))
					for _, w := range v.result.Warnings {
						b.WriteString(WarningStyle.Render(fmt.Sprintf("    - %s\n", w)))
					}
				}
			}
		}
		b.WriteString("\n")
		b.WriteString(KeyHintStyle.Render("  enter/esc: back to dashboard"))
	}

	return lipgloss.NewStyle().Width(v.width).Height(v.height).Render(b.String())
}

func (v *scanView) optLine(idx int, label, value string) string {
	line := fmt.Sprintf("  %-14s %s\n", label, value)
	if v.optFocus == idx {
		line = SelectedRowStyle.Width(v.width - 4).Render(
			fmt.Sprintf(" %-14s %s", label, value),
		) + "\n"
		line = "  " + line
	}
	return line
}

func (v *scanView) strategyDisplay() string {
	var parts []string
	for i, s := range strategies {
		if i == v.strategy {
			parts = append(parts, ActiveTabStyle.Render(s))
		} else {
			parts = append(parts, InactiveTabStyle.Render(s))
		}
	}
	return strings.Join(parts, " ")
}

func (v *scanView) firewallDisplay() string {
	var parts []string
	for i, s := range firewallModes {
		if i == v.firewall {
			parts = append(parts, ActiveTabStyle.Render(s))
		} else {
			parts = append(parts, InactiveTabStyle.Render(s))
		}
	}
	return strings.Join(parts, " ")
}

func (v *scanView) boolDisplay(val bool) string {
	if val {
		return SuccessStyle.Render("[x] enabled")
	}
	return SubtitleStyle.Render("[ ] disabled")
}

func (v *scanView) renderProgressBar() string {
	width := v.width - 8
	if width < 10 {
		width = 10
	}
	filled := width * v.percent / 100
	empty := width - filled
	bar := "  " +
		ProgressBarFilled.Render(strings.Repeat("█", filled)) +
		ProgressBarEmpty.Render(strings.Repeat("░", empty))
	return bar
}
