package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/theencryptedafro/appwrap/internal/service"
)

type inspectCompleteMsg struct {
	result *service.InspectResult
	err    error
}

type inspectStep int

const (
	inspectStepPath inspectStep = iota
	inspectStepRunning
	inspectStepResults
)

type inspectView struct {
	svc    *service.AppService
	keys   KeyMap
	width  int
	height int

	step      inspectStep
	pathInput textinput.Model
	spinner   spinner.Model
	viewport  viewport.Model

	result *service.InspectResult
	err    error
}

func newInspectView(svc *service.AppService) *inspectView {
	ti := textinput.New()
	ti.Placeholder = "path/to/app.exe"
	ti.Focus()
	ti.Width = 60

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	vp := viewport.New(80, 20)

	return &inspectView{
		svc:       svc,
		keys:      DefaultKeyMap(),
		step:      inspectStepPath,
		pathInput: ti,
		spinner:   sp,
		viewport:  vp,
	}
}

func (v *inspectView) Title() string { return "Inspect Binary" }

func (v *inspectView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.pathInput.Width = w - 10
	if v.pathInput.Width < 20 {
		v.pathInput.Width = 20
	}
	v.viewport.Width = w - 6
	v.viewport.Height = h - 12
	if v.viewport.Height < 5 {
		v.viewport.Height = 5
	}
}

func (v *inspectView) Init() tea.Cmd {
	return textinput.Blink
}

func (v *inspectView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch v.step {
		case inspectStepPath:
			switch {
			case key.Matches(msg, v.keys.Back):
				return v, func() tea.Msg { return PopViewMsg{} }
			case key.Matches(msg, v.keys.Enter):
				if strings.TrimSpace(v.pathInput.Value()) != "" {
					return v, v.startInspect()
				}
			default:
				var cmd tea.Cmd
				v.pathInput, cmd = v.pathInput.Update(msg)
				return v, cmd
			}
		case inspectStepRunning:
			return v, nil
		case inspectStepResults:
			if key.Matches(msg, v.keys.Back) {
				return v, func() tea.Msg { return PopViewMsg{} }
			}
			var cmd tea.Cmd
			v.viewport, cmd = v.viewport.Update(msg)
			return v, cmd
		}

	case inspectCompleteMsg:
		v.result = msg.result
		v.err = msg.err
		v.step = inspectStepResults
		if v.result != nil {
			v.viewport.SetContent(v.renderResults())
		}
		return v, nil

	case spinner.TickMsg:
		if v.step == inspectStepRunning {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return v, tea.Batch(cmds...)
}

func (v *inspectView) startInspect() tea.Cmd {
	v.step = inspectStepRunning
	return tea.Batch(
		v.spinner.Tick,
		func() tea.Msg {
			result, err := v.svc.InspectBinary(context.Background(), service.InspectOpts{
				TargetPath: v.pathInput.Value(),
			})
			return inspectCompleteMsg{result: result, err: err}
		},
	)
}

func (v *inspectView) renderResults() string {
	if v.result == nil {
		return ""
	}
	r := v.result
	var b strings.Builder

	b.WriteString(fmt.Sprintf("File:      %s\n", r.FileName))
	b.WriteString(fmt.Sprintf("Path:      %s\n", r.FullPath))
	b.WriteString(fmt.Sprintf("Arch:      %s\n", r.Arch))
	b.WriteString(fmt.Sprintf("Subsystem: %s\n", r.Subsystem))
	b.WriteString(fmt.Sprintf("\nDLL Imports (%d):\n", len(r.Imports)))
	b.WriteString(strings.Repeat("-", 50) + "\n")

	// Table header.
	header := fmt.Sprintf("%-30s %-8s", "DLL Name", "Type")
	b.WriteString(TableHeaderStyle.Render(header) + "\n")

	for _, imp := range r.Imports {
		typeStr := "app"
		marker := ""
		if imp.IsSystem {
			typeStr = "system"
			marker = " [S]"
		}
		line := fmt.Sprintf("%-30s %-8s%s", imp.Name, typeStr, marker)
		b.WriteString(TableCellStyle.Render(line) + "\n")
	}

	// Summary.
	appCount := 0
	sysCount := 0
	for _, imp := range r.Imports {
		if imp.IsSystem {
			sysCount++
		} else {
			appCount++
		}
	}
	b.WriteString(fmt.Sprintf("\nTotal: %d | App: %d | System: %d [S]\n", len(r.Imports), appCount, sysCount))

	return b.String()
}

func (v *inspectView) View() string {
	var b strings.Builder

	switch v.step {
	case inspectStepPath:
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Inspect Binary (PE Analysis)"))
		b.WriteString("\n\n")
		b.WriteString("  Enter the path to an .exe file:\n\n")
		b.WriteString("  " + v.pathInput.View())
		b.WriteString("\n\n")
		b.WriteString(KeyHintStyle.Render("  enter: inspect  esc: back"))

	case inspectStepRunning:
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s Analyzing binary...\n", v.spinner.View()))

	case inspectStepResults:
		b.WriteString("\n")
		if v.err != nil {
			b.WriteString(HeadingStyle.Render("  Inspect Failed"))
			b.WriteString("\n\n")
			b.WriteString("  " + ErrorStyle.Render(v.err.Error()))
		} else {
			b.WriteString(HeadingStyle.Render("  Inspect Results"))
			b.WriteString("\n\n")
			b.WriteString("  " + NormalBorderStyle.Width(v.width-6).Render(v.viewport.View()))
		}
		b.WriteString("\n\n")
		b.WriteString(KeyHintStyle.Render("  scroll: up/down  esc: back"))
	}

	return lipgloss.NewStyle().Width(v.width).Height(v.height).Render(b.String())
}
