package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func runCmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting %s: %v\n", name, err)
	}
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		return
	}

	// Start eww daemon
	runCmd("eww", "daemon")

	// Start widgets
	runCmd(filepath.Join(home, ".config/eww/bin/niv-bar"))
	runCmd(filepath.Join(home, ".config/eww/bin/niv-icon"))
	runCmd(filepath.Join(home, ".config/eww/bin/niv-clock"))
}
