package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	sourceDir := flag.String("source", "", "Path to chezmoi source directory (containing packages.json)")
	flag.Parse()

	// Default source dir: the parent of the installer directory
	if *sourceDir == "" {
		exe, err := os.Executable()
		if err == nil {
			*sourceDir = filepath.Dir(exe)
		}
		// Fallback: try relative to current working directory
		if *sourceDir == "" || !fileExists(filepath.Join(*sourceDir, "packages.json")) {
			cwd, _ := os.Getwd()
			*sourceDir = filepath.Dir(cwd)
		}
		// Fallback: chezmoi source dir
		if !fileExists(filepath.Join(*sourceDir, "packages.json")) {
			*sourceDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "chezmoi")
		}
	}

	if !fileExists(filepath.Join(*sourceDir, "packages.json")) {
		fmt.Println(errorStyle.Render("Error: packages.json not found in " + *sourceDir))
		fmt.Println(dimStyle.Render("Use --source to specify the chezmoi source directory."))
		os.Exit(1)
	}

	app := NewApp(*sourceDir)
	if err := app.Run(); err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render(fmt.Sprintf("Error: %v", err)))
		os.Exit(1)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
