package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Package represents a single package entry in packages.json
type Package struct {
	Name        string
	Description string
	Packages    map[string]InstallMethod // keyed by OS target
}

// InstallMethod represents how to install a package on a specific OS.
// Only one field will be set.
type InstallMethod struct {
	Brew        string       `json:"brew,omitempty"`
	Cask        string       `json:"cask,omitempty"`
	Apt         string       `json:"apt,omitempty"`
	Dnf         string       `json:"dnf,omitempty"`
	UvTool      string       `json:"uv_tool,omitempty"`
	Cargo       string       `json:"cargo,omitempty"`
	GoTool      string       `json:"go_tool,omitempty"`
	Snap        *SnapSpec    `json:"snap,omitempty"`
	Flatpak     string       `json:"flatpak,omitempty"`
	Yay         string       `json:"yay,omitempty"`
	GhExtension string       `json:"gh_extension,omitempty"`
	Eget        string       `json:"eget,omitempty"`
	Manual      *ManualSpec  `json:"manual,omitempty"`
}

// SnapSpec handles snap packages which can be a string or object
type SnapSpec struct {
	Name    string `json:"name"`
	Classic bool   `json:"classic,omitempty"`
}

func (s *SnapSpec) UnmarshalJSON(data []byte) error {
	// Try as string first
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		s.Name = name
		return nil
	}
	// Try as object
	type snapObj struct {
		Name    string `json:"name"`
		Classic bool   `json:"classic,omitempty"`
	}
	var obj snapObj
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	s.Name = obj.Name
	s.Classic = obj.Classic
	return nil
}

// ManualSpec for manual installation (script, git_clone, dmg, deb, appimage)
type ManualSpec struct {
	URL          string `json:"url,omitempty"`
	Repo         string `json:"repo,omitempty"`         // GitHub repo "owner/name" for gh release download
	AssetPattern string `json:"asset_pattern,omitempty"` // suffix to match release asset filename
	Type         string `json:"type"`                   // "script", "git_clone", "dmg", "deb", "rpm", "appimage"
	Dest         string `json:"dest,omitempty"`
	CheckCommand string `json:"check_command,omitempty"`
	CheckDir     string `json:"check_dir,omitempty"`
	Args         string `json:"args,omitempty"`
}

// MethodName returns the install method name for this InstallMethod
func (im InstallMethod) MethodName() string {
	switch {
	case im.Brew != "":
		return "brew"
	case im.Cask != "":
		return "cask"
	case im.Apt != "":
		return "apt"
	case im.Dnf != "":
		return "dnf"
	case im.UvTool != "":
		return "uv_tool"
	case im.Cargo != "":
		return "cargo"
	case im.GoTool != "":
		return "go_tool"
	case im.Snap != nil:
		return "snap"
	case im.Flatpak != "":
		return "flatpak"
	case im.Yay != "":
		return "yay"
	case im.GhExtension != "":
		return "gh_extension"
	case im.Eget != "":
		return "eget"
	case im.Manual != nil:
		return "manual"
	}
	return "unknown"
}

// IsSystemMethod returns true for brew/cask/apt/dnf
func (im InstallMethod) IsSystemMethod() bool {
	m := im.MethodName()
	return m == "brew" || m == "cask" || m == "apt" || m == "dnf"
}

// PackageCatalog holds all parsed packages and brew taps
type PackageCatalog struct {
	BrewTaps []string
	Packages []Package
}

// Category groupings for the package selection form
type PackageCategory struct {
	Name     string
	Packages []Package
}

