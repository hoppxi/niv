package watchers

import (
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/hoppxi/wigo/internal/utils"
)

const stateFile = "/tmp/wigo/wigo_widget_state"

func StartEscWatcher(stop <-chan struct{}) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	bindEsc := func() {
		if err := utils.HyprCmd("keyword bind ,escape,exec,wigo close all"); err != nil {
			log.Printf("Unable to bind esc to close the widget. error: %v", err)
		}
		// exec.Command("hyprctl", "keyword", "bind", ",escape,exec,wigo close all").Run()
	}

	unbindEsc := func() {
		if err := utils.HyprCmd("keyword unbind ,escape"); err != nil {
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
