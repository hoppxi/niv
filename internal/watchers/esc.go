package watchers

import (
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

const stateFile = "/tmp/wigo/wigo_widget_state"

func ctlSocket() string {
	return filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "hypr", os.Getenv("HYPRLAND_INSTANCE_SIGNATURE"), ".socket.sock")
}

func runHyprctlCmd(cmd string) error {
	conn, err := net.Dial("unix", ctlSocket())
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(cmd))
	if err != nil {
		return err
	}
	return nil
}

func StartEscWatcher(stop <-chan struct{}) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	bindEsc := func() {
		if err := runHyprctlCmd("keyword bind ,escape,exec,wigo close all"); err != nil {
			log.Printf("Unable to bind esc to close the widget. error: %v", err)
		}
		// exec.Command("hyprctl", "keyword", "bind", ",escape,exec,wigo close all").Run()
	}

	unbindEsc := func() {
		if err := runHyprctlCmd("keyword unbind ,escape"); err != nil {
			log.Printf("Unable to bind esc to close the widget. error: %v", err)
		}
		// exec.Command("hyprctl", "keyword", "unbind", ",escape").Run()
	}

	checkState := func() {
		_, err := os.Stat(stateFile)
		if err == nil {
			data, err := os.ReadFile(stateFile)
			if err == nil && len(data) > 0 {
				bindEsc()
				return
			}
		}
		unbindEsc()
	}

	checkState()

	dir := "/tmp/wigo"
	if err := watcher.Add(dir); err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case <-stop:
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Name == stateFile && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Remove == fsnotify.Remove) {
				checkState()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("watcher error:", err)
		}
	}
}
