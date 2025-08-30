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
	runCmd(filepath.Join(home, ".config/eww/bin/niv-power-menu"))
}
