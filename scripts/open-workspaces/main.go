package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	configFlag := flag.String("config", "", "JSON mapping of workspace to commands")
	extraCmdFlag := flag.String("extra-cmd", "", "Extra command to run after all workspaces are set up")
	flag.Parse()

	if *configFlag == "" {
		fmt.Println("Error: --config is required")
		os.Exit(1)
	}

	workspaces := make(map[string][]string)
	if err := json.Unmarshal([]byte(*configFlag), &workspaces); err != nil {
		fmt.Println("Error parsing JSON:", err)
		os.Exit(1)
	}

	for ws, cmds := range workspaces {
		for _, cmdStr := range cmds {
			cmd := exec.Command("hyprctl", fmt.Sprintf("dispatch exec [workspace %s silent] %s", ws, cmdStr))
			if err := cmd.Start(); err != nil {
				fmt.Printf("Error starting command '%s' in workspace %s: %v\n", cmdStr, ws, err)
			} 
		}
	}

	if *extraCmdFlag != "" {
		fmt.Println("Running extra command:", *extraCmdFlag)
		parts := strings.Fields(*extraCmdFlag)
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println("Error running extra command:", err)
		}
	}
}
