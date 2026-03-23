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

type buildCompleteMsg struct {
	result *service.BuildResult
	err    error
}

type buildStep int

const (
	buildStepProfile buildStep = iota
	buildStepOpts
	buildStepRunning
	buildStepDone
)

type buildView struct {
	svc    *service.AppService
	keys   KeyMap
	width  int
	height int

	step         buildStep
	profileInput textinput.Model
	tagInput     textinput.Model
	noCache      bool
	optFocus     int

	spinner  spinner.Model
	percent  int
	logLines []string
	viewport viewport.Model

	result *service.BuildResult
	err    error
}

func newBuildView(svc *service.AppService) *buildView {
	pi := textinput.New()
	pi.Placeholder = "path/to/app-profile.yaml"
	pi.Focus()
	pi.Width = 60

	ti := textinput.New()
	ti.Placeholder = "myapp:latest"
	ti.Width = 40

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	vp := viewport.New(80, 20)

	return &buildView{
		svc:          svc,
		keys:         DefaultKeyMap(),
		step:         buildStepProfile,
		profileInput: pi,
		tagInput:     ti,
		spinner:      sp,
		viewport:     vp,
	}
}

func (v *buildView) Title() string { return "Build Image" }

func (v *buildView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.profileInput.Width = w - 10
	if v.profileInput.Width < 20 {
		v.profileInput.Width = 20
	}
	v.tagInput.Width = w - 10
	if v.tagInput.Width < 20 {
		v.tagInput.Width = 20
	}
	v.viewport.Width = w - 4
	v.viewport.Height = h - 10
	if v.viewport.Height < 5 {
		v.viewport.Height = 5
	}
}

func (v *buildView) Init() tea.Cmd {
	return textinput.Blink
}

func (v *buildView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch v.step {
		case buildStepProfile:
			return v.updateProfile(msg)
		case buildStepOpts:
			return v.updateOpts(msg)
		case buildStepRunning:
			// Allow scrolling in viewport.
			var cmd tea.Cmd
			v.viewport, cmd = v.viewport.Update(msg)
			return v, cmd
		case buildStepDone:
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
			v.logLines = append(v.logLines, evt.Message)
			v.viewport.SetContent(strings.Join(v.logLines, "\n"))
			v.viewport.GotoBottom()
		default:
			v.logLines = append(v.logLines, evt.Message)
			v.viewport.SetContent(strings.Join(v.logLines, "\n"))
			v.viewport.GotoBottom()
		}
		return v, nil

	case buildCompleteMsg:
		v.result = msg.result
		v.err = msg.err
		v.step = buildStepDone
		return v, nil

	case spinner.TickMsg:
		if v.step == buildStepRunning {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return v, tea.Batch(cmds...)
}

func (v *buildView) updateProfile(msg tea.KeyMsg) (View, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Back):
		return v, func() tea.Msg { return PopViewMsg{} }
	case key.Matches(msg, v.keys.Enter):
		if strings.TrimSpace(v.profileInput.Value()) != "" {
			v.step = buildStepOpts
			v.profileInput.Blur()
			v.tagInput.Focus()
			return v, textinput.Blink
		}
	}
	var cmd tea.Cmd
	v.profileInput, cmd = v.profileInput.Update(msg)
	return v, cmd
}

func (v *buildView) updateOpts(msg tea.KeyMsg) (View, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Back):
		v.step = buildStepProfile
		v.tagInput.Blur()
		v.profileInput.Focus()
		return v, textinput.Blink
	case msg.String() == "up" || msg.String() == "k":
		if v.optFocus > 0 {
			v.optFocus--
		}
	case msg.String() == "down" || msg.String() == "j":
		if v.optFocus < 2 {
			v.optFocus++
		}
	case msg.String() == " ":
		if v.optFocus == 1 {
			v.noCache = !v.noCache
		}
	case key.Matches(msg, v.keys.Enter):
		if v.optFocus == 2 {
			return v, v.startBuild()
		}
	}

	// Update tag input if focused.
	if v.optFocus == 0 {
		if !v.tagInput.Focused() {
			v.tagInput.Focus()
		}
		var cmd tea.Cmd
		v.tagInput, cmd = v.tagInput.Update(msg)
		return v, cmd
	}
	v.tagInput.Blur()

	return v, nil
}

