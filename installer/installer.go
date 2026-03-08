package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// InstallResult tracks the outcome of a package installation
type InstallResult struct {
	Name   string
	Method string
	Status string // "ok", "done", "skip", "fail"
	Error  string
}

// InstalledCache caches the list of installed packages per method
type InstalledCache struct {
	mu    sync.Mutex
	cache map[string]map[string]bool
}

func NewInstalledCache() *InstalledCache {
	return &InstalledCache{cache: make(map[string]map[string]bool)}
}

func (c *InstalledCache) get(method string, loader func() map[string]bool) map[string]bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := c.cache[method]; ok {
		return cached
	}
	result := loader()
	c.cache[method] = result
	return result
}

func parseLines(output string) map[string]bool {
	m := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			m[line] = true
		}
	}
	return m
}

func parseFirstWord(output string) map[string]bool {
	m := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			m[fields[0]] = true
		}
	}
	return m
}

// PackageInstaller handles all package installation logic
type PackageInstaller struct {
	target string
	cache  *InstalledCache
}

func NewPackageInstaller(target string) *PackageInstaller {
	return &PackageInstaller{
		target: target,
		cache:  NewInstalledCache(),
	}
}

// run executes a shell command. In verbose mode it streams output to the terminal;
// otherwise it runs silently. Use this for install commands where output is not needed.
func (pi *PackageInstaller) run(cmd string) error {
	debugLog("$ %s", cmd)
	if Verbose {
		err := runShell(cmd)
		if err != nil {
			debugLog("command failed: %v", err)
		}
		return err
	}
	_, err := runShellSilent(cmd)
	if err != nil {
		debugLog("command failed: %v", err)
	}
	return err
}

// runCapture executes a shell command and always returns its output (needed for parsing).
// It logs the command when verbose.
func (pi *PackageInstaller) runCapture(cmd string) (string, error) {
	debugLog("$ %s", cmd)
	out, err := runShellSilent(cmd)
	if err != nil {
		debugLog("command failed: %v", err)
	}
	return out, err
}

// IsInstalled checks if a package is already installed
func (pi *PackageInstaller) IsInstalled(name string, method InstallMethod) bool {
	m := method.MethodName()
	switch m {
	case "brew":
		installed := pi.cache.get("brew", func() map[string]bool {
			out, _ := runShellSilent("brew list --formula -1")
			return parseLines(out)
		})
		// Tap-qualified formulas like "owner/repo/name" appear as just "name" in brew list
		formula := method.Brew
		if idx := strings.LastIndex(formula, "/"); idx >= 0 {
			formula = formula[idx+1:]
		}
		return installed[formula] || installed[method.Brew]
	case "cask":
		installed := pi.cache.get("cask", func() map[string]bool {
			out, _ := runShellSilent("brew list --cask -1")
			result := parseLines(out)
			// Also check /Applications
			for _, appDir := range []string{"/Applications", filepath.Join(os.Getenv("HOME"), "Applications")} {
				entries, _ := os.ReadDir(appDir)
				for _, e := range entries {
					name := strings.TrimSuffix(e.Name(), ".app")
					result[strings.ToLower(strings.ReplaceAll(name, " ", "-"))] = true
				}
			}
			return result
		})
		return installed[method.Cask]
	case "apt":
		installed := pi.cache.get("apt", func() map[string]bool {
			out, _ := runShellSilent("dpkg-query -W -f='${Package}\n' 2>/dev/null")
			return parseLines(out)
		})
		return installed[method.Apt]
	case "dnf":
		installed := pi.cache.get("dnf", func() map[string]bool {
			out, _ := runShellSilent("rpm -qa --qf '%{NAME}\n'")
			return parseLines(out)
		})
		// dnf can have multiple packages like "gcc gcc-c++ make"
		for _, p := range strings.Fields(method.Dnf) {
			if !installed[p] {
				return false
			}
		}
		return true
	case "uv_tool":
		installed := pi.cache.get("uv_tool", func() map[string]bool {
			out, _ := runShellSilent("uv tool list")
			return parseFirstWord(out)
		})
		return installed[method.UvTool]
	case "cargo":
		installed := pi.cache.get("cargo", func() map[string]bool {
			out, _ := runShellSilent("cargo install --list")
			return parseFirstWord(out)
		})
		return installed[method.Cargo]
	case "go_tool":
		binName := method.GoTool
		if idx := strings.LastIndex(binName, "/"); idx >= 0 {
			binName = binName[idx+1:]
		}
		if idx := strings.Index(binName, "@"); idx >= 0 {
			binName = binName[:idx]
		}
		return commandExists(binName)
	case "snap":
		installed := pi.cache.get("snap", func() map[string]bool {
			out, _ := runShellSilent("snap list 2>/dev/null")
			return parseFirstWord(out)
		})
		return installed[method.Snap.Name]
	case "flatpak":
		installed := pi.cache.get("flatpak", func() map[string]bool {
			out, _ := runShellSilent("flatpak list --columns=application 2>/dev/null")
			return parseLines(out)
		})
		return installed[method.Flatpak]
	case "yay":
		installed := pi.cache.get("yay", func() map[string]bool {
			out, _ := runShellSilent("yay -Qq 2>/dev/null")
			return parseLines(out)
		})
		return installed[method.Yay]
	case "gh_extension":
		installed := pi.cache.get("gh_ext", func() map[string]bool {
			out, _ := runShellSilent("gh extension list 2>/dev/null")
			return parseLines(out)
		})
		extName := method.GhExtension
		if idx := strings.LastIndex(extName, "/"); idx >= 0 {
			extName = extName[idx+1:]
		}
		for entry := range installed {
			if strings.Contains(entry, extName) {
				return true
			}
		}
		return false
	case "eget":
		toolName := method.Eget
		if idx := strings.LastIndex(toolName, "/"); idx >= 0 {
			toolName = toolName[idx+1:]
		}
		return commandExists(toolName)
	case "manual":
		return pi.isManualInstalled(name, method.Manual)
	}
	// Fallback: check if name is a command
	return commandExists(name)
}

