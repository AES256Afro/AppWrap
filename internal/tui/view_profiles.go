package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/theencryptedafro/appwrap/internal/service"
)

// profilesLoadedMsg is sent after listing profiles.
type profilesLoadedMsg struct {
	profiles []service.ProfileSummary
	err      error
}

// profileContentMsg delivers the raw file content for detail view.
type profileContentMsg struct {
	content string
	err     error
}

// profileDeletedMsg indicates the profile was deleted.
type profileDeletedMsg struct {
	err error
}

type profilesMode int

const (
	profilesModeList profilesMode = iota
	profilesModeDetail
)

type profilesView struct {
	svc    *service.AppService
	keys   KeyMap
	width  int
	height int

	mode     profilesMode
	profiles []service.ProfileSummary
	cursor   int
	err      error

	// Detail view.
	detailVP      viewport.Model
	detailContent string
}

func newProfilesView(svc *service.AppService) *profilesView {
	vp := viewport.New(80, 20)
	return &profilesView{
		svc:      svc,
		keys:     DefaultKeyMap(),
		mode:     profilesModeList,
		detailVP: vp,
	}
}

func (v *profilesView) Title() string { return "Profiles" }

func (v *profilesView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.detailVP.Width = w - 6
	v.detailVP.Height = h - 8
	if v.detailVP.Height < 5 {
		v.detailVP.Height = 5
	}
}

func (v *profilesView) Init() tea.Cmd {
	return v.loadProfiles
}

func (v *profilesView) loadProfiles() tea.Msg {
	profiles, err := v.svc.ListProfiles("")
	return profilesLoadedMsg{profiles: profiles, err: err}
}

func (v *profilesView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case profilesLoadedMsg:
		v.profiles = msg.profiles
		v.err = msg.err
		if v.cursor >= len(v.profiles) {
			v.cursor = len(v.profiles) - 1
		}
		if v.cursor < 0 {
			v.cursor = 0
		}
		return v, nil

	case profileContentMsg:
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.detailContent = msg.content
		v.detailVP.SetContent(msg.content)
		v.detailVP.GotoTop()
		v.mode = profilesModeDetail
		return v, nil

	case profileDeletedMsg:
		if msg.err != nil {
			v.err = msg.err
		}
		return v, v.loadProfiles

	case tea.KeyMsg:
		switch v.mode {
		case profilesModeList:
			return v.updateList(msg)
		case profilesModeDetail:
			return v.updateDetail(msg)
		}
	}

	return v, nil
}

func (v *profilesView) updateList(msg tea.KeyMsg) (View, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Back):
		return v, func() tea.Msg { return PopViewMsg{} }
	case msg.String() == "up" || msg.String() == "k":
		if v.cursor > 0 {
			v.cursor--
		}
	case msg.String() == "down" || msg.String() == "j":
		if v.cursor < len(v.profiles)-1 {
			v.cursor++
		}
	case key.Matches(msg, v.keys.Enter):
		if len(v.profiles) > 0 {
			return v, v.loadProfileContent(v.profiles[v.cursor].Path)
		}
	case msg.String() == "d":
		if len(v.profiles) > 0 {
			path := v.profiles[v.cursor].Path
			return v, func() tea.Msg {
				err := v.svc.DeleteProfile(path)
				return profileDeletedMsg{err: err}
			}
		}
	case msg.String() == "b":
		if len(v.profiles) > 0 {
			bv := newBuildView(v.svc)
			bv.profileInput.SetValue(v.profiles[v.cursor].Path)
			return v, func() tea.Msg { return PushViewMsg{View: bv} }
		}
	}
	return v, nil
}

func (v *profilesView) updateDetail(msg tea.KeyMsg) (View, tea.Cmd) {
	if key.Matches(msg, v.keys.Back) {
		v.mode = profilesModeList
		return v, nil
	}
	var cmd tea.Cmd
	v.detailVP, cmd = v.detailVP.Update(msg)
	return v, cmd
}

func (v *profilesView) loadProfileContent(path string) tea.Cmd {
	return func() tea.Msg {
		data, err := os.ReadFile(path)
		if err != nil {
			return profileContentMsg{err: err}
		}
		return profileContentMsg{content: string(data)}
	}
}

func (v *profilesView) View() string {
	var b strings.Builder

	switch v.mode {
	case profilesModeList:
		b.WriteString("\n")
		b.WriteString(HeadingStyle.Render("  Profiles"))
		b.WriteString("\n\n")

		if v.err != nil {
			b.WriteString("  " + ErrorStyle.Render(v.err.Error()) + "\n")
		}

		if len(v.profiles) == 0 {
			b.WriteString(SubtitleStyle.Render("  No profiles found in the current directory.\n"))
			b.WriteString(SubtitleStyle.Render("  Use [S]can to generate one.\n"))
		} else {
			// Table header.
			header := fmt.Sprintf("  %-24s %-20s %-22s %-8s", "Name", "App", "Strategy", "Arch")
			b.WriteString(TableHeaderStyle.Render(header) + "\n")

			for i, p := range v.profiles {
				line := fmt.Sprintf("  %-24s %-20s %-22s %-8s", p.Name, p.AppName, p.Strategy, p.Arch)
				if i == v.cursor {
					line = SelectedRowStyle.Width(v.width - 4).Render(
						fmt.Sprintf(" %-24s %-20s %-22s %-8s", p.Name, p.AppName, p.Strategy, p.Arch),
					)
					line = "  " + line
				}
				b.WriteString(line + "\n")
			}
		}

		b.WriteString("\n")
		b.WriteString(KeyHintStyle.Render("  enter: view  d: delete  b: build  esc: back"))

	case profilesModeDetail:
		b.WriteString("\n")
		if v.cursor < len(v.profiles) {
			b.WriteString(HeadingStyle.Render(fmt.Sprintf("  Profile: %s", v.profiles[v.cursor].Name)))
		} else {
			b.WriteString(HeadingStyle.Render("  Profile Detail"))
		}
		b.WriteString("\n\n")
		b.WriteString("  " + NormalBorderStyle.Width(v.width-6).Render(v.detailVP.View()))
		b.WriteString("\n\n")
		b.WriteString(KeyHintStyle.Render("  scroll: up/down  esc: back to list"))
	}

	return lipgloss.NewStyle().Width(v.width).Height(v.height).Render(b.String())
}