// package categories by name
var categoryMap = map[string]string{
	// System Tools
	"git": "System Tools", "curl": "System Tools", "zsh": "System Tools",
	"make": "System Tools", "build-essential": "System Tools", "ripgrep": "System Tools",
	"jq": "System Tools", "bat": "System Tools",

	// Editors
	"neovim": "Editors", "helix": "Editors", "fresh-editor": "Editors",
	"visual-studio-code": "Editors", "zed": "Editors",

	// Terminal Tools
	"fzf": "Terminal Tools", "eza": "Terminal Tools", "bottom": "Terminal Tools",
	"zoxide": "Terminal Tools", "zellij": "Terminal Tools", "lazygit": "Terminal Tools",
	"lazydocker": "Terminal Tools", "glow": "Terminal Tools", "television": "Terminal Tools",
	"difftastic": "Terminal Tools", "ghostty": "Terminal Tools",

	// Development
	"node": "Development", "npm": "Development", "nvm": "Development",
	"go": "Development", "uv": "Development", "python3-pip": "Development",
	"pre-commit": "Development", "biome": "Development", "ipython": "Development",
	"jupyter": "Development", "just": "Development", "act": "Development",
	"rumdl": "Development", "djlint": "Development", "harlequin": "Development",
	"euporie": "Development",

	// GUI Applications
	"zen-browser": "GUI Applications", "claude": "GUI Applications",
	"claude-code": "GUI Applications", "raycast": "GUI Applications",
	"obsidian": "GUI Applications", "thunderbird": "GUI Applications",
	"gimp": "GUI Applications", "flameshot": "GUI Applications",
	"protonvpn": "GUI Applications",
	"localsend": "GUI Applications", "httpie-desktop": "GUI Applications",
	"ente-auth": "GUI Applications", "proton-pass": "GUI Applications",
	"alt-tab": "GUI Applications", "logi-options-plus": "GUI Applications",

	// Shell & Prompt
	"oh-my-zsh": "Shell & Prompt", "zsh-autosuggestions": "Shell & Prompt",
	"zsh-syntax-highlighting": "Shell & Prompt", "pure-prompt": "Shell & Prompt",

	// Utilities
	"rclone": "Utilities", "rclone-ui": "Utilities", "topgrade": "Utilities", "httpie": "Utilities",
	"vhs": "Utilities", "gum": "Utilities", "hyperfine": "Utilities",
	"fx": "Utilities", "zola": "Utilities", "vhs-eget": "Utilities",
	"tv": "Utilities", "crush": "Utilities", "eget": "Utilities",
	"intelli-shell": "Utilities", "dockutil": "Utilities",

	// GitHub
	"gh": "GitHub", "gh-dash": "GitHub", "gama": "GitHub",

	// Podman
	"podman": "Containers",

	// Proton
	"proton-pass-cli": "Proton",
}

var categoryOrder = []string{
	"System Tools", "Editors", "Terminal Tools", "Development",
	"GUI Applications", "Shell & Prompt", "GitHub", "Containers",
	"Proton", "Utilities",
}

func categorizePackages(pkgs []Package) []PackageCategory {
	groups := make(map[string][]Package)
	for _, pkg := range pkgs {
		cat, ok := categoryMap[pkg.Name]
		if !ok {
			cat = "Other"
		}
		groups[cat] = append(groups[cat], pkg)
	}

	var categories []PackageCategory
	for _, name := range categoryOrder {
		if pkgs, ok := groups[name]; ok {
			sort.Slice(pkgs, func(i, j int) bool {
				return pkgs[i].Name < pkgs[j].Name
			})
			categories = append(categories, PackageCategory{Name: name, Packages: pkgs})
		}
	}
	// Add "Other" if there are uncategorized packages
	if others, ok := groups["Other"]; ok {
		sort.Slice(others, func(i, j int) bool {
			return others[i].Name < others[j].Name
		})
		categories = append(categories, PackageCategory{Name: "Other", Packages: others})
	}
	return categories
}

// LoadPackages reads and parses packages.json
func LoadPackages(sourceDir string) (*PackageCatalog, error) {
	path := filepath.Join(sourceDir, "packages.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading packages.json: %w", err)
	}

	// Parse as generic map first to handle _brew_taps vs package entries
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing packages.json: %w", err)
	}

	catalog := &PackageCatalog{}

	// Extract _brew_taps
	if tapsRaw, ok := raw["_brew_taps"]; ok {
		if err := json.Unmarshal(tapsRaw, &catalog.BrewTaps); err != nil {
			return nil, fmt.Errorf("parsing _brew_taps: %w", err)
		}
	}

	// Parse package entries
	for name, rawPkg := range raw {
		if name[0] == '_' {
			continue
		}

		var entry struct {
			Description string                     `json:"description"`
			Packages    map[string]json.RawMessage `json:"packages"`
		}
		if err := json.Unmarshal(rawPkg, &entry); err != nil {
			continue // skip malformed entries
		}
		if entry.Packages == nil {
			continue
		}

		pkg := Package{
			Name:        name,
			Description: entry.Description,
			Packages:    make(map[string]InstallMethod),
		}

		for osTarget, methodRaw := range entry.Packages {
			var method InstallMethod
			if err := json.Unmarshal(methodRaw, &method); err != nil {
				continue
			}
			pkg.Packages[osTarget] = method
		}

		catalog.Packages = append(catalog.Packages, pkg)
	}

	// Sort packages by name for consistent ordering
	sort.Slice(catalog.Packages, func(i, j int) bool {
		return catalog.Packages[i].Name < catalog.Packages[j].Name
	})

	return catalog, nil
}

// FilterForTarget returns only packages that have an install method for the given OS target
func (c *PackageCatalog) FilterForTarget(target string) []Package {
	var result []Package
	for _, pkg := range c.Packages {
		if _, ok := pkg.Packages[target]; ok {
			result = append(result, pkg)
		}
	}
	return result
}
