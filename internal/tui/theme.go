package tui

import "github.com/charmbracelet/lipgloss"

// Docker-inspired color palette.
var (
	ColorDockerBlue = lipgloss.Color("#2496DC")
	ColorWhite      = lipgloss.Color("#FFFFFF")
	ColorDimGray    = lipgloss.Color("#626262")
	ColorDarkBg     = lipgloss.Color("#1A1A2E")
	ColorRed        = lipgloss.Color("#FF4444")
	ColorYellow     = lipgloss.Color("#FFCC00")
	ColorGreen      = lipgloss.Color("#00CC66")
	ColorCyan       = lipgloss.Color("#00BCD4")
	ColorMuted      = lipgloss.Color("#888888")
)

// Title and headings.
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorDockerBlue)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorDimGray)

	HeadingStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite)
)

// Navigation and tabs.
var (
	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorDockerBlue).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(ColorDimGray).
				Padding(0, 2)
)

// Status bar.
var (
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Background(ColorDarkBg).
			Padding(0, 1)

	StatusTextStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// Messages and feedback.
var (
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorYellow)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorCyan)
)

// Borders.
var (
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDockerBlue).
			Padding(1, 2)

	FocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorDockerBlue)

	NormalBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorDimGray)
)

// Tables.
var (
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorDockerBlue).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottomForeground(ColorDockerBlue).
				Padding(0, 1)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	SelectedRowStyle = lipgloss.NewStyle().
				Foreground(ColorWhite).
				Background(ColorDockerBlue).
				Bold(true).
				Padding(0, 1)
)

// Key hints shown in the footer.
var (
	KeyHintStyle = lipgloss.NewStyle().
			Foreground(ColorDimGray)

	KeyStyle = lipgloss.NewStyle().
			Foreground(ColorDockerBlue).
			Bold(true)
)

// Dashboard action button styles.
var (
	ActionBtnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDimGray).
			Padding(0, 2).
			Width(22).
			Align(lipgloss.Center)

	ActionBtnFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorDockerBlue).
				Foreground(ColorDockerBlue).
				Bold(true).
				Padding(0, 2).
				Width(22).
				Align(lipgloss.Center)
)

// Spinner/progress.
var (
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(ColorDockerBlue)

	ProgressBarFilled = lipgloss.NewStyle().
				Foreground(ColorDockerBlue)

	ProgressBarEmpty = lipgloss.NewStyle().
				Foreground(ColorDimGray)
)

// Logo renders the AppWrap ASCII title in Docker blue.
func Logo() string {
	logo := `
    _                __        __
   / \   _ __  _ __ \ \      / / __ __ _ _ __
  / _ \ | '_ \| '_ \ \ \ /\ / / '__/ _` + "`" + ` | '_ \
 / ___ \| |_) | |_) | \ V  V /| | | (_| | |_) |
/_/   \_\ .__/| .__/   \_/\_/ |_|  \__,_| .__/
        |_|   |_|                        |_|    `
	return TitleStyle.Render(logo)
}