func (pi *PackageInstaller) isManualInstalled(name string, manual *ManualSpec) bool {
	if manual.CheckCommand != "" {
		return commandExists(manual.CheckCommand)
	}
	if manual.CheckDir != "" {
		expanded, _ := runShellSilent(fmt.Sprintf("echo %s", manual.CheckDir))
		expanded = strings.TrimSpace(expanded)
		if info, err := os.Stat(expanded); err == nil && info.IsDir() {
			return true
		}
	}
	if manual.Dest != "" {
		expanded, _ := runShellSilent(fmt.Sprintf("echo %s", manual.Dest))
		expanded = strings.TrimSpace(expanded)
		if info, err := os.Stat(expanded); err == nil {
			_ = info
			return true
		}
	}
	// For dmg/tar_gz installs, check /Applications for any .app containing the package name
	if manual.Type == "dmg" || manual.Type == "tar_gz" {
		for _, appDir := range []string{"/Applications", filepath.Join(os.Getenv("HOME"), "Applications")} {
			entries, _ := os.ReadDir(appDir)
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".app") &&
					strings.Contains(strings.ToLower(e.Name()), strings.ToLower(name)) {
					return true
				}
			}
		}
	}
	return commandExists(name)
}

// Install installs a single package and returns the result
func (pi *PackageInstaller) Install(pkg Package) InstallResult {
	method, ok := pkg.Packages[pi.target]
	if !ok {
		return InstallResult{Name: pkg.Name, Method: "n/a", Status: "skip"}
	}

	methodName := method.MethodName()

	if pi.IsInstalled(pkg.Name, method) {
		return InstallResult{Name: pkg.Name, Method: methodName, Status: "ok"}
	}

	debugLog("installing %s via %s", pkg.Name, methodName)

	var err error
	switch methodName {
	case "brew":
		err = pi.run(fmt.Sprintf("brew install %s", method.Brew))
	case "cask":
		err = pi.run(fmt.Sprintf("brew install --cask %s", method.Cask))
	case "apt":
		err = pi.run(fmt.Sprintf("sudo apt install -y %s", method.Apt))
	case "dnf":
		err = pi.run(fmt.Sprintf("sudo dnf install -y %s", method.Dnf))
	case "uv_tool":
		err = pi.run(fmt.Sprintf("uv tool install %s", method.UvTool))
	case "cargo":
		err = pi.run(fmt.Sprintf("cargo install %s", method.Cargo))
	case "go_tool":
		err = pi.run(fmt.Sprintf("go install %s", method.GoTool))
	case "snap":
		snapFlags := ""
		if method.Snap.Classic {
			snapFlags += " --classic"
		}
		if method.Snap.Channel != "" {
			snapFlags += fmt.Sprintf(" --channel %s", method.Snap.Channel)
		}
		err = pi.run(fmt.Sprintf("sudo snap install %s%s", method.Snap.Name, snapFlags))
	case "flatpak":
		err = pi.run(fmt.Sprintf("flatpak install -y flathub %s", method.Flatpak))
	case "yay":
		err = pi.run(fmt.Sprintf("yay -S --noconfirm %s", method.Yay))
	case "gh_extension":
		if _, ghErr := runShellSilent("gh auth status"); ghErr != nil {
			return InstallResult{Name: pkg.Name, Method: methodName, Status: "skip", Error: "gh not authenticated"}
		}
		err = pi.run(fmt.Sprintf("gh extension install %s", method.GhExtension))
	case "eget":
		os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".local", "bin"), 0o755)
		err = pi.run(fmt.Sprintf("eget %s --to ~/.local/bin", method.Eget))
	case "manual":
		err = pi.installManual(pkg.Name, method.Manual)
	default:
		return InstallResult{Name: pkg.Name, Method: methodName, Status: "skip", Error: "unknown method"}
	}

	if err != nil {
		return InstallResult{Name: pkg.Name, Method: methodName, Status: "fail", Error: err.Error()}
	}
	return InstallResult{Name: pkg.Name, Method: methodName, Status: "done"}
}

