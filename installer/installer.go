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

// IsInstalled checks if a package is already installed
func (pi *PackageInstaller) IsInstalled(name string, method InstallMethod) bool {
	m := method.MethodName()
	switch m {
	case "brew":
		installed := pi.cache.get("brew", func() map[string]bool {
			out, _ := runShellSilent("brew list --formula -1")
			return parseLines(out)
		})
		return installed[method.Brew]
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
	// For dmg installs, check /Applications for any .app containing the package name
	if manual.Type == "dmg" {
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

	var err error
	switch methodName {
	case "brew":
		_, err = runShellSilent(fmt.Sprintf("zb install %s", method.Brew))
	case "cask":
		_, err = runShellSilent(fmt.Sprintf("brew install --cask %s", method.Cask))
	case "apt":
		_, err = runShellSilent(fmt.Sprintf("sudo apt install -y %s", method.Apt))
	case "dnf":
		_, err = runShellSilent(fmt.Sprintf("sudo dnf install -y %s", method.Dnf))
	case "uv_tool":
		_, err = runShellSilent(fmt.Sprintf("uv tool install %s", method.UvTool))
	case "cargo":
		_, err = runShellSilent(fmt.Sprintf("cargo install %s", method.Cargo))
	case "go_tool":
		_, err = runShellSilent(fmt.Sprintf("go install %s", method.GoTool))
	case "snap":
		flag := ""
		if method.Snap.Classic {
			flag = " --classic"
		}
		_, err = runShellSilent(fmt.Sprintf("sudo snap install %s%s", method.Snap.Name, flag))
	case "flatpak":
		_, err = runShellSilent(fmt.Sprintf("flatpak install -y flathub %s", method.Flatpak))
	case "yay":
		_, err = runShellSilent(fmt.Sprintf("yay -S --noconfirm %s", method.Yay))
	case "gh_extension":
		if _, ghErr := runShellSilent("gh auth status"); ghErr != nil {
			return InstallResult{Name: pkg.Name, Method: methodName, Status: "skip", Error: "gh not authenticated"}
		}
		_, err = runShellSilent(fmt.Sprintf("gh extension install %s", method.GhExtension))
	case "eget":
		os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".local", "bin"), 0o755)
		_, err = runShellSilent(fmt.Sprintf("eget %s --to ~/.local/bin", method.Eget))
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
		_, err := runShellSilent(cmd)
		return err
	case "git_clone":
		expanded, _ := runShellSilent(fmt.Sprintf("echo %s", manual.Dest))
		dest := strings.TrimSpace(expanded)
		os.MkdirAll(filepath.Dir(dest), 0o755)
		_, err := runShellSilent(fmt.Sprintf("git clone %s %s", manual.URL, dest))
		return err
	case "dmg":
		return pi.installDmg(manual)
	case "deb":
		return pi.installDeb(manual)
	case "rpm":
		return pi.installRpm(manual)
	case "appimage":
		return pi.installAppImage(manual)
	}
	return fmt.Errorf("unknown manual type: %s", manual.Type)
}

// resolveGhAssetURL returns a temp-downloaded path for a GitHub release asset matching the pattern.
func resolveGhAssetURL(repo, assetPattern string) (string, error) {
	// Use gh to find the matching asset URL from the latest release
	out, err := runShellSilent(fmt.Sprintf(
		`gh release view --repo %s --json assets -q '.assets[] | select(.name | endswith("%s")) | .url'`,
		repo, assetPattern,
	))
	if err != nil {
		return "", fmt.Errorf("gh release view: %w", err)
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
	tmpFile := filepath.Join(os.TempDir(), "zebar-install.dmg")
	if _, err := runShellSilent(fmt.Sprintf("curl -fsSL -o %s %s", tmpFile, url)); err != nil {
		return fmt.Errorf("download dmg: %w", err)
	}
	mountOut, err := runShellSilent(fmt.Sprintf("hdiutil attach -nobrowse -quiet %s", tmpFile))
	if err != nil {
		return fmt.Errorf("mount dmg: %w", err)
	}
	// Find the mount point (last line of hdiutil output)
	var mountPoint string
	for _, line := range strings.Split(strings.TrimSpace(mountOut), "\n") {
		if strings.Contains(line, "/Volumes/") {
			parts := strings.Fields(line)
			mountPoint = parts[len(parts)-1]
		}
	}
	if mountPoint == "" {
		return fmt.Errorf("could not determine dmg mount point")
	}
	defer runShellSilent(fmt.Sprintf("hdiutil detach -quiet %s", mountPoint)) //nolint:errcheck

	// Copy .app to /Applications
	entries, _ := os.ReadDir(mountPoint)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".app") {
			dest := filepath.Join("/Applications", e.Name())
			if _, err := runShellSilent(fmt.Sprintf("cp -R %s %s", filepath.Join(mountPoint, e.Name()), dest)); err != nil {
				return fmt.Errorf("copy app: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("no .app found in dmg")
}

func (pi *PackageInstaller) installDeb(manual *ManualSpec) error {
	url, err := resolveGhAssetURL(manual.Repo, manual.AssetPattern)
	if err != nil {
		return err
	}
	tmpFile := filepath.Join(os.TempDir(), "install.deb")
	if _, err := runShellSilent(fmt.Sprintf("curl -fsSL -o %s %s", tmpFile, url)); err != nil {
		return fmt.Errorf("download deb: %w", err)
	}
	_, err = runShellSilent(fmt.Sprintf("sudo dpkg -i %s", tmpFile))
	return err
}

func (pi *PackageInstaller) installRpm(manual *ManualSpec) error {
	url, err := resolveGhAssetURL(manual.Repo, manual.AssetPattern)
	if err != nil {
		return err
	}
	tmpFile := filepath.Join(os.TempDir(), "install.rpm")
	if _, err := runShellSilent(fmt.Sprintf("curl -fsSL -o %s %s", tmpFile, url)); err != nil {
		return fmt.Errorf("download rpm: %w", err)
	}
	_, err = runShellSilent(fmt.Sprintf("sudo dnf install -y %s", tmpFile))
	return err
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
	if _, err := runShellSilent(fmt.Sprintf("curl -fsSL -o %s %s", dest, url)); err != nil {
		return fmt.Errorf("download appimage: %w", err)
	}
	_, err = runShellSilent(fmt.Sprintf("chmod +x %s", dest))
	return err
}

// BatchInstallBrew installs multiple brew formulas at once
func (pi *PackageInstaller) BatchInstallBrew(formulas []string) error {
	if len(formulas) == 0 {
		return nil
	}
	_, err := runShellSilent(fmt.Sprintf("zb install %s", strings.Join(formulas, " ")))
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
