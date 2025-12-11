package watchers

import (
	"log"
	"os"
	"os/exec"

	"github.com/fsnotify/fsnotify"
)

const stateFile = "/tmp/niv_widget_state"

func StartEscWatcher(stop <-chan struct{}) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	bindEsc := func() {
		exec.Command("hyprctl", "keyword", "bind", ",escape,exec,~/.config/eww/niv close all").Run()
	}

	unbindEsc := func() {
		exec.Command("hyprctl", "keyword", "unbind", ",escape").Run()
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

	dir := "/tmp"
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
