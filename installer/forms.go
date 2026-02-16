package main

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// BuildPackageSelectionForm creates a multi-page Huh form for package selection.
// Each category gets its own page (group) with a MultiSelect.
// All packages are selected by default.
func BuildPackageSelectionForm(categories []PackageCategory, selected map[string]*[]string) *huh.Form {
	var groups []*huh.Group

	// Welcome note as first group
	groups = append(groups, huh.NewGroup(
		huh.NewNote().
			Title("Package Selection").
			Description("Select the packages you want to install.\nAll packages are selected by default — deselect any you don't need.\n\nUse ↑/↓ to navigate, space to toggle, enter to confirm."),
	))

	for _, cat := range categories {
		var options []huh.Option[string]
		for _, pkg := range cat.Packages {
			label := fmt.Sprintf("%s — %s", pkg.Name, pkg.Description)
			options = append(options, huh.NewOption(label, pkg.Name).Selected(true))
		}

		vals := make([]string, len(cat.Packages))
		for i, pkg := range cat.Packages {
			vals[i] = pkg.Name
		}
		selected[cat.Name] = &vals

		groups = append(groups, huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(cat.Name).
				Options(options...).
				Value(selected[cat.Name]).
				Height(min(len(options)+2, 20)).
				Filterable(true),
		))
	}

	return huh.NewForm(groups...)
}

// ConfirmStep creates a simple confirm prompt for a step
func ConfirmStep(title, description string) (bool, error) {
	var confirmed bool
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Affirmative("Yes").
				Negative("Skip").
				Value(&confirmed),
		),
	).Run()
	return confirmed, err
}

// CollectSelectedPackages gathers all selected package names from the form results
func CollectSelectedPackages(selected map[string]*[]string) map[string]bool {
	result := make(map[string]bool)
	for _, names := range selected {
		for _, name := range *names {
			result[name] = true
		}
	}
	return result
}
