package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func runCmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting %s: %v\n", name, err)
	}
}

func runWidget() {
		home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		return
	}
	runCmd(filepath.Join(home, ".config/eww/bin/niv-bar"))
}

var commands = map[string]string{
	"workspace": `
workspaces=$(
  hyprctl workspaces -j | jq -c '
    . as $ws
    | ($ws | map(.id)) as $existing_ids
    | [range(1;6)] as $all_ids
    | ($all_ids - $existing_ids) as $missing_ids
    | ($missing_ids | map({id: ., windows: 0})) as $missing_ws
    | ($ws + $missing_ws)
    | sort_by(.id)
    '
);
current_ws=$(hyprctl activeworkspace -j | jq -c .); 
eww update WORKSPACES="$workspaces"; 
eww update FOCUSED_WORKSPACE="$current_ws"
`,
	"activewindow": `
active_win=$(hyprctl activewindow -j | jq -c .); 
eww update ACTIVE_WINDOW="$active_win"
`,
}

func runShellCommand(eventType string) {
	cmdStr, ok := commands[eventType]
	if !ok {
		return
	}

	// Prepare the command to run in bash shell
	cmd := exec.Command("/usr/bin/env", "bash", "-c", cmdStr)
	
	// Set timeout using context with deadline
	done := make(chan error, 1)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Start()
	if err != nil {
		log.Printf("Failed to start command for %s: %v\n", eventType, err)
		return
	}

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("Subprocess error for %s: %s\n", eventType, stderr.String())
		} else {
			fmt.Print(stdout.String())
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		log.Printf("Subprocess timeout for %s\n", eventType)
	}
}

func main() {
	runWidget();

	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	instanceSig := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")

	if runtimeDir == "" || instanceSig == "" {
		log.Fatalf("Error: environment variables XDG_RUNTIME_DIR or HYPRLAND_INSTANCE_SIGNATURE not found.")
	}

	socketPath := filepath.Join(runtimeDir, "hypr", instanceSig, ".socket2.sock")

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Fatalf("Failed to connect to Hyprland IPC socket: %v", err)
	}
	defer conn.Close()

	log.Println("Connected to Hyprland IPC socket, subscribing to events...")

	// Subscribe to events
	subs := []string{"subscribe:workspace", "subscribe:activewindow", "subscribe:activeworkspace"}
	for _, sub := range subs {
		_, err := conn.Write([]byte(sub + "\x00"))
		if err != nil {
			log.Fatalf("Failed to write subscription: %v", err)
		}
	}

	// Run commands initially for all events
	for event := range commands {
		runShellCommand(event)
	}

	// Channel for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Reading data loop
	go func() {
		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				log.Println("Disconnected from Hyprland IPC.")
				os.Exit(0)
			}

			line = strings.TrimSpace(line)
			for eventType := range commands {
				if strings.Contains(line, eventType) {
					runShellCommand(eventType)
					break
				}
			}
		}
	}()

	// Wait for termination signal
	<-sigChan
	log.Println("Exiting...")
	conn.Close()
}
