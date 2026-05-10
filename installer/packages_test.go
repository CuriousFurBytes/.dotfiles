package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestMethodName tests InstallMethod.MethodName() for all method types.
func TestMethodName(t *testing.T) {
	tests := []struct {
		name   string
		method InstallMethod
		want   string
	}{
		{"brew", InstallMethod{Brew: "git"}, "brew"},
		{"cask", InstallMethod{Cask: "firefox"}, "cask"},
		{"apt", InstallMethod{Apt: "git"}, "apt"},
		{"dnf", InstallMethod{Dnf: "git"}, "dnf"},
		{"uv_tool", InstallMethod{UvTool: "ruff"}, "uv_tool"},
		{"cargo", InstallMethod{Cargo: "ripgrep"}, "cargo"},
		{"go_tool", InstallMethod{GoTool: "golang.org/x/tools/gopls@latest"}, "go_tool"},
		{"snap", InstallMethod{Snap: &SnapSpec{Name: "firefox"}}, "snap"},
		{"flatpak", InstallMethod{Flatpak: "org.gnome.gedit"}, "flatpak"},
		{"yay", InstallMethod{Yay: "aur-pkg"}, "yay"},
		{"gh_extension", InstallMethod{GhExtension: "dlvhdr/gh-dash"}, "gh_extension"},
		{"eget", InstallMethod{Eget: "owner/repo"}, "eget"},
		{"npm_global", InstallMethod{NpmGlobal: "prettier"}, "npm_global"},
		{"manual", InstallMethod{Manual: &ManualSpec{Type: "script", URL: "https://example.com"}}, "manual"},
		{"empty/unknown", InstallMethod{}, "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.method.MethodName()
			if got != tc.want {
				t.Errorf("MethodName() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestIsSystemMethod tests InstallMethod.IsSystemMethod().
func TestIsSystemMethod(t *testing.T) {
	tests := []struct {
		name   string
		method InstallMethod
		want   bool
	}{
		{"brew is system", InstallMethod{Brew: "git"}, true},
		{"cask is system", InstallMethod{Cask: "firefox"}, true},
		{"apt is system", InstallMethod{Apt: "git"}, true},
		{"dnf is system", InstallMethod{Dnf: "git"}, true},
		{"cargo is not system", InstallMethod{Cargo: "ripgrep"}, false},
		{"go_tool is not system", InstallMethod{GoTool: "golang.org/x/tools/gopls@latest"}, false},
		{"snap is not system", InstallMethod{Snap: &SnapSpec{Name: "firefox"}}, false},
		{"flatpak is not system", InstallMethod{Flatpak: "org.gnome.gedit"}, false},
		{"manual is not system", InstallMethod{Manual: &ManualSpec{Type: "script"}}, false},
		{"uv_tool is not system", InstallMethod{UvTool: "ruff"}, false},
		{"gh_extension is not system", InstallMethod{GhExtension: "dlvhdr/gh-dash"}, false},
		{"eget is not system", InstallMethod{Eget: "owner/repo"}, false},
		{"npm_global is not system", InstallMethod{NpmGlobal: "prettier"}, false},
		{"unknown is not system", InstallMethod{}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.method.IsSystemMethod()
			if got != tc.want {
				t.Errorf("IsSystemMethod() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestFilterForTarget tests PackageCatalog.FilterForTarget().
func TestFilterForTarget(t *testing.T) {
	catalog := &PackageCatalog{
		Packages: []Package{
			{
				Name: "git",
				Packages: map[string]InstallMethod{
					"darwin": {Brew: "git"},
					"ubuntu": {Apt: "git"},
				},
			},
			{
				Name: "neovim",
				Packages: map[string]InstallMethod{
					"darwin": {Brew: "neovim"},
				},
			},
			{
				Name: "ripgrep",
				Packages: map[string]InstallMethod{
					"ubuntu": {Apt: "ripgrep"},
				},
			},
		},
	}

	t.Run("darwin returns darwin packages", func(t *testing.T) {
		result := catalog.FilterForTarget("darwin")
		if len(result) != 2 {
			t.Fatalf("FilterForTarget(darwin) returned %d packages, want 2", len(result))
		}
		names := map[string]bool{}
		for _, p := range result {
			names[p.Name] = true
		}
		if !names["git"] || !names["neovim"] {
			t.Errorf("FilterForTarget(darwin) = %v, want [git neovim]", names)
		}
	})

	t.Run("ubuntu returns ubuntu packages", func(t *testing.T) {
		result := catalog.FilterForTarget("ubuntu")
		if len(result) != 2 {
			t.Fatalf("FilterForTarget(ubuntu) returned %d packages, want 2", len(result))
		}
		names := map[string]bool{}
		for _, p := range result {
			names[p.Name] = true
		}
		if !names["git"] || !names["ripgrep"] {
			t.Errorf("FilterForTarget(ubuntu) = %v, want [git ripgrep]", names)
		}
	})

	t.Run("fedora returns empty (no fedora packages)", func(t *testing.T) {
		result := catalog.FilterForTarget("fedora")
		if len(result) != 0 {
			t.Errorf("FilterForTarget(fedora) returned %d packages, want 0", len(result))
		}
	})
}

// TestCategorizePackages tests categorizePackages().
func TestCategorizePackages(t *testing.T) {
	pkgs := []Package{
		{Name: "git"},      // in categoryMap -> "System Tools"
		{Name: "neovim"},   // in categoryMap -> "Editors"
		{Name: "mypkg"},    // not in categoryMap -> "Other"
		{Name: "anotherpkg"}, // not in categoryMap -> "Other"
	}

	cats := categorizePackages(pkgs)

	t.Run("known package categorized correctly", func(t *testing.T) {
		found := false
		for _, cat := range cats {
			if cat.Name == "System Tools" {
				for _, p := range cat.Packages {
					if p.Name == "git" {
						found = true
					}
				}
			}
		}
		if !found {
			t.Error("expected 'git' in 'System Tools' category")
		}
	})

	t.Run("unknown package goes to Other", func(t *testing.T) {
		found := false
		for _, cat := range cats {
			if cat.Name == "Other" {
				for _, p := range cat.Packages {
					if p.Name == "mypkg" {
						found = true
					}
				}
			}
		}
		if !found {
			t.Error("expected 'mypkg' in 'Other' category")
		}
	})

	t.Run("categories follow categoryOrder (System Tools before Editors)", func(t *testing.T) {
		systemIdx, editorsIdx := -1, -1
		for i, cat := range cats {
			if cat.Name == "System Tools" {
				systemIdx = i
			}
			if cat.Name == "Editors" {
				editorsIdx = i
			}
		}
		if systemIdx == -1 || editorsIdx == -1 {
			t.Fatalf("systemIdx=%d editorsIdx=%d, both must be present", systemIdx, editorsIdx)
		}
		if systemIdx >= editorsIdx {
			t.Errorf("System Tools (idx %d) should appear before Editors (idx %d)", systemIdx, editorsIdx)
		}
	})

	t.Run("Other category comes last", func(t *testing.T) {
		if len(cats) == 0 {
			t.Fatal("no categories returned")
		}
		last := cats[len(cats)-1]
		if last.Name != "Other" {
			t.Errorf("last category = %q, want %q", last.Name, "Other")
		}
	})

	t.Run("packages within a category are sorted alphabetically", func(t *testing.T) {
		for _, cat := range cats {
			if cat.Name == "Other" {
				if len(cat.Packages) < 2 {
					break
				}
				for i := 1; i < len(cat.Packages); i++ {
					if cat.Packages[i].Name < cat.Packages[i-1].Name {
						t.Errorf("Other: packages not sorted: %q before %q",
							cat.Packages[i-1].Name, cat.Packages[i].Name)
					}
				}
			}
		}
	})
}

// TestSnapSpecUnmarshalJSON tests SnapSpec.UnmarshalJSON.
func TestSnapSpecUnmarshalJSON(t *testing.T) {
	t.Run("string form", func(t *testing.T) {
		var s SnapSpec
		if err := json.Unmarshal([]byte(`"firefox"`), &s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Name != "firefox" {
			t.Errorf("Name = %q, want %q", s.Name, "firefox")
		}
		if s.Classic {
			t.Error("Classic should be false")
		}
	})

	t.Run("object form with classic", func(t *testing.T) {
		var s SnapSpec
		if err := json.Unmarshal([]byte(`{"name":"code","classic":true}`), &s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Name != "code" {
			t.Errorf("Name = %q, want %q", s.Name, "code")
		}
		if !s.Classic {
			t.Error("Classic should be true")
		}
	})

	t.Run("object form without classic", func(t *testing.T) {
		var s SnapSpec
		if err := json.Unmarshal([]byte(`{"name":"vlc"}`), &s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Name != "vlc" {
			t.Errorf("Name = %q, want %q", s.Name, "vlc")
		}
		if s.Classic {
			t.Error("Classic should be false")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		var s SnapSpec
		if err := json.Unmarshal([]byte(`{invalid`), &s); err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})
}

// TestLoadPackages tests LoadPackages().
func TestLoadPackages(t *testing.T) {
	t.Run("valid packages.json", func(t *testing.T) {
		dir := t.TempDir()
		content := `{
			"_brew_taps": ["homebrew/cask"],
			"git": {
				"description": "Version control",
				"packages": {
					"darwin": {"brew": "git"},
					"ubuntu": {"apt": "git"}
				}
			}
		}`
		if err := os.WriteFile(filepath.Join(dir, "packages.json"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing packages.json: %v", err)
		}

		catalog, err := LoadPackages(dir)
		if err != nil {
			t.Fatalf("LoadPackages() error: %v", err)
		}

		if len(catalog.BrewTaps) != 1 || catalog.BrewTaps[0] != "homebrew/cask" {
			t.Errorf("BrewTaps = %v, want [homebrew/cask]", catalog.BrewTaps)
		}

		if len(catalog.Packages) != 1 {
			t.Fatalf("Packages count = %d, want 1", len(catalog.Packages))
		}

		pkg := catalog.Packages[0]
		if pkg.Name != "git" {
			t.Errorf("Package.Name = %q, want %q", pkg.Name, "git")
		}
		if pkg.Description != "Version control" {
			t.Errorf("Package.Description = %q, want %q", pkg.Description, "Version control")
		}
		if _, ok := pkg.Packages["darwin"]; !ok {
			t.Error("expected darwin install method")
		}
		if _, ok := pkg.Packages["ubuntu"]; !ok {
			t.Error("expected ubuntu install method")
		}
	})

	t.Run("multiple packages are sorted by name", func(t *testing.T) {
		dir := t.TempDir()
		content := `{
			"zebra": {"description": "Z tool", "packages": {"ubuntu": {"apt": "zebra"}}},
			"alpha": {"description": "A tool", "packages": {"ubuntu": {"apt": "alpha"}}},
			"middle": {"description": "M tool", "packages": {"ubuntu": {"apt": "middle"}}}
		}`
		if err := os.WriteFile(filepath.Join(dir, "packages.json"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing packages.json: %v", err)
		}

		catalog, err := LoadPackages(dir)
		if err != nil {
			t.Fatalf("LoadPackages() error: %v", err)
		}

		if len(catalog.Packages) != 3 {
			t.Fatalf("Packages count = %d, want 3", len(catalog.Packages))
		}

		want := []string{"alpha", "middle", "zebra"}
		for i, w := range want {
			if catalog.Packages[i].Name != w {
				t.Errorf("Packages[%d].Name = %q, want %q", i, catalog.Packages[i].Name, w)
			}
		}
	})

	t.Run("non-existent directory returns error", func(t *testing.T) {
		_, err := LoadPackages("/nonexistent/path/that/does/not/exist")
		if err == nil {
			t.Error("expected error for non-existent directory, got nil")
		}
	})

	t.Run("underscore-prefixed keys are skipped", func(t *testing.T) {
		dir := t.TempDir()
		content := `{
			"_brew_taps": ["homebrew/cask"],
			"_metadata": {"description": "ignored", "packages": {"darwin": {"brew": "nope"}}}
		}`
		if err := os.WriteFile(filepath.Join(dir, "packages.json"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing packages.json: %v", err)
		}

		catalog, err := LoadPackages(dir)
		if err != nil {
			t.Fatalf("LoadPackages() error: %v", err)
		}
		if len(catalog.Packages) != 0 {
			t.Errorf("expected 0 packages (underscore keys skipped), got %d", len(catalog.Packages))
		}
	})
}
