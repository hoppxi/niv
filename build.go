package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func buildWidgets(root string, folder string) {
	dirPath := filepath.Join(root, folder)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s dir: %v\n", folder, err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		widgetDir := filepath.Join(dirPath, name)

		hasGo := false
		sub, _ := os.ReadDir(widgetDir)
		for _, f := range sub {
			if f.Type().IsRegular() && filepath.Ext(f.Name()) == ".go" {
				hasGo = true
				break
			}
		}
		if !hasGo {
			fmt.Printf("Skipping %s/%s (no .go files)\n", folder, name)
			continue
		}

		outPath := filepath.Join(root, "./bin/niv-"+name)
		fmt.Printf("Building %s -> %s (dir: %s)\n", name, outPath, widgetDir)

		cmd := exec.Command("go", "build", "-o", outPath)
		cmd.Dir = widgetDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()

		if err := cmd.Run(); err != nil {
			fmt.Printf("Build failed for %s : %v\n", name, err)
		} else {
			fmt.Printf("Built %s\n", outPath)
		}
	}
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't get working dir:", err)
		os.Exit(1)
	}

	buildWidgets(root, "scripts")
	buildWidgets(root, "lib")
}
