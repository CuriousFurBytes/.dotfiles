package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"aitools/internal/shader"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── styles ────────────────────────────────────────────────────────────────────

var (
	accent = lipgloss.Color("105")

	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(1, 3)

	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(accent)

	styleSubtle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	styleSuccess = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))

	styleWarn = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))

	styleError = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))

	styleKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("238")).
			Padding(0, 1)

	styleLog = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

// ── state machine ─────────────────────────────────────────────────────────────

type phase int

const (
	phaseGenerating phase = iota
	phaseReady
	phaseConfirm
	phaseActing
	phaseResult
	phaseError
)

// ── messages ──────────────────────────────────────────────────────────────────

type generateDoneMsg struct {
	body string
	err  error
}

type editDoneMsg struct {
	content string
}

type actionDoneMsg struct {
	output string
	err    error
}

type tickMsg time.Time

// ── model ─────────────────────────────────────────────────────────────────────

type model struct {
	phase    phase
	spinner  spinner.Model
	viewport viewport.Model
	body     string
	tmpFile  string
	log      string
	err      error
	elapsed  time.Duration
	start    time.Time
	shader   *shader.Session
	width    int
	height   int
	ready    bool
}

func newModel(ss *shader.Session) model {
	sp := spinner.New()
	sp.Spinner = spinner.Points
	sp.Style = lipgloss.NewStyle().Foreground(accent)
	return model{
		spinner: sp,
		phase:   phaseGenerating,
		start:   time.Now(),
		shader:  ss,
	}
}

// ── init ──────────────────────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tick(), generatePR())
}

// ── update ────────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.phase == phaseReady || m.phase == phaseConfirm {
			m.viewport = makeViewport(m.width, m.height, m.body)
			m.ready = true
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tickMsg:
		m.elapsed = time.Since(m.start)
		if m.phase == phaseGenerating || m.phase == phaseActing {
			cmds = append(cmds, tick())
		}

	case generateDoneMsg:
		m.elapsed = time.Since(m.start)
		if msg.err != nil {
			m.phase = phaseError
			m.err = msg.err
			m.shader.Stop()
		} else {
			m.body = msg.body
			m.phase = phaseReady
			copyToClipboard(m.body)
			if m.width > 0 {
				m.viewport = makeViewport(m.width, m.height, m.body)
				m.ready = true
			}
		}
		return m, nil

	case editDoneMsg:
		if msg.content != "" {
			m.body = msg.content
			copyToClipboard(m.body)
		}
		if m.tmpFile != "" {
			os.Remove(m.tmpFile)
			m.tmpFile = ""
		}
		m.phase = phaseReady
		if m.width > 0 {
			m.viewport = makeViewport(m.width, m.height, m.body)
			m.ready = true
		}
		return m, nil

	case actionDoneMsg:
		m.log = msg.output
		if msg.err != nil {
			m.phase = phaseError
			m.err = msg.err
		} else {
			m.phase = phaseResult
		}
		return m, nil

	case tea.KeyMsg:
		switch m.phase {

		case phaseReady:
			switch msg.String() {
			case "q", "ctrl+c":
				m.shader.Stop()
				return m, tea.Quit
			case "r":
				m.ready = false
				m.phase = phaseGenerating
				m.start = time.Now()
				return m, tea.Batch(m.spinner.Tick, tick(), generatePR())
			case "e":
				return m, openEditor(m.body, &m.tmpFile)
			case "c":
				m.phase = phaseConfirm
				return m, nil
			}

		case phaseConfirm:
			switch msg.String() {
			case "y", "Y", "enter":
				m.phase = phaseActing
				m.start = time.Now()
				return m, tea.Batch(m.spinner.Tick, tick(), runCreatePR(m.body))
			case "n", "N", "esc", "ctrl+c":
				m.phase = phaseReady
				return m, nil
			case "q":
				m.shader.Stop()
				return m, tea.Quit
			}

		case phaseResult, phaseError:
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				m.shader.Stop()
				return m, tea.Quit
			}
		}
	}

	// forward scroll events to viewport when body is visible
	if (m.phase == phaseReady || m.phase == phaseConfirm) && m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// ── view ──────────────────────────────────────────────────────────────────────

