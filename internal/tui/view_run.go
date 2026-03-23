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

type runCompleteMsg struct {
	err error
}

type runStep int

const (
	runStepImage runStep = iota
	runStepOpts
	runStepRunning
	runStepDone
)

var displayModes = []string{"none", "vnc", "novnc", "rdp"}

type runView struct {
	svc    *service.AppService
	keys   KeyMap
	width  int
	height int

	step       runStep
	imageInput textinput.Model
	nameInput  textinput.Model
	profInput  textinput.Model
	display    int  // index into displayModes
	detach     bool
	optFocus   int

	spinner spinner.Model
	events  []string
	err     error
}

func newRunView(svc *service.AppService) *runView {
	ii := textinput.New()
	ii.Placeholder = "myapp:latest"
	ii.Focus()
	ii.Width = 50

	ni := textinput.New()
	ni.Placeholder = "container-name (optional)"
	ni.Width = 40

	pi := textinput.New()
	pi.Placeholder = "profile.yaml (optional, for security)"
	pi.Width = 50

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	return &runView{
		svc:        svc,
		keys:       DefaultKeyMap(),
		step:       runStepImage,
		imageInput: ii,
		nameInput:  ni,
		profInput:  pi,
		spinner:    sp,
	}
}

func (v *runView) Title() string { return "Run Container" }

func (v *runView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.imageInput.Width = w - 10
	if v.imageInput.Width < 20 {
		v.imageInput.Width = 20
	}
}

func (v *runView) Init() tea.Cmd {
	return textinput.Blink
}

func (v *runView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch v.step {
		case runStepImage:
			return v.updateImage(msg)
		case runStepOpts:
			return v.updateOpts(msg)
		case runStepRunning:
			return v, nil
		case runStepDone:
			if key.Matches(msg, v.keys.Back) || key.Matches(msg, v.keys.Enter) {
				return v, func() tea.Msg { return PopViewMsg{} }
			}
		}

	case EventMsg:
		evt := service.Event(msg)
		v.events = append(v.events, evt.Message)
		return v, nil

	case runCompleteMsg:
		v.err = msg.err
		v.step = runStepDone
		return v, nil

	case spinner.TickMsg:
		if v.step == runStepRunning {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return v, tea.Batch(cmds...)
}

func (v *runView) updateImage(msg tea.KeyMsg) (View, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Back):
		return v, func() tea.Msg { return PopViewMsg{} }
	case key.Matches(msg, v.keys.Enter):
		if strings.TrimSpace(v.imageInput.Value()) != "" {
			v.step = runStepOpts
			v.imageInput.Blur()
			return v, nil
		}
	}
	var cmd tea.Cmd
	v.imageInput, cmd = v.imageInput.Update(msg)
	return v, cmd
}

func (v *runView) updateOpts(msg tea.KeyMsg) (View, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Back):
		v.step = runStepImage
		v.imageInput.Focus()
		return v, textinput.Blink
	case msg.String() == "up":
		if v.optFocus > 0 {
			v.optFocus--
		}
	case msg.String() == "down":
		if v.optFocus < 4 {
			v.optFocus++
		}
	case msg.String() == "left":
		if v.optFocus == 0 && v.display > 0 {
			v.display--
		}
	case msg.String() == "right":
		if v.optFocus == 0 && v.display < len(displayModes)-1 {
			v.display++
		}
	case msg.String() == " ":
		if v.optFocus == 3 {
			v.detach = !v.detach
		}
	case key.Matches(msg, v.keys.Enter):
		if v.optFocus == 4 {
			return v, v.startRun()
		}
	}

	// Text inputs for name and profile.
	if v.optFocus == 1 {
		if !v.nameInput.Focused() {
			v.nameInput.Focus()
			v.profInput.Blur()
		}
		var cmd tea.Cmd
		v.nameInput, cmd = v.nameInput.Update(msg)
		return v, cmd
	}
	v.nameInput.Blur()

	if v.optFocus == 2 {
		if !v.profInput.Focused() {
			v.profInput.Focus()
		}
		var cmd tea.Cmd
		v.profInput, cmd = v.profInput.Update(msg)
		return v, cmd
	}
	v.profInput.Blur()

	return v, nil
}

