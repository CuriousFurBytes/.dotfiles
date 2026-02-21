// Package shader manages toggling the Ghostty custom shader for the duration
// of a command. It swaps the configured shader to just-snow.glsl on Start and
// restores the original on Stop by sending SIGUSR2 to the running Ghostty
// process.
package shader

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	configPath  = ".config/ghostty/config"
	snowShader  = "shaders/just-snow.glsl"
	shaderKey   = "custom-shader"
)

var shaderLine = regexp.MustCompile(`(?m)^custom-shader\s*=\s*(.+)$`)

// Session holds state needed to restore the original shader.
type Session struct {
	original string
	pid      string
}

// Start swaps the Ghostty shader to the snow shader and reloads Ghostty.
// Returns a Session that must be passed to Stop when done.
func Start() (*Session, error) {
	cfgPath := os.ExpandEnv("$HOME/") + configPath
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	content := string(data)
	match := shaderLine.FindStringSubmatch(content)
	original := ""
	if match != nil {
		original = strings.TrimSpace(match[1])
	}

	pid, err := ghosttyPID()
	if err != nil {
		// Ghostty not running — nothing to do, return a no-op session.
		return &Session{}, nil
	}

	if original != snowShader {
		updated := shaderLine.ReplaceAllString(content, shaderKey+" = "+snowShader)
		if err := os.WriteFile(cfgPath, []byte(updated), 0o644); err != nil {
			return nil, err
		}
		if err := reloadGhostty(pid); err != nil {
			// Non-fatal: shader just won't change.
			_ = os.WriteFile(cfgPath, data, 0o644)
			return &Session{}, nil
		}
	}

	return &Session{original: original, pid: pid}, nil
}

// Stop restores the original shader and reloads Ghostty.
func (s *Session) Stop() {
	if s == nil || s.pid == "" || s.original == snowShader {
		return
	}

	cfgPath := os.ExpandEnv("$HOME/") + configPath
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return
	}

	content := string(data)
	var updated string
	if s.original == "" {
		updated = shaderLine.ReplaceAllString(content, "")
	} else {
		updated = shaderLine.ReplaceAllString(content, shaderKey+" = "+s.original)
	}

	if err := os.WriteFile(cfgPath, []byte(updated), 0o644); err != nil {
		return
	}
	_ = reloadGhostty(s.pid)
}

func ghosttyPID() (string, error) {
	out, err := exec.Command("pgrep", "-f", "Applications/Ghostty.app").Output()
	if err != nil {
		return "", err
	}
	lines := strings.Fields(strings.TrimSpace(string(out)))
	if len(lines) == 0 {
		return "", os.ErrNotExist
	}
	return lines[0], nil
}

func reloadGhostty(pid string) error {
	return exec.Command("kill", "-USR2", pid).Run()
}
