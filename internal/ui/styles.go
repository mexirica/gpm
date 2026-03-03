// Package ui contains UI styles and components for the application.
package ui

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary   = lipgloss.Color("#7D56F4")
	ColorSecondary = lipgloss.Color("#6C6C6C")
	ColorSuccess   = lipgloss.Color("#04B575")
	ColorDanger    = lipgloss.Color("#FF4672")
	ColorWarning   = lipgloss.Color("#FFC107")
	ColorInfo      = lipgloss.Color("#00BCD4")
	ColorWhite     = lipgloss.Color("#FAFAFA")
	ColorDark      = lipgloss.Color("#1A1A2E")
	ColorMuted     = lipgloss.Color("#4A4A4A")

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorPrimary).
			Padding(0, 2).
			MarginBottom(1)

	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorPrimary).
			Padding(0, 2)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Background(lipgloss.Color("#2A2A3E")).
				Padding(0, 2)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Background(lipgloss.Color("#333346")).
			Padding(0, 1)

	InstalledBadge = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true).
			SetString("●")

	NotInstalledBadge = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				SetString("○")

	UpgradableBadge = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true).
			SetString("↑")

	PackageNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorWhite)

	PackageVersionStyle = lipgloss.NewStyle().
				Foreground(ColorInfo)

	PackageDescStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary)

	SelectedItemStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#2A2A5E")).
				Foreground(ColorWhite)

	DetailLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary)

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(ColorWhite)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	LogoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorPrimary).
			Padding(0, 1)

	LogoAccentStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWarning).
			Background(ColorPrimary)

	HeaderBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1E3A")).
			Foreground(ColorSecondary).
			Padding(0, 1)
)
