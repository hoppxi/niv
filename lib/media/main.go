package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	defaultTitle    = "----"
	defaultArtist   = "--"
	defaultArt      = "./assets/images/album-art.jpg"
	defaultStatus   = "Stopped"
	defaultElapsed  = "00:00"
	defaultTotal    = "00:00"
	defaultNormStr  = "0"
	tmpArtDir       = "./assets/images/album-art-tmp"
	objPath         = "/org/mpris/MediaPlayer2"
	ifacePlayer     = "org.mpris.MediaPlayer2.Player"
	ifaceProps      = "org.freedesktop.DBus.Properties"
	methodGet       = "org.freedesktop.DBus.Properties.Get"
	signalSeeked    = "Seeked"
)

type SongState struct {
	Title      string
	Artist     string
	AlbumArt   string
	Status     string
	Playing    bool
	Paused     bool
	Stopped    bool
	ElapsedStr string // "MM:SS"
	TotalStr   string // "MM:SS"
	NormStr    string // "0".."1" (as string)
	// internal numeric helpers
	elapsedSec int64
	totalSec   int64
}

var (
	bus          *dbus.Conn
	activeSender string // unique name (e.g., ":1.123")
	state        = SongState{
		Title:      defaultTitle,
		Artist:     defaultArtist,
		AlbumArt:   defaultArt,
		Status:     defaultStatus,
		Playing:    false,
		Paused:     false,
		Stopped:    true,
		ElapsedStr: defaultElapsed,
		TotalStr:   defaultTotal,
		NormStr:    defaultNormStr,
		elapsedSec: 0,
		totalSec:   0,
	}
)

func main() {
	
	var err error
	bus, err = dbus.SessionBus()
	if err != nil {
		panic(err)
	}

	_ = os.MkdirAll(tmpArtDir, 0o755)
	_ = os.MkdirAll(filepath.Dir(defaultArt), 0o755)

	// Subscribe to:
	// 1) PropertiesChanged (for Metadata/PlaybackStatus/Position)
	// 2) Seeked (more frequent position updates)
	// 3) NameOwnerChanged (to detect player disappearance)
	err = bus.AddMatchSignal(
		dbus.WithMatchInterface(ifaceProps),
	)
	if err != nil {
		panic(err)
	}
	err = bus.AddMatchSignal(
		dbus.WithMatchInterface(ifacePlayer),
		dbus.WithMatchMember(signalSeeked),
	)
	if err != nil {
		panic(err)
	}
	err = bus.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"),
	)
	if err != nil {
		panic(err)
	}

	sigCh := make(chan *dbus.Signal, 64)
	bus.Signal(sigCh)

	fmt.Println("MPRIS watcher running. Waiting for players...")

	// defaults
	pushAllToEww()

	for sig := range sigCh {
		switch sig.Name {
		case ifaceProps + ".PropertiesChanged":
			handlePropertiesChanged(sig)
		case ifacePlayer + "." + signalSeeked:
			handleSeeked(sig)
		case "org.freedesktop.DBus.NameOwnerChanged":
			handleNameOwnerChanged(sig)
		}
	}
}