func (pi *PackageInstaller) installManual(name string, manual *ManualSpec) error {
	switch manual.Type {
	case "script":
		args := manual.Args
		var cmd string
		if args != "" {
			cmd = fmt.Sprintf(`sh -c "$(curl -fsSL %s)" "" %s`, manual.URL, args)
		} else {
			cmd = fmt.Sprintf("curl -fsSL %s | bash", manual.URL)
		}
		return pi.run(cmd)
	case "git_clone":
		expanded, _ := runShellSilent(fmt.Sprintf("echo %s", manual.Dest))
		dest := strings.TrimSpace(expanded)
		os.MkdirAll(filepath.Dir(dest), 0o755)
		return pi.run(fmt.Sprintf("git clone %s %s", manual.URL, dest))
	case "dmg":
		return pi.installDmg(manual)
	case "zip":
		return pi.installZip(manual)
	case "tar_gz":
		return pi.installTarGz(manual)
	case "deb":
		return pi.installDeb(manual)
	case "rpm":
		return pi.installRpm(manual)
	case "appimage":
		return pi.installAppImage(manual)
	}
	return fmt.Errorf("unknown manual type: %s", manual.Type)
}

// resolveGhAssetURL returns the download URL for a GitHub release asset matching
// assetPattern. Supports multiple "|"-separated substrings that must ALL be present
// in the asset name (AND logic). Searches all releases including pre-releases.
func resolveGhAssetURL(repo, assetPattern string) (string, error) {
	parts := strings.Split(assetPattern, "|")
	conditions := make([]string, len(parts))
	for i, p := range parts {
		conditions[i] = fmt.Sprintf(`(.name | contains("%s"))`, strings.TrimSpace(p))
	}
	jqSelect := strings.Join(conditions, " and ")

	out, err := runShellSilent(fmt.Sprintf(
		`gh api repos/%s/releases --jq '.[].assets[] | select(%s) | .browser_download_url' | head -1`,
		repo, jqSelect,
	))
	if err != nil {
		return "", fmt.Errorf("gh api releases: %w", err)
	}
	url := strings.TrimSpace(out)
	if url == "" {
		return "", fmt.Errorf("no asset matching %q in %s", assetPattern, repo)
	}
	return url, nil
}

