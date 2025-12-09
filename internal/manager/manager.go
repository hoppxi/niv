package manager

import (
	"context"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

type TrackedCmd struct {
	Cmd    *exec.Cmd
	Cancel context.CancelFunc
}

type AppManager struct {
	mu    sync.Mutex
	cmds  []TrackedCmd
	stops []chan struct{}
	wg    sync.WaitGroup
}

var Manage = &AppManager{}

// Helper to get the socket path
func getSocketPath() string {
	return filepath.Join(os.TempDir(), "niv.sock")
}

// ----------------------------------------------------------------------
// NEW COMMAND FUNCTIONS
// ----------------------------------------------------------------------

// NewCmd creates and configures the command but DOES NOT start it.
// The caller (watcher) must use this to set up pipes before starting.
func NewCmd(command string, args ...string) (*exec.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd, cancel
}

// StartTrackedCmd takes an initialized command and starts it, adding it to the manager.
func StartTrackedCmd(cmd *exec.Cmd, cancel context.CancelFunc) *exec.Cmd {
	if err := cmd.Start(); err != nil {
		cancel()
		log.Printf("Failed to start %s: %v", cmd.Args[0], err)
		return nil
	}

	Manage.mu.Lock()
	Manage.cmds = append(Manage.cmds, TrackedCmd{Cmd: cmd, Cancel: cancel})
	Manage.mu.Unlock()

	return cmd
}

// ----------------------------------------------------------------------
// IPC SERVER FOR REMOTE KILL
// ----------------------------------------------------------------------

func (m *AppManager) StartIPCServer() {
	// Clean up old socket if it exists
	os.Remove(getSocketPath())

	listener, err := net.Listen("unix", getSocketPath())
	if err != nil {
		log.Fatalf("Error listening on socket: %v", err)
	}
	defer listener.Close()

	log.Printf("IPC Server listening on: %s", getSocketPath())

	// Add listener close to StopAll cleanup
	m.StartWatcher(func(stop <-chan struct{}) {
		<-stop
		listener.Close()
	})

	for {
		conn, err := listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go m.handleConnection(conn)
	}
}

func (m *AppManager) handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Printf("Error reading command: %v", err)
		return
	}

	command := strings.TrimSpace(string(buf[:n]))

	if command == "STOP" {
		log.Println("Received STOP command via IPC. Initiating shutdown...")
		conn.Write([]byte("OK: Shutting down."))
		
		// Run cleanup and force exit the main process
		m.StopAll()
		os.Exit(0)
	} else {
		conn.Write([]byte("Error: Unknown command."))
	}
}

// ----------------------------------------------------------------------
// WATCHERS AND STOP EVERYTHING (UNCHANGED)
// ----------------------------------------------------------------------

func (m *AppManager) StartWatcher(f func(stop <-chan struct{})) {
    // ... (Your existing StartWatcher code) ...
	stop := make(chan struct{})
	m.mu.Lock()
	m.stops = append(m.stops, stop)
	m.mu.Unlock()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		f(stop)
	}()
}

func (m *AppManager) WaitWatchers() {
    // ... (Your existing WaitWatchers code) ...
	m.wg.Wait()
}

func (m *AppManager) StopAll() {
    // ... (Your existing StopAll code) ...
	m.mu.Lock()
	cmds := m.cmds
	stops := m.stops
	m.cmds = nil
	m.stops = nil
	m.mu.Unlock()

	// tell watchers to stop
	for _, s := range stops {
		close(s)
	}
	m.wg.Wait()

	// NOW KILL ALL PROCESS CONTEXTS + PIDS
	for _, t := range cmds {
		if t.Cancel != nil {
			t.Cancel()
		}
		if t.Cmd == nil || t.Cmd.Process == nil {
			continue
		}

		pid := t.Cmd.Process.Pid
		pgid, _ := syscall.Getpgid(pid)

		// Kill process group
		if pgid > 0 {
			// Using SIGTERM first for a graceful exit, then SIGKILL as fallback if needed
			syscall.Kill(-pgid, syscall.SIGTERM) 
			// Wait briefly... (can be improved with a timeout, but SIGKILL is safer for immediate cleanup)
			syscall.Kill(-pgid, syscall.SIGKILL)
		}

		// Kill the process itself (fallback)
		t.Cmd.Process.Kill()

		// WAIT to fully reap the process (removes zombie)
		t.Cmd.Wait()
	}
	log.Println("All processes killed and watchers stopped.")
}