func handlePropertiesChanged(sig *dbus.Signal) {
	if len(sig.Body) < 2 {
		return
	}
	iface, _ := sig.Body[0].(string)
	if iface != ifacePlayer {
		return
	}
	props, _ := sig.Body[1].(map[string]dbus.Variant)
	if len(props) == 0 {
		return
	}
	sender := sig.Sender

	// Get the first active playing
	if activeSender == "" && isPlayingFromProps(props) {
		activeSender = sender
		refreshSnapshotFromPlayer(sender)
	}

	if sender != activeSender {
		if activeSender == "" && isPlayingFromProps(props) {
			activeSender = sender
			refreshSnapshotFromPlayer(sender)
		}
		return
	}

	for key, v := range props {
		switch key {
		case "Metadata":
			md, _ := v.Value().(map[string]dbus.Variant)
			title := getString(md["xesam:title"], defaultTitle)
			artist := getStringFromArray(md["xesam:artist"], defaultArtist)
			if updateString(&state.Title, "SONG_TITLE", title) {
				// ok
			}
			updateString(&state.Artist, "SONG_ARTIST", artist)

			// Total in ms
			totalUSec := getInt64(md["mpris:length"], 0)
			totalSec := totalUSec / 1_000_000
			if totalSec <= 0 {
				state.totalSec = 0
				updateString(&state.TotalStr, "SONG_SEEK_POSITION_TOTAL", defaultTotal)
			} else {
				state.totalSec = totalSec
				updateString(&state.TotalStr, "SONG_SEEK_POSITION_TOTAL", fmtTime(totalSec))
			}

			// album art
			artURL := getString(md["mpris:artUrl"], "")
			if artURL != "" {
				local := downloadAlbumArt(artURL)
				if local != "" {
					updateString(&state.AlbumArt, "SONG_ALBUM_ART", local)
				}
			}

			// normalized 0-100
			recomputeNormalized()

		case "PlaybackStatus":
			status := getString(v, defaultStatus)
			updateString(&state.Status, "SONG_STATUS", status)
			updateBool(&state.Playing, "SONG_PLAYING", status == "Playing")
			updateBool(&state.Paused, "SONG_PAUSED", status == "Paused")
			updateBool(&state.Stopped, "SONG_STOPPED", status == "Stopped")

			// If stopped, reset to defaults and release the player.
			if status == "Stopped" {
				resetToDefaultsAndPush()
				activeSender = ""
			}

		case "Position":
			// ms to ss
			posUSec, ok := v.Value().(int64)
			if !ok {
				continue
			}
			secs := posUSec / 1_000_000
			state.elapsedSec = max64(secs, 0)
			updateString(&state.ElapsedStr, "SONG_SEEK_POSITION_ELAPSED", fmtTime(state.elapsedSec))
			recomputeNormalized()
		}
	}
}

func handleSeeked(sig *dbus.Signal) {
	if len(sig.Body) < 1 {
		return
	}
	if sig.Sender != activeSender {
		return
	}
	posUSec, ok := sig.Body[0].(int64)
	if !ok {
		return
	}
	secs := posUSec / 1_000_000
	state.elapsedSec = max64(secs, 0)
	updateString(&state.ElapsedStr, "SONG_SEEK_POSITION_ELAPSED", fmtTime(state.elapsedSec))
	recomputeNormalized()
}

func handleNameOwnerChanged(sig *dbus.Signal) {
	// Body: name, old_owner, new_owner
	if len(sig.Body) != 3 {
		return
	}
	name, _ := sig.Body[0].(string)
	oldOwner, _ := sig.Body[1].(string)
	newOwner, _ := sig.Body[2].(string)

	if oldOwner != "" && newOwner == "" && name == activeSender {
		resetToDefaultsAndPush()
		activeSender = ""
	}
}

