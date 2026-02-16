package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/huh/spinner"
)

const repoURL = "https://github.com/CuriousFurBytes/.dotfiles.git"

// App is the main installer application
type App struct {
	osInfo    OSInfo
	sourceDir string
	catalog   *PackageCatalog
	installer *PackageInstaller
	results   []InstallResult
	selected  map[string]bool
}

func NewApp(sourceDir string) *App {
	return &App{
		sourceDir: sourceDir,
		results:   []InstallResult{},
	}
}

func (a *App) Run() error {
	// ── Step 1: Welcome ────────────────────────────────────────────
	a.osInfo = detectOS()
	fmt.Println(welcomeBanner(a.osInfo.Name, a.osInfo.Hostname, a.osInfo.User))
	fmt.Println()

	a.installer = NewPackageInstaller(a.osInfo.Target)

	// ── Step 2: Load & select packages ─────────────────────────────
	catalog, err := LoadPackages(a.sourceDir)
	if err != nil {
		fmt.Println(statusFail(fmt.Sprintf("Failed to load packages.json: %v", err)))
		return err
	}
	a.catalog = catalog

	targetPkgs := catalog.FilterForTarget(a.osInfo.Target)
	categories := categorizePackages(targetPkgs)

	selectedMap := make(map[string]*[]string)
	form := BuildPackageSelectionForm(categories, selectedMap)
	if err := form.Run(); err != nil {
		return fmt.Errorf("package selection cancelled: %w", err)
	}
	a.selected = CollectSelectedPackages(selectedMap)

	fmt.Println()
	fmt.Println(statusDone(fmt.Sprintf("Selected %d packages", len(a.selected))))
	fmt.Println()

	// ── Step 3: Install chezmoi ────────────────────────────────────
	if err := a.stepInstallChezmoi(); err != nil {
		return err
	}

	// ── Step 4: Install Proton Pass + CLI ──────────────────────────
	if err := a.stepInstallProtonPass(); err != nil {
		return err
	}

	// ── Step 5: Proton Pass CLI login ──────────────────────────────
	if err := a.stepProtonPassLogin(); err != nil {
		return err
	}

	// ── Step 6: chezmoi init ───────────────────────────────────────
	if err := a.stepChezmoiInit(); err != nil {
		return err
	}

	// ── Step 7: chezmoi apply ──────────────────────────────────────
	if err := a.stepChezmoiApply(); err != nil {
		return err
	}

	// ── Step 8: gh auth login ──────────────────────────────────────
	if err := a.stepGhLogin(); err != nil {
		return err
	}

	// ── Step 9: Install gh-dash ────────────────────────────────────
	if err := a.stepInstallGhDash(); err != nil {
		return err
	}

	// ── Step 10: Install selected packages ─────────────────────────
	if err := a.stepInstallPackages(); err != nil {
		return err
	}

	// ── Step 11: Summary ───────────────────────────────────────────
	a.showSummary()

	return nil
}

func (a *App) stepInstallChezmoi() error {
	fmt.Println(sectionHeader("Chezmoi"))

	if commandExists("chezmoi") {
		fmt.Println(statusOK("chezmoi already installed"))
		return nil
	}

	confirmed, err := ConfirmStep(
		"Install chezmoi?",
		"chezmoi is required to manage your dotfiles.",
	)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println(statusSkip("chezmoi"))
		return nil
	}

	var installErr error
	_ = spinner.New().
		Title("Installing chezmoi...").
		Action(func() {
			if a.osInfo.Target == "darwin" {
				_, installErr = runShellSilent("brew install chezmoi")
			} else {
				_, installErr = runShellSilent(`sh -c "$(curl -fsLS get.chezmoi.io)" -- -b "$HOME/.local/bin"`)
				if installErr == nil {
					os.Setenv("PATH", filepath.Join(os.Getenv("HOME"), ".local", "bin")+":"+os.Getenv("PATH"))
				}
			}
		}).
		Run()

	if installErr != nil {
		fmt.Println(statusFail("chezmoi"))
		return fmt.Errorf("failed to install chezmoi: %w", installErr)
	}
	fmt.Println(statusDone("chezmoi"))
	return nil
}