func (v *buildView) startBuild() tea.Cmd {
	v.step = buildStepRunning
	v.logLines = nil
	v.percent = 0

	opts := service.BuildOpts{
		ProfilePath: v.profileInput.Value(),
		Tag:         v.tagInput.Value(),
		NoCache:     v.noCache,
	}

	return tea.Batch(
		v.spinner.Tick,
		func() tea.Msg {
			evCh := make(chan service.Event, 128)
			var result *service.BuildResult
			var buildErr error

			done := make(chan struct{})
			go func() {
				result, buildErr = v.svc.BuildImage(context.Background(), opts, evCh)
				close(evCh)
				close(done)
			}()

			for range evCh {
				// Events are consumed; in a real implementation
				// they'd be bridged via program.Send().
			}
			<-done

			return buildCompleteMsg{result: result, err: buildErr}
		},
	)
}

func (v *buildView) View() string {
	var b strings.Builder

	switch v.step {
	case buildStepProfile:
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Step 1: Select Profile"))
		b.WriteString("\n\n")
		b.WriteString("  Enter the path to a profile file:\n\n")
		b.WriteString("  " + v.profileInput.View())
		b.WriteString("\n\n")
		b.WriteString(KeyHintStyle.Render("  enter: next  esc: back"))

	case buildStepOpts:
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Step 2: Build Options"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  Profile: %s\n\n", InfoStyle.Render(v.profileInput.Value())))

		// Tag input.
		tagLabel := "  Image Tag   "
		if v.optFocus == 0 {
			tagLabel = SelectedRowStyle.Render(tagLabel)
		}
		b.WriteString(tagLabel + v.tagInput.View() + "\n")

		// No cache toggle.
		noCacheStr := v.boolDisplay(v.noCache)
		line := fmt.Sprintf("  %-14s %s\n", "No Cache", noCacheStr)
		if v.optFocus == 1 {
			line = "  " + SelectedRowStyle.Width(v.width-4).Render(
				fmt.Sprintf(" %-14s %s", "No Cache", noCacheStr),
			) + "\n"
		}
		b.WriteString(line)

		// Start button.
		btn := "  [ Start Build ]"
		if v.optFocus == 2 {
			btn = SelectedRowStyle.Render("  [ Start Build ]")
		}
		b.WriteString("\n" + btn + "\n")

		b.WriteString("\n")
		b.WriteString(KeyHintStyle.Render("  space: toggle  enter: build  esc: back"))

	case buildStepRunning:
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Step 3: Building..."))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  %s Building... %d%%\n\n", v.spinner.View(), v.percent))
		b.WriteString(v.renderProgressBar())
		b.WriteString("\n\n")
		// Viewport for log output.
		b.WriteString(NormalBorderStyle.Width(v.width - 6).Render(v.viewport.View()))

	case buildStepDone:
		b.WriteString("\n")
		if v.err != nil {
			b.WriteString(HeadingStyle.Render("  Build Failed"))
			b.WriteString("\n\n")
			b.WriteString("  " + ErrorStyle.Render(v.err.Error()))
		} else {
			b.WriteString(HeadingStyle.Render("  Step 4: Build Complete"))
			b.WriteString("\n\n")
			b.WriteString(SuccessStyle.Render("  Image built successfully!"))
			b.WriteString("\n\n")
			if v.result != nil {
				b.WriteString(fmt.Sprintf("  Image Tag: %s\n", InfoStyle.Render(v.result.ImageTag)))
			}
		}
		b.WriteString("\n")
		b.WriteString(KeyHintStyle.Render("  enter/esc: back to dashboard"))
	}

	return lipgloss.NewStyle().Width(v.width).Height(v.height).Render(b.String())
}

func (v *buildView) boolDisplay(val bool) string {
	if val {
		return SuccessStyle.Render("[x] enabled")
	}
	return SubtitleStyle.Render("[ ] disabled")
}

func (v *buildView) renderProgressBar() string {
	width := v.width - 8
	if width < 10 {
		width = 10
	}
	filled := width * v.percent / 100
	empty := width - filled
	return "  " +
		ProgressBarFilled.Render(strings.Repeat("█", filled)) +
		ProgressBarEmpty.Render(strings.Repeat("░", empty))
}
