package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Catppuccin Mocha palette
var (
	colorRosewater = lipgloss.Color("#f5e0dc")
	colorFlamingo  = lipgloss.Color("#f2cdcd")
	colorPink      = lipgloss.Color("#f5c2e7")
	colorMauve     = lipgloss.Color("#cba6f7")
	colorRed       = lipgloss.Color("#f38ba8")
	colorMaroon    = lipgloss.Color("#eba0ac")
	colorPeach     = lipgloss.Color("#fab387")
	colorYellow    = lipgloss.Color("#f9e2af")
	colorGreen     = lipgloss.Color("#a6e3a1")
	colorTeal      = lipgloss.Color("#94e2d5")
	colorSky       = lipgloss.Color("#89dceb")
	colorSapphire  = lipgloss.Color("#74c7ec")
	colorBlue      = lipgloss.Color("#89b4fa")
	colorLavender  = lipgloss.Color("#b4befe")
	colorText      = lipgloss.Color("#cdd6f4")
	colorSubtext1  = lipgloss.Color("#bac2de")
	colorSubtext0  = lipgloss.Color("#a6adc8")
	colorOverlay2  = lipgloss.Color("#9399b2")
	colorOverlay1  = lipgloss.Color("#7f849c")
	colorOverlay0  = lipgloss.Color("#6c7086")
	colorSurface2  = lipgloss.Color("#585b70")
	colorSurface1  = lipgloss.Color("#45475a")
	colorSurface0  = lipgloss.Color("#313244")
	colorBase      = lipgloss.Color("#1e1e2e")
	colorMantle    = lipgloss.Color("#181825")
	colorCrust     = lipgloss.Color("#11111b")
)

// Reusable styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorLavender)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorSubtext1)

	successStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorYellow)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	infoStyle = lipgloss.NewStyle().
			Foreground(colorBlue)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorOverlay0)

	boldStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText)
)

// Panel renders a bordered panel with a title
func styledPanel(title, content string, borderColor lipgloss.Color) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(72).
		Render(
			titleStyle.Render(title) + "\n\n" + content,
		)
}

// Terminal window style panel (like a TUI app window)
func terminalWindow(title, content string) string {
	titleBar := lipgloss.NewStyle().
		Background(colorSurface1).
		Foreground(colorText).
		Bold(true).
		Padding(0, 2).
		Width(72).
		Render("  " + title)

	body := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSurface2).
		BorderTop(false).
		Padding(1, 2).
		Width(72).
		Render(content)

	return titleBar + "\n" + body
}

// Status badges
func statusOK(name string) string {
	badge := successStyle.Render("[ok]")
	return fmt.Sprintf("  %s %s", badge, name)
}

func statusDone(name string) string {
	badge := infoStyle.Render("[done]")
	return fmt.Sprintf("  %s %s", badge, name)
}

func statusSkip(name string) string {
	badge := warningStyle.Render("[skip]")
	return fmt.Sprintf("  %s %s", badge, name)
}

func statusFail(name string) string {
	badge := errorStyle.Render("[fail]")
	return fmt.Sprintf("  %s %s", badge, name)
}

func statusInstalling(name string) string {
	badge := lipgloss.NewStyle().Foreground(colorMauve).Render("[...]")
	return fmt.Sprintf("  %s %s", badge, name)
}

// Section header
func sectionHeader(title string) string {
	line := lipgloss.NewStyle().
		Foreground(colorSurface2).
		Render("─────────────────────────────────────────────────────────────────")
	header := titleStyle.Render(title)
	return "\n" + line + "\n  " + header + "\n" + line + "\n"
}

// Welcome banner
func welcomeBanner(osName, hostname, user string) string {
	logo := lipgloss.NewStyle().
		Foreground(colorMauve).
		Bold(true).
		Render(`
    ╔══════════════════════════════════════╗
    ║     ·  Dotfiles Installer  ·         ║
    ╚══════════════════════════════════════╝`)

	info := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s",
		boldStyle.Render("   OS:"), osName,
		boldStyle.Render(" Host:"), hostname,
		boldStyle.Render(" User:"), user,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(colorLavender).
		Padding(1, 3).
		Width(72).
		Render(logo + "\n\n" + info)
}

// Summary panel
func summaryPanel(installed, alreadyOK, skipped, failed int, nextSteps string) string {
	counters := fmt.Sprintf(
		"%s %d  %s %d  %s %d  %s %d",
		successStyle.Render("●"), installed,
		infoStyle.Render("●"), alreadyOK,
		warningStyle.Render("●"), skipped,
		errorStyle.Render("●"), failed,
	)

	legend := fmt.Sprintf(
		"%s  %s  %s  %s",
		successStyle.Render("● Installed"),
		infoStyle.Render("● Already OK"),
		warningStyle.Render("● Skipped"),
		errorStyle.Render("● Failed"),
	)

	content := counters + "\n" + legend

	if nextSteps != "" {
		content += "\n\n" + boldStyle.Render("Next Steps:") + "\n" + nextSteps
	}

	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(colorGreen).
		Padding(1, 2).
		Width(72).
		Render(
			titleStyle.Render("Installation Complete") + "\n\n" + content,
		)
}
