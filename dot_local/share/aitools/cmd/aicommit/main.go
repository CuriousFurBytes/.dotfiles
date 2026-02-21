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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── styles ────────────────────────────────────────────────────────────────────

var (
	accent = lipgloss.Color("63")

	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(1, 3)

	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(accent)

	styleSubtle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	styleCommit = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

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
	phaseGenerating phase = iota // running claude
	phaseReady                   // showing result, waiting for keypress
	phaseConfirm                 // "press y to confirm"
	phaseActing                  // running git commit / push
	phaseResult                  // showing outcome
	phaseError
)

type action int

const (
	actionCommit action = iota
	actionPush
)

// ── messages ──────────────────────────────────────────────────────────────────

type generateDoneMsg struct {
	commit string
	err    error
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
	phase   phase
	action  action
	spinner spinner.Model
	commit  string
	tmpFile string
	log     string
	err     error
	elapsed time.Duration
	start   time.Time
	shader  *shader.Session
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
	return tea.Batch(m.spinner.Tick, tick(), generateCommit())
}

// ── update ────────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tickMsg:
		m.elapsed = time.Since(m.start)
		if m.phase == phaseGenerating || m.phase == phaseActing {
			return m, tick()
		}

	case generateDoneMsg:
		m.elapsed = time.Since(m.start)
		if msg.err != nil {
			m.phase = phaseError
			m.err = msg.err
			m.shader.Stop()
		} else {
			m.commit = msg.commit
			m.phase = phaseReady
			copyToClipboard(m.commit)
		}
		return m, nil

	case editDoneMsg:
		if msg.content != "" {
			m.commit = msg.content
			copyToClipboard(m.commit)
		}
		if m.tmpFile != "" {
			os.Remove(m.tmpFile)
			m.tmpFile = ""
		}
		m.phase = phaseReady
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
				m.phase = phaseGenerating
				m.start = time.Now()
				return m, tea.Batch(m.spinner.Tick, tick(), generateCommit())
			case "e":
				return m, openEditor(m.commit, &m.tmpFile)
			case "c":
				m.action = actionCommit
				m.phase = phaseConfirm
				return m, nil
			case "P":
				m.action = actionPush
				m.phase = phaseConfirm
				return m, nil
			}

		case phaseConfirm:
			switch msg.String() {
			case "y", "Y", "enter":
				m.phase = phaseActing
				m.start = time.Now()
				if m.action == actionCommit {
					return m, tea.Batch(m.spinner.Tick, tick(), runCommit(m.commit))
				}
				return m, tea.Batch(m.spinner.Tick, tick(), runPush())
			case "n", "N", "esc", "ctrl+c":
				m.phase = phaseReady
				return m, nil
			case "q":
				m.shader.Stop()
				return m, tea.Quit
			}

		case phaseResult:
			switch msg.String() {
			case "q", "ctrl+c":
				m.shader.Stop()
				return m, tea.Quit
			case "P":
				m.action = actionPush
				m.phase = phaseConfirm
				return m, nil
			}

		case phaseError:
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				m.shader.Stop()
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

// ── view ──────────────────────────────────────────────────────────────────────

func (m model) View() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render("aicommit") + "\n\n")

	switch m.phase {

	case phaseGenerating:
		b.WriteString(m.spinner.View() + " generating commit message…\n")
		b.WriteString(styleSubtle.Render(fmt.Sprintf("%.1fs", m.elapsed.Seconds())))

	case phaseReady:
		b.WriteString(styleCommit.Render(m.commit) + "\n\n")
		b.WriteString(styleSubtle.Render("─────────────────────────────────────\n"))
		b.WriteString(help("r", "regenerate", "e", "edit", "c", "commit", "P", "push", "q", "quit"))

	case phaseConfirm:
		b.WriteString(styleCommit.Render(m.commit) + "\n\n")
		b.WriteString(styleSubtle.Render("─────────────────────────────────────\n"))
		action := "commit"
		if m.action == actionPush {
			action = "push to upstream"
		}
		b.WriteString(styleWarn.Render(fmt.Sprintf("confirm %s?", action)) + "  ")
		b.WriteString(styleKey.Render("y") + styleSubtle.Render(" yes") + "  ")
		b.WriteString(styleKey.Render("n") + styleSubtle.Render(" no") + "\n")

	case phaseActing:
		action := "committing…"
		if m.action == actionPush {
			action = "pushing…"
		}
		b.WriteString(m.spinner.View() + " " + action + "\n")
		b.WriteString(styleSubtle.Render(fmt.Sprintf("%.1fs", m.elapsed.Seconds())))

	case phaseResult:
		b.WriteString(styleSuccess.Render("✓ done") + "\n\n")
		if m.log != "" {
			b.WriteString(styleLog.Render(m.log) + "\n\n")
		}
		b.WriteString(styleSubtle.Render("─────────────────────────────────────\n"))
		b.WriteString(help("P", "push", "q", "quit"))

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

func copyToClipboard(s string) {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = bytes.NewBufferString(s)
	_ = cmd.Run()
}

// ── commands ──────────────────────────────────────────────────────────────────

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func generateCommit() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("git", "diff", "--staged").Output()
		if err != nil {
			return generateDoneMsg{err: fmt.Errorf("git diff: %w", err)}
		}
		diff := string(out)
		if strings.TrimSpace(diff) == "" {
			return generateDoneMsg{err: fmt.Errorf("no staged changes")}
		}

		const prompt = `Generate a conventional commit message for my staged changes: <type>(<scope>): <subject> (<=72ch). Types: feat|fix|docs|style|refactor|perf|test|chore|build. Use imperative mood. Use list format for body (<=72ch, max 5 items, start each with -). Do not include the string Co-Authored-By. Output only the raw commit message, with no markdown, no code blocks, no backticks, no explanations.`

		cmd := exec.Command("claude", "-p", prompt, "--model", "haiku", "--output-format", "text")
		cmd.Stdin = strings.NewReader(diff)
		result, err := cmd.Output()
		if err != nil {
			return generateDoneMsg{err: fmt.Errorf("claude: %w", err)}
		}

		return generateDoneMsg{commit: strings.TrimSpace(string(result))}
	}
}

func openEditor(content string, tmpFile *string) tea.Cmd {
	f, err := os.CreateTemp("", "aicommit-*.txt")
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

func runCommit(msg string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "commit", "-m", msg)
		out, err := cmd.CombinedOutput()
		return actionDoneMsg{output: strings.TrimSpace(string(out)), err: err}
	}
}

func runPush() tea.Cmd {
	return func() tea.Msg {
		branch, err := currentBranch()
		if err != nil {
			return actionDoneMsg{err: fmt.Errorf("get branch: %w", err)}
		}
		cmd := exec.Command("git", "push", "origin", branch)
		out, err := cmd.CombinedOutput()
		return actionDoneMsg{output: strings.TrimSpace(string(out)), err: err}
	}
}

func currentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
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