func (pi *PackageInstaller) installDmg(manual *ManualSpec) error {
	url, err := resolveGhAssetURL(manual.Repo, manual.AssetPattern)
	if err != nil {
		return err
	}
	tmpF, err := os.CreateTemp("", "install-*.dmg")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpFile := tmpF.Name()
	tmpF.Close()
	defer os.Remove(tmpFile)
	if err := pi.run(fmt.Sprintf("curl -fsSL -o %s %s", tmpFile, url)); err != nil {
		return fmt.Errorf("download dmg: %w", err)
	}
	mountPoint, err := os.MkdirTemp("", "dmg-mount-*")
	if err != nil {
		return fmt.Errorf("create mount dir: %w", err)
	}
	if err := pi.run(fmt.Sprintf("hdiutil attach -nobrowse -mountpoint %s %s", mountPoint, tmpFile)); err != nil {
		os.RemoveAll(mountPoint)
		return fmt.Errorf("mount dmg: %w", err)
	}
	defer func() {
		pi.run(fmt.Sprintf("hdiutil detach -quiet %s", mountPoint)) //nolint:errcheck
		os.RemoveAll(mountPoint)
	}()

	// Find .app bundle (may be nested) and copy to /Applications
	var appPath string
	_ = filepath.Walk(mountPoint, func(path string, info os.FileInfo, err error) error {
		if err != nil || appPath != "" {
			return nil
		}
		if info.IsDir() && strings.HasSuffix(info.Name(), ".app") {
			appPath = path
		}
		return nil
	})
	if appPath == "" {
		return fmt.Errorf("no .app found in dmg")
	}
	dest := filepath.Join("/Applications", filepath.Base(appPath))
	if err := pi.run(fmt.Sprintf("cp -R %s %s", appPath, dest)); err != nil {
		return fmt.Errorf("copy app: %w", err)
	}
	return nil
}

func (pi *PackageInstaller) installZip(manual *ManualSpec) error {
	url, err := resolveGhAssetURL(manual.Repo, manual.AssetPattern)
	if err != nil {
		return err
	}
	tmpF, err := os.CreateTemp("", "install-*.zip")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpFile := tmpF.Name()
	tmpF.Close()
	defer os.Remove(tmpFile)
	if err := pi.run(fmt.Sprintf("curl -fsSL -o %s %s", tmpFile, url)); err != nil {
		return fmt.Errorf("download zip: %w", err)
	}
	tmpDir := filepath.Join(os.TempDir(), "zip-extract")
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0o755)
	defer os.RemoveAll(tmpDir)
	if err := pi.run(fmt.Sprintf("unzip -o %s -d %s", tmpFile, tmpDir)); err != nil {
		return fmt.Errorf("extract zip: %w", err)
	}
	// Find .app bundle and copy to /Applications
	entries, _ := os.ReadDir(tmpDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".app") {
			dest := filepath.Join("/Applications", e.Name())
			if err := pi.run(fmt.Sprintf("cp -R %s %s", filepath.Join(tmpDir, e.Name()), dest)); err != nil {
				return fmt.Errorf("copy app: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("no .app found in zip")
}

func (pi *PackageInstaller) installTarGz(manual *ManualSpec) error {
	url, err := resolveGhAssetURL(manual.Repo, manual.AssetPattern)
	if err != nil {
		return err
	}
	tmpF, err := os.CreateTemp("", "install-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpFile := tmpF.Name()
	tmpF.Close()
	defer os.Remove(tmpFile)
	if err := pi.run(fmt.Sprintf("curl -fsSL -o %s %s", tmpFile, url)); err != nil {
		return fmt.Errorf("download tar.gz: %w", err)
	}
	tmpDir := filepath.Join(os.TempDir(), "targz-extract")
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0o755)
	defer os.RemoveAll(tmpDir)
	if err := pi.run(fmt.Sprintf("tar -xzf %s -C %s", tmpFile, tmpDir)); err != nil {
		return fmt.Errorf("extract tar.gz: %w", err)
	}
	// Find .app bundle and copy to /Applications
	var appName string
	_ = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || appName != "" {
			return nil
		}
		if info.IsDir() && strings.HasSuffix(info.Name(), ".app") {
			appName = path
		}
		return nil
	})
	if appName == "" {
		return fmt.Errorf("no .app found in tar.gz")
	}
	dest := filepath.Join("/Applications", filepath.Base(appName))
	if err := pi.run(fmt.Sprintf("cp -R %s %s", appName, dest)); err != nil {
		return fmt.Errorf("copy app: %w", err)
	}
	// Remove quarantine attribute so macOS Gatekeeper doesn't block it
	_ = pi.run(fmt.Sprintf("xattr -cr %s", dest))
	return nil
}

