package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/theencryptedafro/appwrap/internal/service"
)

// containersLoadedMsg is sent after listing containers.
type containersLoadedMsg struct {
	containers []service.ContainerInfo
	err        error
}

// containerActionMsg reports the result of stop/remove.
type containerActionMsg struct {
	action string
	err    error
}

type containersMode int

const (
	containersModeList containersMode = iota
	containersModeLogs
)

type containersView struct {
	svc    *service.AppService
	keys   KeyMap
	width  int
	height int

	mode       containersMode
	containers []service.ContainerInfo
	cursor     int
	err        error

	// Logs sub-view.
	logsVP       viewport.Model
	logLines     []string
	logsSpinner  spinner.Model
	logsCancel   context.CancelFunc
}

func newContainersView(svc *service.AppService) *containersView {
	vp := viewport.New(80, 20)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	return &containersView{
		svc:         svc,
		keys:        DefaultKeyMap(),
		mode:        containersModeList,
		logsVP:      vp,
		logsSpinner: sp,
	}
}

func (v *containersView) Title() string { return "Containers" }

func (v *containersView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.logsVP.Width = w - 6
	v.logsVP.Height = h - 8
	if v.logsVP.Height < 5 {
		v.logsVP.Height = 5
	}
}

func (v *containersView) Init() tea.Cmd {
	return v.loadContainers
}

func (v *containersView) loadContainers() tea.Msg {
	containers, err := v.svc.ListContainers(context.Background())
	return containersLoadedMsg{containers: containers, err: err}
}

func (v *containersView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case containersLoadedMsg:
		v.containers = msg.containers
		v.err = msg.err
		if v.cursor >= len(v.containers) {
			v.cursor = len(v.containers) - 1
		}
		if v.cursor < 0 {
			v.cursor = 0
		}
		return v, nil

	case containerActionMsg:
		if msg.err != nil {
			v.err = msg.err
		}
		return v, v.loadContainers

	case EventMsg:
		if v.mode == containersModeLogs {
			evt := service.Event(msg)
			if evt.Kind == service.EventLogLine {
				v.logLines = append(v.logLines, evt.Message)
				v.logsVP.SetContent(strings.Join(v.logLines, "\n"))
				v.logsVP.GotoBottom()
			}
		}
		return v, nil

	case tea.KeyMsg:
		switch v.mode {
		case containersModeList:
			return v.updateList(msg)
		case containersModeLogs:
			return v.updateLogs(msg)
		}

	case spinner.TickMsg:
		if v.mode == containersModeLogs {
			var cmd tea.Cmd
			v.logsSpinner, cmd = v.logsSpinner.Update(msg)
			return v, cmd
		}
	}

	return v, nil
}

func (v *containersView) updateList(msg tea.KeyMsg) (View, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Back):
		return v, func() tea.Msg { return PopViewMsg{} }
	case msg.String() == "up" || msg.String() == "k":
		if v.cursor > 0 {
			v.cursor--
		}
	case msg.String() == "down" || msg.String() == "j":
		if v.cursor < len(v.containers)-1 {
			v.cursor++
		}
	case msg.String() == "s":
		if len(v.containers) > 0 {
			id := v.containers[v.cursor].ID
			return v, func() tea.Msg {
				err := v.svc.StopContainer(context.Background(), id)
				return containerActionMsg{action: "stop", err: err}
			}
		}
	case msg.String() == "r":
		if len(v.containers) > 0 {
			id := v.containers[v.cursor].ID
			return v, func() tea.Msg {
				err := v.svc.RemoveContainer(context.Background(), id)
				return containerActionMsg{action: "remove", err: err}
			}
		}
	case msg.String() == "l":
		if len(v.containers) > 0 {
			return v, v.startLogs()
		}
	case msg.String() == "R":
		// Refresh.
		return v, v.loadContainers
	}
	return v, nil
}

func (v *containersView) updateLogs(msg tea.KeyMsg) (View, tea.Cmd) {
	if key.Matches(msg, v.keys.Back) {
		if v.logsCancel != nil {
			v.logsCancel()
			v.logsCancel = nil
		}
		v.mode = containersModeList
		v.logLines = nil
		return v, nil
	}
	var cmd tea.Cmd
	v.logsVP, cmd = v.logsVP.Update(msg)
	return v, cmd
}

func (v *containersView) startLogs() tea.Cmd {
	v.mode = containersModeLogs
	v.logLines = nil
	id := v.containers[v.cursor].ID

	ctx, cancel := context.WithCancel(context.Background())
	v.logsCancel = cancel

	return tea.Batch(
		v.logsSpinner.Tick,
		func() tea.Msg {
			evCh := make(chan service.Event, 128)
			go func() {
				_ = v.svc.ContainerLogs(ctx, id, evCh)
				close(evCh)
			}()

			for range evCh {
				// In a real implementation, these would be bridged
				// via program.Send(EventMsg(evt)).
			}
			return nil
		},
	)
}

func (v *containersView) View() string {
	var b strings.Builder

	switch v.mode {
	case containersModeList:
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Containers"))
		b.WriteString("\n\n")

		if v.err != nil {
			b.WriteString("  " + ErrorStyle.Render(v.err.Error()) + "\n\n")
		}

		if len(v.containers) == 0 {
			b.WriteString(SubtitleStyle.Render("  No containers found.\n"))
			b.WriteString(SubtitleStyle.Render("  Use [R]un to start one.\n"))
		} else {
			// Table header.
			header := fmt.Sprintf("  %-14s %-24s %-14s %-24s", "Name", "Image", "Status", "Ports")
			b.WriteString(TableHeaderStyle.Render(header) + "\n")

			for i, c := range v.containers {
				name := c.Name
				if len(name) > 13 {
					name = name[:13]
				}
				image := c.Image
				if len(image) > 23 {
					image = image[:23]
				}
				status := c.Status
				if len(status) > 13 {
					status = status[:13]
				}
				ports := c.Ports
				if len(ports) > 23 {
					ports = ports[:23]
				}

				line := fmt.Sprintf("  %-14s %-24s %-14s %-24s", name, image, status, ports)
				if i == v.cursor {
					line = SelectedRowStyle.Width(v.width - 4).Render(
						fmt.Sprintf(" %-14s %-24s %-14s %-24s", name, image, status, ports),
					)
					line = "  " + line
				}

				// Color the status.
				if strings.Contains(strings.ToLower(c.State), "running") && i != v.cursor {
					statusStyled := SuccessStyle.Render(status)
					line = fmt.Sprintf("  %-14s %-24s %-14s %-24s", name, image, statusStyled, ports)
				}

				b.WriteString(line + "\n")
			}
		}

		b.WriteString("\n")
		b.WriteString(KeyHintStyle.Render("  s: stop  l: logs  r: remove  R: refresh  esc: back"))

	case containersModeLogs:
		b.WriteString("\n")
		containerName := ""
		if v.cursor < len(v.containers) {
			containerName = v.containers[v.cursor].Name
		}
		b.WriteString(HeadingStyle.Render(fmt.Sprintf("  Logs: %s", containerName)))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  %s Streaming logs...\n\n", v.logsSpinner.View()))
		b.WriteString("  " + NormalBorderStyle.Width(v.width-6).Render(v.logsVP.View()))
		b.WriteString("\n\n")
		b.WriteString(KeyHintStyle.Render("  scroll: up/down  esc: back to list"))
	}

	return lipgloss.NewStyle().Width(v.width).Height(v.height).Render(b.String())
}
