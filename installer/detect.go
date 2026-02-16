package main

import (
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
)

type OSInfo struct {
	// "darwin", "ubuntu", "fedora", "pop_os", "linux"
	Target   string
	// Human-readable name
	Name     string
	Hostname string
	User     string
}

func detectOS() OSInfo {
	info := OSInfo{}

	// Get user
	if u, err := user.Current(); err == nil {
		info.User = u.Username
	} else {
		info.User = os.Getenv("USER")
	}

	// Get hostname
	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}

	if runtime.GOOS == "darwin" {
		info.Target = "darwin"
		info.Name = "macOS"
		return info
	}

	// Linux: read /etc/os-release
	info.Name = "Linux"
	info.Target = "linux"

	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return info
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "ID=") {
			id := strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			info.Target = id
			switch id {
			case "ubuntu":
				info.Name = "Ubuntu"
			case "fedora":
				info.Name = "Fedora"
			case "pop":
				info.Target = "pop_os"
				info.Name = "Pop!_OS"
			}
		}
	}

	return info
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCmdSilent(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runShell(command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runShellSilent(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