func (pi *PackageInstaller) installDeb(manual *ManualSpec) error {
	url, err := resolveGhAssetURL(manual.Repo, manual.AssetPattern)
	if err != nil {
		return err
	}
	tmpF, err := os.CreateTemp("", "install-*.deb")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpFile := tmpF.Name()
	tmpF.Close()
	defer os.Remove(tmpFile)
	if err := pi.run(fmt.Sprintf("curl -fsSL -o %s %s", tmpFile, url)); err != nil {
		return fmt.Errorf("download deb: %w", err)
	}
	return pi.run(fmt.Sprintf("sudo dpkg -i %s", tmpFile))
}

func (pi *PackageInstaller) installRpm(manual *ManualSpec) error {
	url, err := resolveGhAssetURL(manual.Repo, manual.AssetPattern)
	if err != nil {
		return err
	}
	tmpF, err := os.CreateTemp("", "install-*.rpm")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpFile := tmpF.Name()
	tmpF.Close()
	defer os.Remove(tmpFile)
	if err := pi.run(fmt.Sprintf("curl -fsSL -o %s %s", tmpFile, url)); err != nil {
		return fmt.Errorf("download rpm: %w", err)
	}
	return pi.run(fmt.Sprintf("sudo dnf install -y %s", tmpFile))
}

func (pi *PackageInstaller) installAppImage(manual *ManualSpec) error {
	url, err := resolveGhAssetURL(manual.Repo, manual.AssetPattern)
	if err != nil {
		return err
	}
	expanded, _ := runShellSilent(fmt.Sprintf("echo %s", manual.Dest))
	dest := strings.TrimSpace(expanded)
	if dest == "" {
		dest = filepath.Join(os.Getenv("HOME"), ".local", "bin", manual.Repo[strings.LastIndex(manual.Repo, "/")+1:])
	}
	os.MkdirAll(filepath.Dir(dest), 0o755)
	if err := pi.run(fmt.Sprintf("curl -fsSL -o %s %s", dest, url)); err != nil {
		return fmt.Errorf("download appimage: %w", err)
	}
	return pi.run(fmt.Sprintf("chmod +x %s", dest))
}

// BatchInstallBrew installs multiple brew formulas at once
func (pi *PackageInstaller) BatchInstallBrew(formulas []string) error {
	if len(formulas) == 0 {
		return nil
	}
	_, err := runShellSilent(fmt.Sprintf("brew install %s", strings.Join(formulas, " ")))
	return err
}

// BatchInstallCask installs multiple cask packages at once
func (pi *PackageInstaller) BatchInstallCask(casks []string) error {
	if len(casks) == 0 {
		return nil
	}
	_, err := runShellSilent(fmt.Sprintf("brew install --cask %s", strings.Join(casks, " ")))
	return err
}

// BatchInstallApt installs multiple apt packages at once
func (pi *PackageInstaller) BatchInstallApt(pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}
	_, err := runShellSilent(fmt.Sprintf("sudo apt install -y %s", strings.Join(pkgs, " ")))
	return err
}

// BatchInstallDnf installs multiple dnf packages at once
func (pi *PackageInstaller) BatchInstallDnf(pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}
	_, err := runShellSilent(fmt.Sprintf("sudo dnf install -y %s", strings.Join(pkgs, " ")))
	return err
}

// InstallBrewTaps taps all configured homebrew taps
func (pi *PackageInstaller) InstallBrewTaps(taps []string) []InstallResult {
	var results []InstallResult
	for _, tap := range taps {
		_, err := runShellSilent(fmt.Sprintf("brew tap %s", tap))
		if err != nil {
			results = append(results, InstallResult{Name: tap, Method: "tap", Status: "fail", Error: err.Error()})
		} else {
			results = append(results, InstallResult{Name: tap, Method: "tap", Status: "ok"})
		}
	}
	return results
}

// InstallSingleTool installs a specific tool by running a command
func InstallSingleTool(name string, installCmd string) error {
	cmd := exec.Command("sh", "-c", installCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