func refreshSnapshotFromPlayer(sender string) {
	obj := bus.Object(sender, dbus.ObjectPath(objPath))

	// Metadata
	err := obj.Call(methodGet, 0, ifacePlayer, "Metadata").Store()
	if err == nil {
		// godbus peculiarity; use direct Store into variable
	}
	var metaVar dbus.Variant
	if call := obj.Call(methodGet, 0, ifacePlayer, "Metadata"); call.Err == nil {
		_ = call.Store(&metaVar)
		if md, ok := metaVar.Value().(map[string]dbus.Variant); ok {
			title := getString(md["xesam:title"], defaultTitle)
			artist := getStringFromArray(md["xesam:artist"], defaultArtist)
			updateString(&state.Title, "SONG_TITLE", title)
			updateString(&state.Artist, "SONG_ARTIST", artist)

			totalUSec := getInt64(md["mpris:length"], 0)
			state.totalSec = totalUSec / 1_000_000
			if state.totalSec > 0 {
				updateString(&state.TotalStr, "SONG_SEEK_POSITION_TOTAL", fmtTime(state.totalSec))
			} else {
				updateString(&state.TotalStr, "SONG_SEEK_POSITION_TOTAL", defaultTotal)
			}

			artURL := getString(md["mpris:artUrl"], "")
			if artURL != "" {
				local := downloadAlbumArt(artURL)
				if local != "" {
					updateString(&state.AlbumArt, "SONG_ALBUM_ART", local)
				}
			}
		}
	}

	// PlaybackStatus
	var psVar dbus.Variant
	if call := obj.Call(methodGet, 0, ifacePlayer, "PlaybackStatus"); call.Err == nil {
		_ = call.Store(&psVar)
		status := getString(psVar, defaultStatus)
		updateString(&state.Status, "SONG_STATUS", status)
		updateBool(&state.Playing, "SONG_PLAYING", status == "Playing")
		updateBool(&state.Paused, "SONG_PAUSED", status == "Paused")
		updateBool(&state.Stopped, "SONG_STOPPED", status == "Stopped")
	}

	// Position
	var posVar dbus.Variant
	if call := obj.Call(methodGet, 0, ifacePlayer, "Position"); call.Err == nil {
		_ = call.Store(&posVar)
		if p, ok := posVar.Value().(int64); ok {
			state.elapsedSec = max64(p/1_000_000, 0)
			updateString(&state.ElapsedStr, "SONG_SEEK_POSITION_ELAPSED", fmtTime(state.elapsedSec))
		}
	}

	recomputeNormalized()
}

func recomputeNormalized() {
	norm := "0"
	if state.totalSec > 0 {
		r := float64(state.elapsedSec) / float64(state.totalSec)
		if r < 0 {
			r = 0
		}
		if r > 1 {
			r = 1
		}
		// keep it compact for eww; 3 decimals
		norm = strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", r), "0"), ".")
	}
	updateString(&state.NormStr, "SONG_SEEK_POSITION_NORMALIZED", norm)
}

// Decode D-Bus values
func getString(v interface{}, def string) string {
	if v == nil {
		return def
	}
	switch t := v.(type) {
	case string:
		return t
	case dbus.Variant:
		return getString(t.Value(), def)
	default:
		return def
	}
}

func getStringFromArray(v interface{}, def string) string {
	if v == nil {
		return def
	}
	switch t := v.(type) {
	case []string:
		if len(t) > 0 && t[0] != "" {
			return t[0]
		}
	case []interface{}:
		if len(t) > 0 {
			if s, ok := t[0].(string); ok && s != "" {
				return s
			}
		}
	case dbus.Variant:
		return getStringFromArray(t.Value(), def)
	}
	return def
}

func getInt64(v interface{}, def int64) int64 {
	if v == nil {
		return def
	}
	switch t := v.(type) {
	case int64:
		return t
	case int32:
		return int64(t)
	case uint64:
		if t > ^uint64(0)>>1 {
			return def
		}
		return int64(t)
	case dbus.Variant:
		return getInt64(t.Value(), def)
	default:
		return def
	}
}

func isPlayingFromProps(props map[string]dbus.Variant) bool {
	if val, ok := props["PlaybackStatus"]; ok {
		return getString(val, "") == "Playing"
	}
	return false
}

func fmtTime(sec int64) string {
	if sec < 0 {
		sec = 0
	}
	m := sec / 60
	s := sec % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// Download album-art
func downloadAlbumArt(url string) string {
	// file:// directly.
	if strings.HasPrefix(url, "file://") {
		return strings.TrimPrefix(url, "file://")
	}

	// HTTP(S) download.
	resp, err := http.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}

	ext := guessExt(url, resp.Header.Get("Content-Type"))
	name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	dst := filepath.Join(tmpArtDir, name)

	f, err := os.Create(dst)
	if err != nil {
		return ""
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return ""
	}

	_ = copyFile(dst, defaultArt)
	return dst
}