func (m model) View() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render("aipr") + "\n\n")

	switch m.phase {

	case phaseGenerating:
		b.WriteString(m.spinner.View() + " generating PR description…\n")
		b.WriteString(styleSubtle.Render(fmt.Sprintf("%.1fs", m.elapsed.Seconds())))

	case phaseReady:
		if m.ready {
			b.WriteString(m.viewport.View() + "\n")
		} else {
			b.WriteString(m.body + "\n")
		}
		b.WriteString(styleSubtle.Render("─────────────────────────────────────\n"))
		b.WriteString(help("r", "regenerate", "e", "edit", "c", "create PR", "↑/↓", "scroll", "q", "quit"))

	case phaseConfirm:
		if m.ready {
			b.WriteString(m.viewport.View() + "\n")
		} else {
			b.WriteString(m.body + "\n")
		}
		b.WriteString(styleSubtle.Render("─────────────────────────────────────\n"))
		b.WriteString(styleWarn.Render("create pull request?") + "  ")
		b.WriteString(styleKey.Render("y") + styleSubtle.Render(" yes") + "  ")
		b.WriteString(styleKey.Render("n") + styleSubtle.Render(" no") + "\n")

	case phaseActing:
		b.WriteString(m.spinner.View() + " creating pull request…\n")
		b.WriteString(styleSubtle.Render(fmt.Sprintf("%.1fs", m.elapsed.Seconds())))

	case phaseResult:
		b.WriteString(styleSuccess.Render("✓ pull request created") + "\n\n")
		if m.log != "" {
			b.WriteString(styleLog.Render(m.log) + "\n\n")
		}
		b.WriteString(styleSubtle.Render("─────────────────────────────────────\n"))
		b.WriteString(help("q", "quit"))

	case phaseError:
		b.WriteString(styleError.Render("✗ error") + "\n\n")
		b.WriteString(m.err.Error() + "\n")
		if m.log != "" {
			b.WriteString("\n" + styleLog.Render(m.log) + "\n")
		}
		b.WriteString("\n" + styleSubtle.Render("q to quit"))
	}

	return styleBorder.Render(b.String()) + "\n"
}

// ── helpers ───────────────────────────────────────────────────────────────────

func help(pairs ...string) string {
	var b strings.Builder
	for i := 0; i+1 < len(pairs); i += 2 {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(styleKey.Render(pairs[i]))
		b.WriteString(" " + styleSubtle.Render(pairs[i+1]))
	}
	b.WriteString("\n")
	return b.String()
}

func makeViewport(w, h int, content string) viewport.Model {
	// Reserve rows: title(1) + blank(1) + divider(1) + help(1) + padding(~6) + border(2)
	vp := viewport.New(w-10, h-12)
	vp.SetContent(content)
	return vp
}

func copyToClipboard(s string) {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = bytes.NewBufferString(s)
	_ = cmd.Run()
}

// ── commands ──────────────────────────────────────────────────────────────────

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func generatePR() tea.Cmd {
	return func() tea.Msg {
		prTemplate := ""
		if data, err := os.ReadFile(".github/pull_request_template.md"); err == nil {
			prTemplate = string(data)
		}

		commits, err := gitOutput("git", "log", "origin/main..HEAD", "--oneline")
		if err != nil {
			return generateDoneMsg{err: fmt.Errorf("git log: %w", err)}
		}
		if strings.TrimSpace(commits) == "" {
			return generateDoneMsg{err: fmt.Errorf("no commits ahead of origin/main")}
		}

		changes, err := gitOutput("git", "diff", "origin/main...HEAD")
		if err != nil {
			return generateDoneMsg{err: fmt.Errorf("git diff: %w", err)}
		}

		var prompt strings.Builder
		prompt.WriteString("Generate a pull request description based on the commits and diff below.")
		if prTemplate != "" {
			prompt.WriteString("\n\n---TEMPLATE---\n")
			prompt.WriteString(prTemplate)
			prompt.WriteString("\n---END TEMPLATE---")
		}
		prompt.WriteString("\n\n---COMMITS---\n")
		prompt.WriteString(commits)
		prompt.WriteString("\n---END COMMITS---")
		prompt.WriteString("\n\n---DIFF---\n")
		prompt.WriteString(changes)
		prompt.WriteString("\n---END DIFF---")

		cmd := exec.Command("claude", "-p", prompt.String())
		out, err := cmd.Output()
		if err != nil {
			return generateDoneMsg{err: fmt.Errorf("claude: %w", err)}
		}

		return generateDoneMsg{body: strings.TrimSpace(string(out))}
	}
}

func openEditor(content string, tmpFile *string) tea.Cmd {
	f, err := os.CreateTemp("", "aipr-*.md")
	if err != nil {
		return func() tea.Msg { return editDoneMsg{} }
	}
	_, _ = f.WriteString(content)
	f.Close()
	*tmpFile = f.Name()
	path := f.Name()

	return tea.ExecProcess(exec.Command("hx", path), func(err error) tea.Msg {
		if err != nil {
			os.Remove(path)
			return editDoneMsg{}
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return editDoneMsg{}
		}
		return editDoneMsg{content: strings.TrimSpace(string(data))}
	})
}

func runCreatePR(body string) tea.Cmd {
	return func() tea.Msg {
		// Use first non-empty non-heading line as title
		title := ""
		for _, line := range strings.Split(body, "\n") {
			line = strings.TrimSpace(strings.TrimLeft(line, "# "))
			if line != "" {
				title = line
				break
			}
		}
		if title == "" {
			title = "chore: update"
		}

		cmd := exec.Command(
			"gh", "pr", "create",
			"--title", title,
			"--body", body,
			"--assignee", "@me",
		)
		out, err := cmd.CombinedOutput()
		return actionDoneMsg{output: strings.TrimSpace(string(out)), err: err}
	}
}

func gitOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	ss, err := shader.Start()
	if err != nil {
		ss = &shader.Session{}
	}

	p := tea.NewProgram(newModel(ss), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		ss.Stop()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