func (a *App) stepInstallProtonPass() error {
	fmt.Println(sectionHeader("Proton Pass"))

	// Install proton-pass (GUI app)
	if a.osInfo.Target == "darwin" {
		if !commandExists("pass-cli") || !a.installer.IsInstalled("proton-pass", InstallMethod{Cask: "proton-pass"}) {
			confirmed, err := ConfirmStep(
				"Install Proton Pass?",
				"Proton Pass is used for secrets management.",
			)
			if err != nil {
				return err
			}
			if confirmed {
				var installErr error
				_ = spinner.New().
					Title("Installing Proton Pass...").
					Action(func() {
						if !a.installer.IsInstalled("proton-pass", InstallMethod{Cask: "proton-pass"}) {
							_, installErr = runShellSilent("brew install --cask proton-pass")
						}
					}).
					Run()
				if installErr != nil {
					fmt.Println(statusFail("proton-pass"))
				} else {
					fmt.Println(statusDone("proton-pass"))
				}
			} else {
				fmt.Println(statusSkip("proton-pass"))
			}
		} else {
			fmt.Println(statusOK("proton-pass"))
		}
	}

	// Install proton-pass-cli
	if commandExists("pass-cli") {
		fmt.Println(statusOK("proton-pass-cli"))
		return nil
	}

	confirmed, err := ConfirmStep(
		"Install Proton Pass CLI?",
		"The CLI is used by chezmoi to retrieve secrets.",
	)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println(statusSkip("proton-pass-cli"))
		return nil
	}

	var installErr error
	_ = spinner.New().
		Title("Installing Proton Pass CLI...").
		Action(func() {
			if a.osInfo.Target == "darwin" {
				runShellSilent("brew tap protonpass/tap")
				_, installErr = runShellSilent("brew install protonpass/tap/pass-cli")
			} else {
				_, installErr = runShellSilent("curl -fsSL https://proton.me/download/pass-cli/install.sh | bash")
			}
		}).
		Run()

	if installErr != nil {
		fmt.Println(statusFail("proton-pass-cli"))
		return fmt.Errorf("failed to install proton-pass-cli: %w", installErr)
	}
	fmt.Println(statusDone("proton-pass-cli"))
	return nil
}

func (a *App) stepProtonPassLogin() error {
	fmt.Println(sectionHeader("Proton Pass Authentication"))

	// Check if already authenticated
	if _, err := runShellSilent("pass-cli vault list"); err == nil {
		fmt.Println(statusOK("proton-pass-cli authenticated"))
		return nil
	}

	if !commandExists("pass-cli") {
		fmt.Println(statusSkip("pass-cli not installed"))
		return nil
	}

	confirmed, err := ConfirmStep(
		"Login to Proton Pass CLI?",
		"This will open an interactive session for authentication.",
	)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println(statusSkip("proton-pass-cli login"))
		return nil
	}

	if err := RunInteractiveCommand("Proton Pass CLI Login", "pass-cli", "login"); err != nil {
		fmt.Println(statusFail("proton-pass-cli login"))
		// Don't return error — user can continue without auth
		return nil
	}
	fmt.Println(statusDone("proton-pass-cli login"))

	// Start SSH agent after successful login
	if err := a.stepProtonPassSSHAgent(); err != nil {
		return err
	}

	return nil
}

