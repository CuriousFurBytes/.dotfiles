package main

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/huh"
)

// BuildPackageSelectionForm creates a single-page Huh form for package selection.
// All packages are shown together and sorted alphabetically by name.
// All packages are selected by default.
func BuildPackageSelectionForm(categories []PackageCategory, selected map[string]*[]string) *huh.Form {
	allPackages := flattenPackagesAlphabetical(categories)
	var options []huh.Option[string]
	vals := make([]string, len(allPackages))

	for i, pkg := range allPackages {
		label := fmt.Sprintf("%s — %s", pkg.Name, pkg.Description)
		options = append(options, huh.NewOption(label, pkg.Name).Selected(true))
		vals[i] = pkg.Name
	}

	selected["All Packages"] = &vals

	return huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Package Selection").
				Description("Select the packages you want to install.\nAll packages are selected by default — deselect any you don't need.\n\nUse ↑/↓ to navigate, space to toggle, enter to confirm."),
		),
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("All Packages").
				Options(options...).
				Value(selected["All Packages"]).
				Height(min(len(options)+2, 20)).
				Filterable(true),
		),
	)
}

func flattenPackagesAlphabetical(categories []PackageCategory) []Package {
	var all []Package
	for _, cat := range categories {
		all = append(all, cat.Packages...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	return all
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
