/*
	logic to start restart and close services and binaries in the niv system
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"path/filepath"
	"syscall"
)

type Binary struct {
    Name string
    Path string
    PID  int
}

func main() {

    home, err := os.UserHomeDir()
    if err != nil {
        log.Fatal("Cannot find home directory:", err)
    }

    binDir := filepath.Join(home, ".config", "eww", "bin")
    stateFile := "/tmp/niv_bin_state.json"

    binaries := map[string]*Binary{
        "icons":      {Name: "icons", Path: filepath.Join(binDir, "niv-icons")},
        "media":      {Name: "media", Path: filepath.Join(binDir, "niv-media")},
        "workspaces": {Name: "workspaces", Path: filepath.Join(binDir, "niv-workspaces")},
    }


    if len(os.Args) < 2 {
        printHelp()
        return
    }

    cmd := os.Args[1]
	loadState(binaries, stateFile)

    switch cmd {
    case "help":
        printHelp()
    case "start":
        if len(os.Args) > 2 {
            startBinary(os.Args[2], binaries)
        } else {
            if !anyRunning(binaries) {
			runCommand(fmt.Sprintf("eww daemon; %s open bar; %s open clock",
				filepath.Join(binDir, "niv-ws"), filepath.Join(binDir, "niv-ws")))
            }
            startAll(binaries)
        }
    case "close":
        if len(os.Args) > 2 {
            stopBinary(os.Args[2], binaries)
        } else {
            runCommand("eww kill; pkill eww")
            stopAll(binaries)
        }
    case "restart":
        if len(os.Args) > 2 {
            restartBinary(os.Args[2], binaries)
        } else {
            runCommand("eww kill; pkill eww")
            stopAll(binaries)
            startAll(binaries)
            runCommand(fmt.Sprintf("eww daemon; %s open bar; %s open clock", filepath.Join(binDir, "niv-ws"), filepath.Join(binDir, "niv-ws")))
        }
    default:
        fmt.Println("Unknown command:", cmd)
        printHelp()
    }

    saveState(binaries, stateFile)
}

// anyRunning checks if any of the binaries in the map are currently running.
func anyRunning(binaries map[string]*Binary) bool {
	for _, b := range binaries {
		if b.PID != 0 && processExists(b.PID) {
			return true
		}
	}
	return false
}


func printHelp() {
    fmt.Println(`Usage: niv [command] [binary-name]

Commands:
  help               Show this help
  start              Start all binaries + run 'iosop listen 3000'
  start <name>       Start a specific binary
  close              Stop all binaries + run 'iosop closing'
  close <name>       Stop specific binary
  restart            Restart all binaries (close + start)
  restart <name>     Restart specific binary`)
}


func loadState(binaries map[string]*Binary, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var saved map[string]int
	if err := json.Unmarshal(data, &saved); err != nil {
		return
	}
	for name, pid := range saved {
		if b, ok := binaries[name]; ok {
			b.PID = pid
		}
	}
}

func saveState(binaries map[string]*Binary, path string) {
	saved := map[string]int{}
	for name, b := range binaries {
		if b.PID != 0 {
			saved[name] = b.PID
		}
	}
	data, _ := json.MarshalIndent(saved, "", "  ")
	_ = os.WriteFile(path, data, 0644)
}

func runCommand(command string) {
    fmt.Println("Running:", command)
    cmd := exec.Command("sh", "-c", command)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    _ = cmd.Run()
}
func startAll(binaries map[string]*Binary) {
	for name := range binaries {
		startBinary(name, binaries)
	}
}

func stopAll(binaries map[string]*Binary) {
	for name := range binaries {
		stopBinary(name, binaries)
	}
}

func restartBinary(name string, binaries map[string]*Binary) {
	stopBinary(name, binaries)
	startBinary(name, binaries)
}

func startBinary(name string, binaries map[string]*Binary) {
	b, ok := binaries[name]
	if !ok {
		fmt.Println("Unknown binary:", name)
		return
	}

	if b.PID != 0 && processExists(b.PID) {
		fmt.Println(name, "is already running (PID:", b.PID, ")")
		return
	}

	cmd := exec.Command(b.Path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Println("Error starting", name, ":", err)
		return
	}

	b.PID = cmd.Process.Pid
	fmt.Println("Started", name, "PID:", b.PID)
}

func stopBinary(name string, binaries map[string]*Binary) {
	b, ok := binaries[name]
	if !ok {
		fmt.Println("Unknown binary:", name)
		return
	}

	if b.PID == 0 || !processExists(b.PID) {
		fmt.Println(name, "is not running")
		b.PID = 0
		return
	}

	process, err := os.FindProcess(b.PID)
	if err != nil {
		fmt.Println("Cannot find process for", name, ":", err)
		return
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Println("Error stopping", name, ":", err)
		return
	}

	fmt.Println("Stopped", name, "PID:", b.PID)
	b.PID = 0
}

func processExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
