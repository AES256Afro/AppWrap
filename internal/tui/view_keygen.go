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

type keygenCompleteMsg struct {
	result *service.KeygenResult
	err    error
}

type keygenStep int

const (
	keygenStepDir keygenStep = iota
	keygenStepRunning
	keygenStepDone
)

type keygenView struct {
	svc    *service.AppService
	keys   KeyMap
	width  int
	height int

	step     keygenStep
	dirInput textinput.Model
	spinner  spinner.Model

	result *service.KeygenResult
	err    error
}

func newKeygenView(svc *service.AppService) *keygenView {
	ti := textinput.New()
	ti.Placeholder = "output directory (default: current dir)"
	ti.Focus()
	ti.Width = 60

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	return &keygenView{
		svc:      svc,
		keys:     DefaultKeyMap(),
		step:     keygenStepDir,
		dirInput: ti,
		spinner:  sp,
	}
}

func (v *keygenView) Title() string { return "Generate Keys" }

func (v *keygenView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.dirInput.Width = w - 10
	if v.dirInput.Width < 20 {
		v.dirInput.Width = 20
	}
}

func (v *keygenView) Init() tea.Cmd {
	return textinput.Blink
}

func (v *keygenView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch v.step {
		case keygenStepDir:
			switch {
			case key.Matches(msg, v.keys.Back):
				return v, func() tea.Msg { return PopViewMsg{} }
			case key.Matches(msg, v.keys.Enter):
				return v, v.startKeygen()
			default:
				var cmd tea.Cmd
				v.dirInput, cmd = v.dirInput.Update(msg)
				return v, cmd
			}
		case keygenStepRunning:
			return v, nil
		case keygenStepDone:
			if key.Matches(msg, v.keys.Back) || key.Matches(msg, v.keys.Enter) {
				return v, func() tea.Msg { return PopViewMsg{} }
			}
		}

	case keygenCompleteMsg:
		v.result = msg.result
		v.err = msg.err
		v.step = keygenStepDone
		return v, nil

	case spinner.TickMsg:
		if v.step == keygenStepRunning {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return v, tea.Batch(cmds...)
}

func (v *keygenView) startKeygen() tea.Cmd {
	v.step = keygenStepRunning
	dir := v.dirInput.Value()
	if dir == "" {
		dir = "."
	}

	return tea.Batch(
		v.spinner.Tick,
		func() tea.Msg {
			result, err := v.svc.GenerateKeys(context.Background(), service.KeygenOpts{
				OutputDir: dir,
			})
			return keygenCompleteMsg{result: result, err: err}
		},
	)
}

func (v *keygenView) View() string {
	var b strings.Builder

	switch v.step {
	case keygenStepDir:
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Generate Age Encryption Keys"))
		b.WriteString("\n\n")
		b.WriteString("  Enter the output directory (leave empty for current dir):\n\n")
		b.WriteString("  " + v.dirInput.View())
		b.WriteString("\n\n")
		b.WriteString(KeyHintStyle.Render("  enter: generate  esc: back"))

	case keygenStepRunning:
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s Generating keys...\n", v.spinner.View()))

	case keygenStepDone:
		b.WriteString("\n")
		if v.err != nil {
			b.WriteString(HeadingStyle.Render("  Key Generation Failed"))
			b.WriteString("\n\n")
			b.WriteString("  " + ErrorStyle.Render(v.err.Error()))
		} else {
			b.WriteString(HeadingStyle.Render("  Keys Generated Successfully"))
			b.WriteString("\n\n")
			b.WriteString(SuccessStyle.Render("  Age keypair created!"))
			b.WriteString("\n\n")
			if v.result != nil {
				b.WriteString(fmt.Sprintf("  Recipient: %s\n", InfoStyle.Render(v.result.Recipient)))
				b.WriteString(fmt.Sprintf("  Key File:  %s\n", SuccessStyle.Render(v.result.KeyFile)))
				b.WriteString("\n")
				b.WriteString(WarningStyle.Render("  Keep the key file secure! It is needed to decrypt at runtime."))
			}
		}
		b.WriteString("\n\n")
		b.WriteString(KeyHintStyle.Render("  enter/esc: back to dashboard"))
	}

	return lipgloss.NewStyle().Width(v.width).Height(v.height).Render(b.String())
}