func (v *runView) startRun() tea.Cmd {
	v.step = runStepRunning
	v.events = nil

	opts := service.RunOpts{
		Image:   v.imageInput.Value(),
		Display: displayModes[v.display],
		Detach:  v.detach,
		Name:    v.nameInput.Value(),
		Profile: v.profInput.Value(),
	}

	return tea.Batch(
		v.spinner.Tick,
		func() tea.Msg {
			evCh := make(chan service.Event, 64)
			var runErr error

			done := make(chan struct{})
			go func() {
				runErr = v.svc.RunContainer(context.Background(), opts, evCh)
				close(evCh)
				close(done)
			}()

			for range evCh {
			}
			<-done

			return runCompleteMsg{err: runErr}
		},
	)
}

func (v *runView) View() string {
	var b strings.Builder

	switch v.step {
	case runStepImage:
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Step 1: Select Image"))
		b.WriteString("\n\n")
		b.WriteString("  Enter the Docker image name:\n\n")
		b.WriteString("  " + v.imageInput.View())
		b.WriteString("\n\n")
		b.WriteString(KeyHintStyle.Render("  enter: next  esc: back"))

	case runStepOpts:
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Step 2: Run Options"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  Image: %s\n\n", InfoStyle.Render(v.imageInput.Value())))

		// Display mode selector.
		b.WriteString(v.optLine(0, "Display", v.displaySelector()))
		// Name input.
		nameLabel := "  Name          "
		if v.optFocus == 1 {
			nameLabel = SelectedRowStyle.Render(nameLabel)
		}
		b.WriteString(nameLabel + v.nameInput.View() + "\n")
		// Profile input.
		profLabel := "  Profile       "
		if v.optFocus == 2 {
			profLabel = SelectedRowStyle.Render(profLabel)
		}
		b.WriteString(profLabel + v.profInput.View() + "\n")
		// Detach toggle.
		detachStr := v.boolDisplay(v.detach)
		line := fmt.Sprintf("  %-14s %s\n", "Detach", detachStr)
		if v.optFocus == 3 {
			line = "  " + SelectedRowStyle.Width(v.width-4).Render(
				fmt.Sprintf(" %-14s %s", "Detach", detachStr),
			) + "\n"
		}
		b.WriteString(line)

		// Run button.
		btn := "  [ Start Container ]"
		if v.optFocus == 4 {
			btn = SelectedRowStyle.Render("  [ Start Container ]")
		}
		b.WriteString("\n" + btn + "\n")

		b.WriteString("\n")
		b.WriteString(KeyHintStyle.Render("  arrows: change  space: toggle  enter: run  esc: back"))

	case runStepRunning:
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Step 3: Starting Container..."))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  %s Starting %s...\n\n", v.spinner.View(), v.imageInput.Value()))
		for _, e := range v.events {
			b.WriteString("  " + SubtitleStyle.Render(e) + "\n")
		}

	case runStepDone:
		b.WriteString("\n")
		if v.err != nil {
			b.WriteString(HeadingStyle.Render("  Run Failed"))
			b.WriteString("\n\n")
			b.WriteString("  " + ErrorStyle.Render(v.err.Error()))
		} else {
			b.WriteString(HeadingStyle.Render("  Container Started"))
			b.WriteString("\n\n")
			b.WriteString(SuccessStyle.Render("  Container is running!"))
			b.WriteString("\n\n")
			b.WriteString(fmt.Sprintf("  Image:   %s\n", v.imageInput.Value()))
			b.WriteString(fmt.Sprintf("  Display: %s\n", displayModes[v.display]))
			if displayModes[v.display] == "novnc" {
				b.WriteString(InfoStyle.Render("  Access: http://localhost:6080") + "\n")
			}
		}
		b.WriteString("\n")
		b.WriteString(KeyHintStyle.Render("  enter/esc: back to dashboard"))
	}

	return lipgloss.NewStyle().Width(v.width).Height(v.height).Render(b.String())
}

func (v *runView) optLine(idx int, label, value string) string {
	line := fmt.Sprintf("  %-14s %s\n", label, value)
	if v.optFocus == idx {
		line = "  " + SelectedRowStyle.Width(v.width-4).Render(
			fmt.Sprintf(" %-14s %s", label, value),
		) + "\n"
	}
	return line
}

func (v *runView) displaySelector() string {
	var parts []string
	for i, s := range displayModes {
		if i == v.display {
			parts = append(parts, ActiveTabStyle.Render(s))
		} else {
			parts = append(parts, InactiveTabStyle.Render(s))
		}
	}
	return strings.Join(parts, " ")
}

func (v *runView) boolDisplay(val bool) string {
	if val {
		return SuccessStyle.Render("[x] enabled")
	}
	return SubtitleStyle.Render("[ ] disabled")
}
