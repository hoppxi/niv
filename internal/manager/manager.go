package manager

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hoppxi/wigo/internal/wallpaper"
	"github.com/hoppxi/wigo/internal/watchers"
)

type TrackedCmd struct {
	Cmd    *exec.Cmd
	Cancel context.CancelFunc
}

type AppManager struct {
	mu      sync.Mutex
	cmds    []TrackedCmd
	stops   []chan struct{}
	wg      sync.WaitGroup
	started bool
}

var Manage = &AppManager{}

func getSocketPath() string {
	var baseDir string
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		baseDir = runtimeDir
	} else {
		baseDir = os.TempDir()
	}

	socketDir := filepath.Join(baseDir, "wigo")
	if err := os.MkdirAll(socketDir, 0o755); err != nil {
		return filepath.Join(os.TempDir(), "wigo-socket.sock")
	}
	return filepath.Join(socketDir, "socket.sock")
}

func NewCmd(command string, args ...string) (*exec.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, command, args...)

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd, cancel
}

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

func (m *AppManager) StartIPCServer() {
	socketPath := getSocketPath()
	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Error listening on socket: %v", err)
	}
	defer listener.Close()

	log.Printf("IPC Server listening on: %s", socketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
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
		return
	}

	command := strings.TrimSpace(string(buf[:n]))

	switch command {

	case "STOP":
		log.Println("Received STOP via IPC. Shutting down...")
		_, _ = conn.Write([]byte("OK: Shutting down."))

		// Close immediately so client doesn't hang
		_ = conn.Close()

		go func() {
			m.StopAll()
			time.Sleep(200 * time.Millisecond)
			os.Exit(0)
		}()

	case "STATUS":
		_, _ = conn.Write([]byte("OK: running"))
	case "START":
		m.mu.Lock()
		if m.started {
			m.mu.Unlock()
			_, _ = conn.Write([]byte("OK: Already started"))
			return
		}
		m.started = true
		m.mu.Unlock()

		log.Println("Received START via IPC. Initializing watchers and widgets...")
		_, _ = conn.Write([]byte("OK: Starting"))

		cfg := Config.Load()
		wallpaper.SetWallpaperStartup()
		watchers.ConfigUpdate(cfg)
		Config.Watch(func() {
			watchers.ConfigUpdate(cfg)
		})

		Manage.StartWatcher(watchers.StartAudioWatcher)
		Manage.StartWatcher(watchers.StartBatteryWatcher)
		Manage.StartWatcher(watchers.StartNetworkWatcher)
		Manage.StartWatcher(watchers.StartBluetoothWatcher)
		Manage.StartWatcher(watchers.StartDisplayWatcher)
		Manage.StartWatcher(watchers.StartMediaWatcher)
		Manage.StartWatcher(watchers.StartWorkspaceWatcher)
		Manage.StartWatcher(watchers.StartMiscWatcher)
		Manage.StartWatcher(watchers.StartEscWatcher)
		Manage.StartWatcher(watchers.StartLEDsWatcher)

		if err := exec.Command("eww", "open-many", "bar", "wallpaper", "clock", "notification-view", "osd").Run(); err != nil {
			log.Printf("Failed to start widgets: %v", err)
		}

	default:
		_, _ = conn.Write([]byte("ERR: unknown command"))
	}
}

func (m *AppManager) StartWatcher(f func(stop <-chan struct{})) {
	stop := make(chan struct{})
	m.mu.Lock()
	m.stops = append(m.stops, stop)
	m.mu.Unlock()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Watcher panic: %v", r)
					}
				}()
				f(stop)
			}()

			select {
			case <-stop:
				return
			case <-time.After(2 * time.Second):
				log.Println("Restarting watcher...")
			}
		}
	}()
}

func (m *AppManager) StopAll() {
	m.mu.Lock()
	cmds := m.cmds
	stops := m.stops
	m.cmds = nil
	m.stops = nil
	m.mu.Unlock()

	for _, s := range stops {
		close(s)
	}

	for _, t := range cmds {
		if t.Cancel != nil {
			t.Cancel()
		}
		if t.Cmd == nil || t.Cmd.Process == nil {
			continue
		}

		pid := t.Cmd.Process.Pid
		pgid, err := syscall.Getpgid(pid)

		if err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGTERM)

			time.Sleep(50 * time.Millisecond)
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		}

		_ = t.Cmd.Process.Kill()
		_ = t.Cmd.Wait()
	}
}

func (m *AppManager) ConnectIPC() (net.Conn, error) {
	return net.DialTimeout("unix", getSocketPath(), 500*time.Millisecond)
}

func (m *AppManager) SendIPCCommand(cmd string) (string, error) {
	conn, err := m.ConnectIPC()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(cmd)); err != nil {
		return "", err
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	return string(buf[:n]), nil
}