func (a *App) stepProtonPassSSHAgent() error {
	fmt.Println(sectionHeader("Proton Pass SSH Agent"))

	// Check if already running
	socketPath := filepath.Join(os.Getenv("HOME"), ".ssh", "proton-pass-agent.sock")
	if _, err := os.Stat(socketPath); err == nil {
		fmt.Println(statusOK("SSH agent socket already exists"))
		os.Setenv("SSH_AUTH_SOCK", socketPath)
		return nil
	}

	if !commandExists("pass-cli") {
		fmt.Println(statusSkip("pass-cli not installed"))
		return nil
	}

	// Verify authentication before trying
	if _, err := runShellSilent("pass-cli vault list"); err != nil {
		fmt.Println(statusSkip("pass-cli not authenticated"))
		return nil
	}

	confirmed, err := ConfirmStep(
		"Start Proton Pass SSH Agent?",
		"This will start pass-cli as an SSH agent, loading keys from the \"SSH\" vault.\nThe agent socket will be at ~/.ssh/proton-pass-agent.sock",
	)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println(statusSkip("proton-pass ssh-agent"))
		return nil
	}

	// Ensure directories exist
	home := os.Getenv("HOME")
	os.MkdirAll(filepath.Join(home, ".ssh"), 0o700)
	os.MkdirAll(filepath.Join(home, ".local", "state"), 0o755)

	var agentErr error

	if a.osInfo.Target == "darwin" {
		// Register as launchd service
		plistDir := filepath.Join(home, "Library", "LaunchAgents")
		os.MkdirAll(plistDir, 0o755)
		plistPath := filepath.Join(plistDir, "me.proton.pass.ssh-agent.plist")

		passCliPath, _ := exec.LookPath("pass-cli")

		plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>me.proton.pass.ssh-agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>ssh-agent</string>
        <string>start</string>
        <string>--vault-name</string>
        <string>SSH</string>
        <string>--socket-path</string>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s/.local/state/proton-pass-ssh-agent.log</string>
    <key>StandardErrorPath</key>
    <string>%s/.local/state/proton-pass-ssh-agent.log</string>
</dict>
</plist>`, passCliPath, socketPath, home, home)

		_ = spinner.New().
			Title("Registering Proton Pass SSH Agent with launchd...").
			Action(func() {
				if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
					agentErr = err
					return
				}
				// Unload old version if present
				runShellSilent(fmt.Sprintf(`launchctl bootout "gui/$(id -u)/me.proton.pass.ssh-agent"`))
				_, agentErr = runShellSilent(fmt.Sprintf(`launchctl bootstrap "gui/$(id -u)" "%s"`, plistPath))
			}).
			Run()
	} else {
		// Register as systemd user service
		systemdDir := filepath.Join(home, ".config", "systemd", "user")
		os.MkdirAll(systemdDir, 0o755)

		passCliPath, _ := exec.LookPath("pass-cli")

		unit := fmt.Sprintf(`[Unit]
Description=Proton Pass SSH Agent
After=network-online.target

[Service]
Type=simple
ExecStart=%s ssh-agent start --vault-name SSH --socket-path %s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`, passCliPath, socketPath)

		_ = spinner.New().
			Title("Registering Proton Pass SSH Agent with systemd...").
			Action(func() {
				unitPath := filepath.Join(systemdDir, "proton-pass-ssh-agent.service")
				if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil {
					agentErr = err
					return
				}
				runShellSilent("systemctl --user daemon-reload")
				_, agentErr = runShellSilent("systemctl --user enable --now proton-pass-ssh-agent.service")
			}).
			Run()
	}

	if agentErr != nil {
		fmt.Println(statusFail(fmt.Sprintf("proton-pass ssh-agent: %v", agentErr)))
		return nil // non-fatal
	}

	// Wait for the socket to appear
	for i := 0; i < 15; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	os.Setenv("SSH_AUTH_SOCK", socketPath)
	fmt.Println(statusDone("proton-pass ssh-agent (registered as system service)"))
	fmt.Println(dimStyle.Render(fmt.Sprintf("    SSH_AUTH_SOCK=%s", socketPath)))
	fmt.Println(dimStyle.Render("    Starts automatically at login"))

	return nil
}

func (a *App) stepChezmoiInit() error {
	fmt.Println(sectionHeader("Chezmoi Init"))

	chezmoiDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "chezmoi")
	if info, err := os.Stat(filepath.Join(chezmoiDir, ".git")); err == nil && info.IsDir() {
		fmt.Println(statusOK("dotfiles already initialized"))
		return nil
	}

	confirmed, err := ConfirmStep(
		"Initialize chezmoi?",
		fmt.Sprintf("This will clone %s", repoURL),
	)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println(statusSkip("chezmoi init"))
		return nil
	}

	var initErr error
	_ = spinner.New().
		Title("Initializing chezmoi...").
		Action(func() {
			_, initErr = runShellSilent(fmt.Sprintf("chezmoi init %s", repoURL))
		}).
		Run()

	if initErr != nil {
		fmt.Println(statusFail("chezmoi init"))
		return fmt.Errorf("chezmoi init failed: %w", initErr)
	}
	fmt.Println(statusDone("chezmoi init"))
	return nil
}

func (a *App) stepChezmoiApply() error {
	fmt.Println(sectionHeader("Chezmoi Apply"))

	confirmed, err := ConfirmStep(
		"Apply dotfiles with chezmoi?",
		"This will apply all dotfiles and run configuration scripts.\nThe output will be shown in an interactive terminal.",
	)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println(statusSkip("chezmoi apply"))
		return nil
	}

	if err := RunInteractiveCommand("chezmoi apply -v", "chezmoi", "apply", "-v"); err != nil {
		fmt.Println(statusFail("chezmoi apply"))
		// Don't fail entirely — user may want to continue
	} else {
		fmt.Println(statusDone("chezmoi apply"))
	}
	return nil
}

func (a *App) stepGhLogin() error {
	fmt.Println(sectionHeader("GitHub CLI"))

	if !commandExists("gh") {
		fmt.Println(statusSkip("gh not installed"))
		return nil
	}

	// Check if already authenticated
	if _, err := runShellSilent("gh auth status"); err == nil {
		fmt.Println(statusOK("gh already authenticated"))
		return nil
	}

	confirmed, err := ConfirmStep(
		"Login to GitHub CLI?",
		"This will open an interactive session for GitHub authentication.",
	)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println(statusSkip("gh auth login"))
		return nil
	}

	if err := RunInteractiveCommand("GitHub CLI Login", "gh", "auth", "login"); err != nil {
		fmt.Println(statusFail("gh auth login"))
	} else {
		fmt.Println(statusDone("gh auth login"))
	}
	return nil
}

func (a *App) stepInstallGhDash() error {
	fmt.Println(sectionHeader("GitHub Dashboard"))

	if !commandExists("gh") {
		fmt.Println(statusSkip("gh not installed"))
		return nil
	}

	// Check if gh-dash is already installed
	if out, _ := runShellSilent("gh extension list"); strings.Contains(out, "gh-dash") {
		fmt.Println(statusOK("gh-dash already installed"))
		return nil
	}

	// Check if gh is authenticated
	if _, err := runShellSilent("gh auth status"); err != nil {
		fmt.Println(statusSkip("gh not authenticated"))
		return nil
	}

	confirmed, err := ConfirmStep(
		"Install gh-dash?",
		"GitHub CLI dashboard extension by dlvhdr.",
	)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println(statusSkip("gh-dash"))
		return nil
	}

	var installErr error
	_ = spinner.New().
		Title("Installing gh-dash...").
		Action(func() {
			_, installErr = runShellSilent("gh extension install dlvhdr/gh-dash")
		}).
		Run()

	if installErr != nil {
		fmt.Println(statusFail("gh-dash"))
	} else {
		fmt.Println(statusDone("gh-dash"))
	}
	return nil
}

func (a *App) stepInstallPackages() error {
	fmt.Println(sectionHeader("Package Installation"))

	targetPkgs := a.catalog.FilterForTarget(a.osInfo.Target)

	// Filter to only selected packages
	var toInstall []Package
	for _, pkg := range targetPkgs {
		if a.selected[pkg.Name] {
			toInstall = append(toInstall, pkg)
		}
	}

	if len(toInstall) == 0 {
		fmt.Println(statusSkip("no packages selected"))
		return nil
	}

	confirmed, err := ConfirmStep(
		fmt.Sprintf("Install %d packages?", len(toInstall)),
		"This will install all selected packages using their respective package managers.",
	)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println(statusSkip("package installation"))
		return nil
	}

	// Phase 1: Brew taps
	if commandExists("brew") && len(a.catalog.BrewTaps) > 0 {
		fmt.Println()
		fmt.Println(boldStyle.Render("  Homebrew Taps"))
		var tapErr error
		_ = spinner.New().
			Title("Adding brew taps...").
			Action(func() {
				for _, tap := range a.catalog.BrewTaps {
					if _, err := runShellSilent(fmt.Sprintf("brew tap %s", tap)); err != nil {
						tapErr = err
					}
				}
			}).
			Run()
		if tapErr != nil {
			fmt.Println(statusFail("some taps failed"))
		} else {
			for _, tap := range a.catalog.BrewTaps {
				fmt.Println(statusOK(tap))
			}
		}
	}

	// Update package lists for Linux
	if a.osInfo.Target != "darwin" {
		_ = spinner.New().
			Title("Updating package lists...").
			Action(func() {
				switch a.osInfo.Target {
				case "ubuntu", "pop_os":
					runShellSilent("sudo apt update")
				case "fedora":
					runShellSilent("sudo dnf check-update")
				}
			}).
			Run()
	}

	// Phase 2: Batch system packages
	fmt.Println()
	fmt.Println(boldStyle.Render("  System Packages"))

	var brewFormulas, casks, aptPkgs, dnfPkgs []string
	var brewNames, caskNames, aptNames, dnfNames []string
	var alreadyInstalled []Package

	_ = spinner.New().
		Title("Checking installed packages...").
		Action(func() {
			for _, pkg := range toInstall {
				method := pkg.Packages[a.osInfo.Target]
				if !method.IsSystemMethod() {
					continue
				}
				if a.installer.IsInstalled(pkg.Name, method) {
					alreadyInstalled = append(alreadyInstalled, pkg)
					a.results = append(a.results, InstallResult{Name: pkg.Name, Method: method.MethodName(), Status: "ok"})
					continue
				}
				switch method.MethodName() {
				case "brew":
					brewFormulas = append(brewFormulas, method.Brew)
					brewNames = append(brewNames, pkg.Name)
				case "cask":
					casks = append(casks, method.Cask)
					caskNames = append(caskNames, pkg.Name)
				case "apt":
					aptPkgs = append(aptPkgs, method.Apt)
					aptNames = append(aptNames, pkg.Name)
				case "dnf":
					dnfPkgs = append(dnfPkgs, method.Dnf)
					dnfNames = append(dnfNames, pkg.Name)
				}
			}
		}).
		Run()

	for _, pkg := range alreadyInstalled {
		fmt.Println(statusOK(pkg.Name))
	}

	// Batch install brew formulas
	if len(brewFormulas) > 0 {
		var installErr error
		_ = spinner.New().
			Title(fmt.Sprintf("Installing %d brew formulas...", len(brewFormulas))).
			Action(func() {
				installErr = a.installer.BatchInstallBrew(brewFormulas)
			}).
			Run()
		for _, name := range brewNames {
			if installErr != nil {
				fmt.Println(statusFail(name))
				a.results = append(a.results, InstallResult{Name: name, Method: "brew", Status: "fail"})
			} else {
				fmt.Println(statusDone(name))
				a.results = append(a.results, InstallResult{Name: name, Method: "brew", Status: "done"})
			}
		}
	}

	// Batch install casks
	if len(casks) > 0 {
		var installErr error
		_ = spinner.New().
			Title(fmt.Sprintf("Installing %d cask packages...", len(casks))).
			Action(func() {
				installErr = a.installer.BatchInstallCask(casks)
			}).
			Run()
		for _, name := range caskNames {
			if installErr != nil {
				fmt.Println(statusFail(name))
				a.results = append(a.results, InstallResult{Name: name, Method: "cask", Status: "fail"})
			} else {
				fmt.Println(statusDone(name))
				a.results = append(a.results, InstallResult{Name: name, Method: "cask", Status: "done"})
			}
		}
	}

	// Batch install apt
	if len(aptPkgs) > 0 {
		var installErr error
		_ = spinner.New().
			Title(fmt.Sprintf("Installing %d apt packages...", len(aptPkgs))).
			Action(func() {
				installErr = a.installer.BatchInstallApt(aptPkgs)
			}).
			Run()
		for _, name := range aptNames {
			if installErr != nil {
				fmt.Println(statusFail(name))
				a.results = append(a.results, InstallResult{Name: name, Method: "apt", Status: "fail"})
			} else {
				fmt.Println(statusDone(name))
				a.results = append(a.results, InstallResult{Name: name, Method: "apt", Status: "done"})
			}
		}
	}

	// Batch install dnf
	if len(dnfPkgs) > 0 {
		var installErr error
		_ = spinner.New().
			Title(fmt.Sprintf("Installing %d dnf packages...", len(dnfPkgs))).
			Action(func() {
				installErr = a.installer.BatchInstallDnf(dnfPkgs)
			}).
			Run()
		for _, name := range dnfNames {
			if installErr != nil {
				fmt.Println(statusFail(name))
				a.results = append(a.results, InstallResult{Name: name, Method: "dnf", Status: "fail"})
			} else {
				fmt.Println(statusDone(name))
				a.results = append(a.results, InstallResult{Name: name, Method: "dnf", Status: "done"})
			}
		}
	}

	// Phase 3: Secondary packages (parallel)
	fmt.Println()
	fmt.Println(boldStyle.Render("  Secondary Packages"))

	var secondary []Package
	for _, pkg := range toInstall {
		method := pkg.Packages[a.osInfo.Target]
		if !method.IsSystemMethod() {
			secondary = append(secondary, pkg)
		}
	}

	if len(secondary) > 0 {
		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, 4) // max 4 parallel installs

		for _, pkg := range secondary {
			wg.Add(1)
			go func(p Package) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				result := a.installer.Install(p)

				mu.Lock()
				a.results = append(a.results, result)
				switch result.Status {
				case "ok":
					fmt.Println(statusOK(p.Name))
				case "done":
					fmt.Println(statusDone(p.Name))
				case "skip":
					fmt.Println(statusSkip(p.Name))
				case "fail":
					fmt.Println(statusFail(p.Name))
				}
				mu.Unlock()
			}(pkg)
		}
		wg.Wait()
	}

	return nil
}

func (a *App) showSummary() {
	var installed, alreadyOK, skipped, failed int
	var failedPkgs []string

	for _, r := range a.results {
		switch r.Status {
		case "done":
			installed++
		case "ok":
			alreadyOK++
		case "skip":
			skipped++
		case "fail":
			failed++
			failedPkgs = append(failedPkgs, r.Name)
		}
	}

	var nextSteps strings.Builder
	nextSteps.WriteString(fmt.Sprintf("  %s Restart your shell or run: %s\n",
		dimStyle.Render("1."), boldStyle.Render("exec $SHELL")))
	nextSteps.WriteString(fmt.Sprintf("  %s Verify git config: %s\n",
		dimStyle.Render("2."), boldStyle.Render("git config --list")))
	nextSteps.WriteString(fmt.Sprintf("  %s Check SSH key: %s\n",
		dimStyle.Render("3."), boldStyle.Render("ssh-add -l")))

	if len(failedPkgs) > 0 {
		nextSteps.WriteString(fmt.Sprintf("\n  %s\n",
			warningStyle.Render("Failed packages that need manual attention:")))
		for _, name := range failedPkgs {
			nextSteps.WriteString(fmt.Sprintf("    %s %s\n",
				errorStyle.Render("•"), name))
		}
	}

	fmt.Println()
	fmt.Println(summaryPanel(installed, alreadyOK, skipped, failed, nextSteps.String()))
	fmt.Println()
}
