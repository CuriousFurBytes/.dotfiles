package main

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// terminalModel is a BubbleTea model that shows a styled header,
// then hands control to an external process via tea.ExecProcess.
type terminalModel struct {
	title   string
	cmd     *exec.Cmd
	done    bool
	err     error
	started bool
}

type terminalDoneMsg struct{ err error }

func newTerminalModel(title string, cmd *exec.Cmd) terminalModel {
	return terminalModel{
		title: title,
		cmd:   cmd,
	}
}

func (m terminalModel) Init() tea.Cmd {
	return nil
}

func (m terminalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.started {
			// First keypress launches the process
			m.started = true
			return m, tea.ExecProcess(m.cmd, func(err error) tea.Msg {
				return terminalDoneMsg{err: err}
			})
		}
		if m.done {
			return m, tea.Quit
		}
	case terminalDoneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	}
	return m, nil
}

func (m terminalModel) View() string {
	if m.done {
		var statusLine string
		if m.err != nil {
			statusLine = errorStyle.Render(fmt.Sprintf("  Process exited with error: %v", m.err))
		} else {
			statusLine = successStyle.Render("  Process completed successfully")
		}
		return terminalWindow(m.title, statusLine+"\n\n"+dimStyle.Render("  Press any key to continue..."))
	}

	if !m.started {
		content := fmt.Sprintf(
			"  %s\n\n  %s",
			subtitleStyle.Render("This will open an interactive session."),
			dimStyle.Render("Press any key to start..."),
		)
		return terminalWindow(m.title, content)
	}

	return ""
}

// RunInteractiveCommand opens a BubbleTea program that hands terminal control
// to an external command, with a styled window frame around it.
func RunInteractiveCommand(title string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	model := newTerminalModel(title, cmd)

	// Print styled header
	header := lipgloss.NewStyle().
		Background(colorSurface1).
		Foreground(colorText).
		Bold(true).
		Padding(0, 2).
		Width(72).
		Render("  " + title)
	fmt.Println()
	fmt.Println(header)
	fmt.Println()

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	final := finalModel.(terminalModel)
	if final.err != nil {
		return final.err
	}
	return nil
}
