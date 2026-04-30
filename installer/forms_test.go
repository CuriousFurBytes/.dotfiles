package main

import "testing"

func TestFlattenPackagesAlphabetical(t *testing.T) {
	categories := []PackageCategory{
		{Name: "B", Packages: []Package{{Name: "zeta"}, {Name: "alpha"}}},
		{Name: "A", Packages: []Package{{Name: "beta"}}},
	}

	got := flattenPackagesAlphabetical(categories)
	want := []string{"alpha", "beta", "zeta"}

	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i] {
			t.Fatalf("index %d: got %q want %q", i, got[i].Name, want[i])
		}
	}
}