func guessExt(url, contentType string) string {
	// Try URL path first.
	if u := strings.ToLower(url); strings.Contains(u, ".") {
		switch {
		case strings.HasSuffix(u, ".jpg"), strings.HasSuffix(u, ".jpeg"):
			return ".jpg"
		case strings.HasSuffix(u, ".png"):
			return ".png"
		case strings.HasSuffix(u, ".webp"):
			return ".webp"
		}
	}
	// Then MIME type.
	if ext, _ := mime.ExtensionsByType(contentType); len(ext) > 0 {
		return ext[0]
	}
	// Fallback.
	return ".jpg"
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

// Eww integration

func pushAllToEww() {
	updateString(&state.Title, "SONG_TITLE", state.Title)
	updateString(&state.Artist, "SONG_ARTIST", state.Artist)
	updateString(&state.AlbumArt, "SONG_ALBUM_ART", state.AlbumArt)
	updateString(&state.Status, "SONG_STATUS", state.Status)
	updateBool(&state.Playing, "SONG_PLAYING", state.Playing)
	updateBool(&state.Paused, "SONG_PAUSED", state.Paused)
	updateBool(&state.Stopped, "SONG_STOPPED", state.Stopped)
	updateString(&state.ElapsedStr, "SONG_SEEK_POSITION_ELAPSED", state.ElapsedStr)
	updateString(&state.TotalStr, "SONG_SEEK_POSITION_TOTAL", state.TotalStr)
	updateString(&state.NormStr, "SONG_SEEK_POSITION_NORMALIZED", state.NormStr)
}

func resetToDefaultsAndPush() {
	state = SongState{
		Title:      defaultTitle,
		Artist:     defaultArtist,
		AlbumArt:   defaultArt,
		Status:     defaultStatus,
		Playing:    false,
		Paused:     false,
		Stopped:    true,
		ElapsedStr: defaultElapsed,
		TotalStr:   defaultTotal,
		NormStr:    defaultNormStr,
		elapsedSec: 0,
		totalSec:   0,
	}
	pushAllToEww()
}

func updateString(field *string, varName, newVal string) bool {
	old := *field
	if old == newVal && ewwMatches(varName, newVal) {
		return false
	}
	*field = newVal
	runEwwUpdate(varName, newVal)
	return true
}

func updateBool(field *bool, varName string, newVal bool) bool {
	old := *field
	s := fmt.Sprintf("%v", newVal)
	if old == newVal && ewwMatches(varName, s) {
		return false
	}
	*field = newVal
	runEwwUpdate(varName, s)
	return true
}

func ewwMatches(varName, want string) bool {
	// If eww is not running or returns error, we don't block updates.
	out, err := exec.Command("bash", "-lc", fmt.Sprintf("eww get %s 2>/dev/null || true", shellQuote(varName))).Output()
	if err != nil {
		return false
	}
	got := strings.TrimSpace(string(out))
	got = normalizeEwwValue(got)
	want = normalizeEwwValue(want)
	return got == want
}

func normalizeEwwValue(s string) string {
	t := strings.TrimSpace(s)
	t = strings.Trim(t, `"'`)
	switch strings.ToLower(t) {
	case "true", "1":
		return "true"
	case "false", "0":
		return "false"
	default:
		return t
	}
}

func runEwwUpdate(varName, val string) {
	cmd := exec.Command("bash", "-lc",
		fmt.Sprintf("eww update %s=%s", shellQuote(varName), singleQuote(val)))
	_ = cmd.Run()
}

func singleQuote(s string) string {
	// Safely single-quote a shell string: ' -> '\'' pattern.
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func shellQuote(s string) string {
	// Simple shell-safe wrapper for var names.
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
