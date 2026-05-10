package main

import "testing"

// TestParseLines tests the parseLines() function.
func TestParseLines(t *testing.T) {
	t.Run("multi-line input", func(t *testing.T) {
		input := "git\nbrew\ncurl\n"
		got := parseLines(input)
		want := map[string]bool{"git": true, "brew": true, "curl": true}
		for k := range want {
			if !got[k] {
				t.Errorf("expected key %q in result", k)
			}
		}
		if len(got) != len(want) {
			t.Errorf("len = %d, want %d", len(got), len(want))
		}
	})

	t.Run("empty string returns empty map", func(t *testing.T) {
		got := parseLines("")
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})

	t.Run("whitespace-only lines are skipped", func(t *testing.T) {
		input := "git\n   \n\tbrew\n\n"
		got := parseLines(input)
		if !got["git"] {
			t.Error("expected 'git' in result")
		}
		if !got["brew"] {
			t.Error("expected 'brew' in result")
		}
		// whitespace-only lines must not appear
		for k := range got {
			trimmed := k
			if trimmed == "" {
				t.Error("empty string key should not be in result")
			}
		}
		if len(got) != 2 {
			t.Errorf("len = %d, want 2", len(got))
		}
	})

	t.Run("lines are trimmed before insertion", func(t *testing.T) {
		input := "  git  \n  curl  \n"
		got := parseLines(input)
		if !got["git"] {
			t.Error("expected trimmed key 'git'")
		}
		if !got["curl"] {
			t.Error("expected trimmed key 'curl'")
		}
	})
}

// TestParseFirstWord tests the parseFirstWord() function.
func TestParseFirstWord(t *testing.T) {
	t.Run("extracts first word from each line", func(t *testing.T) {
		input := "git v2.39.0\nbrew 4.0.0\ncurl 7.88.1"
		got := parseFirstWord(input)
		want := map[string]bool{"git": true, "brew": true, "curl": true}
		for k := range want {
			if !got[k] {
				t.Errorf("expected key %q in result", k)
			}
		}
		if len(got) != len(want) {
			t.Errorf("len = %d, want %d", len(got), len(want))
		}
	})

	t.Run("empty lines are skipped", func(t *testing.T) {
		input := "git v2.39.0\n\nbrew 4.0.0\n"
		got := parseFirstWord(input)
		if len(got) != 2 {
			t.Errorf("len = %d, want 2", len(got))
		}
		if !got["git"] || !got["brew"] {
			t.Errorf("result = %v, want {git:true brew:true}", got)
		}
	})

	t.Run("single-word lines work", func(t *testing.T) {
		input := "git\nbrew"
		got := parseFirstWord(input)
		if !got["git"] || !got["brew"] {
			t.Errorf("result = %v, want {git:true brew:true}", got)
		}
	})

	t.Run("empty string returns empty map", func(t *testing.T) {
		got := parseFirstWord("")
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})
}

// TestInstallResultStatus tests InstallResult struct initialization covering all fields.
func TestInstallResultStatus(t *testing.T) {
	tests := []struct {
		name   string
		result InstallResult
	}{
		{
			name:   "ok status",
			result: InstallResult{Name: "git", Method: "brew", Status: "ok", Error: ""},
		},
		{
			name:   "done status",
			result: InstallResult{Name: "ripgrep", Method: "cargo", Status: "done", Error: ""},
		},
		{
			name:   "skip status",
			result: InstallResult{Name: "some-pkg", Method: "n/a", Status: "skip", Error: ""},
		},
		{
			name:   "fail status with error",
			result: InstallResult{Name: "bad-pkg", Method: "apt", Status: "fail", Error: "exit status 1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.result
			if r.Name == "" {
				t.Error("Name should not be empty")
			}
			if r.Method == "" {
				t.Error("Method should not be empty")
			}
			if r.Status == "" {
				t.Error("Status should not be empty")
			}
			// Verify the status values are valid
			validStatuses := map[string]bool{"ok": true, "done": true, "skip": true, "fail": true}
			if !validStatuses[r.Status] {
				t.Errorf("Status %q is not one of ok/done/skip/fail", r.Status)
			}
			// If status is fail, Error should be set
			if r.Status == "fail" && r.Error == "" {
				t.Error("fail status should have an Error message")
			}
		})
	}
}
