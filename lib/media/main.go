/*
	A simple MPRIS media player monitor that updates eww variables with song info.
	a media player that supports MPRIS over DBus (e.g. Browsers, spotify, vlc, etc).
*/

package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

func ewwUpdate(key, value string) {
	value = strings.ReplaceAll(value, "'", `'\''`)
	exec.Command("eww", "update", fmt.Sprintf("%s=%s", key, value)).Run()
}

func formatDurationMicros(micros int64) string {
	if micros <= 0 {
		return "00:00"
	}
	secs := micros / 1_000_000
	min := secs / 60
	sec := secs % 60
	return fmt.Sprintf("%02d:%02d", min, sec)
}

func clampNormalized(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func formatNormalized(elapsed, total int64) string {
	if total <= 0 {
		return "0"
	}
	
	val := float64(elapsed) / float64(total) * 100
	val = clampNormalized(val)
	return fmt.Sprintf("%.0f", val)
}

type SongInfo struct {
	Title      string
	Artist     string
	Album      string
	ArtUrl     string 
	Status     string
	Playing    bool
	Paused     bool
	Stopped    bool
	Elapsed    int64 // microseconds
	Total      int64 // microseconds
	Normalized string
}

func emptySongInfo() SongInfo {
	return SongInfo{
		Title: "----",
		Artist: "--",
		Album: "--",
		ArtUrl: "./assets/icons/music-2.svg",
		Status: "Stopped", 
		Playing: false,
		Paused: false,
		Stopped: true,
		Elapsed: 0,
		Total: 0,
		Normalized: "0",
	}
}

func updateEwwFromSong(info SongInfo) {
	ewwUpdate("SONG_TITLE", info.Title)
	ewwUpdate("SONG_ARTIST", info.Artist)
	ewwUpdate("SONG_ALBUM_ART", info.ArtUrl)
	ewwUpdate("SONG_ALBUM", info.Album)
	ewwUpdate("SONG_STATUS", info.Status)
	ewwUpdate("SONG_PLAYING", fmt.Sprintf("%v", info.Playing))
	ewwUpdate("SONG_PAUSED", fmt.Sprintf("%v", info.Paused))
	ewwUpdate("SONG_STOPPED", fmt.Sprintf("%v", info.Stopped))
	ewwUpdate("SONG_SEEK_POSITION_ELAPSED", formatDurationMicros(info.Elapsed))
	ewwUpdate("SONG_SEEK_POSITION_TOTAL", formatDurationMicros(info.Total))
	ewwUpdate("SONG_SEEK_POSITION_NORMALIZED", info.Normalized)
}

// make a short http client with timeouts
var httpClient = &http.Client{
	Timeout: 8 * time.Second,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	},
}

func CleanArtURL(raw string) string {
	if raw == "" {
		return ""
	}

	if strings.HasPrefix(raw, "file://") {
		path, _ := url.PathUnescape(strings.TrimPrefix(raw, "file://"))
		return "file://" + filepath.Clean(path)
	}

	if strings.HasPrefix(raw, "/") {
		return "file://" + filepath.Clean(raw)
	}
	return raw
}

func FetchArt(artUrl string) string {
	artUrl = strings.TrimSpace(artUrl)
	if artUrl == "" {
		return ""
	}

	artUrl = CleanArtURL(artUrl)

	// Local file
	if strings.HasPrefix(artUrl, "file://") {
		path, _ := url.PathUnescape(strings.TrimPrefix(artUrl, "file://"))
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			abs, _ := filepath.Abs(path)
			return "file://" + abs
		}
		return ""
	}

	// HTTP/HTTPS
	if strings.HasPrefix(artUrl, "http://") || strings.HasPrefix(artUrl, "https://") {
		// cache name
		h := sha1.Sum([]byte(artUrl))
		hash := hex.EncodeToString(h[:])
		ext := ".img"
		u, err := url.Parse(artUrl)
		if err == nil {
			// try to preserve extension if present
			if p := filepath.Ext(u.Path); p != "" {
				ext = p
			}
		}
		tmpPath := filepath.Join(os.TempDir(), "mpris_art_"+hash+ext)
		// if exists, reuse
		if fi, err := os.Stat(tmpPath); err == nil && !fi.IsDir() {
			abs, _ := filepath.Abs(tmpPath)
			return "file://" + abs
		}
		// download
		resp, err := httpClient.Get(artUrl)
		if err != nil {
			fmt.Println("FetchArt: download error:", err)
			return ""
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			fmt.Println("FetchArt: bad status", resp.StatusCode)
			return ""
		}
		out, err := os.Create(tmpPath)
		if err != nil {
			fmt.Println("FetchArt: create file:", err)
			return ""
		}
		defer out.Close()
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			fmt.Println("FetchArt: write file:", err)
			// try to remove partial
			out.Close()
			os.Remove(tmpPath)
			return ""
		}
		abs, _ := filepath.Abs(tmpPath)
		return "file://" + abs
	}

	// unknown scheme, return unchanged (some players may provide "cover:" or data: URIs)
	return ""
}

func getMprisPlayers(conn *dbus.Conn) ([]string, error) {
	var names []string
	err := conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names)
	if err != nil {
		return nil, err
	}
	var players []string
	for _, n := range names {
		if strings.HasPrefix(n, "org.mpris.MediaPlayer2.") {
			players = append(players, n)
		}
	}
	return players, nil
}

func toInt64Micros(v interface{}) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case uint64:
		return int64(t)
	case int32:
		return int64(t)
	case uint32:
		return int64(t)
	case int:
		return int64(t)
	case float64:
		return int64(t)
	default:
		return 0
	}
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case dbus.ObjectPath:
		return string(t)
	default:
		return ""
	}
}

func getSongInfo(obj dbus.BusObject) SongInfo {
	propsIface := "org.freedesktop.DBus.Properties"
	call := obj.Call(propsIface+".GetAll", 0, "org.mpris.MediaPlayer2.Player")
	if call.Err != nil {
		// can't access player props (maybe exited)
		return emptySongInfo()
	}

	rawProps, ok := call.Body[0].(map[string]dbus.Variant)
	if !ok {
		return emptySongInfo()
	}
	// PlaybackStatus may not exist
	status := ""
	if s, ok := rawProps["PlaybackStatus"]; ok {
		status = toString(s.Value())
	}

	metaVariant, hasMeta := rawProps["Metadata"]
	var meta map[string]dbus.Variant
	if hasMeta {
		if mv, ok := metaVariant.Value().(map[string]dbus.Variant); ok {
			meta = mv
		}
	}

	var (
		title, artist, album, artUrl string
		length, pos                  int64
	)

	if meta != nil {
		if t, ok := meta["xesam:title"]; ok {
			title = toString(t.Value())
		}
		if a, ok := meta["xesam:artist"]; ok {
			// could be []string or []interface{}
			switch arr := a.Value().(type) {
			case []string:
				if len(arr) > 0 {
					artist = arr[0]
				}
			case []interface{}:
				if len(arr) > 0 {
					artist = fmt.Sprint(arr[0])
				}
			}
		}
		if al, ok := meta["xesam:album"]; ok {
			album = toString(al.Value())
		}
		if art, ok := meta["mpris:artUrl"]; ok {
			artUrl = toString(art.Value())
		}
		if l, ok := meta["mpris:length"]; ok {
			length = toInt64Micros(l.Value())
		}
	}

	// Position can be present in top-level properties
	if p, ok := rawProps["Position"]; ok {
		pos = toInt64Micros(p.Value())
	}

	// Normalize and fetch art only if changed to avoid repeated downloads
	var fetchedArt string
	if artUrl != "" {
		fetchedArt = FetchArt(artUrl)
	}

	si := SongInfo{
		Title:      title,
		Artist:     artist,
		Album:      album,
		ArtUrl:     fetchedArt,
		Status:     status,
		Playing:    status == "Playing",
		Paused:     status == "Paused",
		Stopped:    status == "Stopped",
		Elapsed:    pos,
		Total:      length,
		Normalized: formatNormalized(pos, length),
	}

	return si
}

func main() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	var (
		mtx          sync.Mutex
		player       string
		obj          dbus.BusObject
		songInfo     = emptySongInfo()
		lastUpdate   = time.Now()
		activeMatches []string // keep track of added match rules so we can remove them when switching players
	)

	updateEwwFromSong(songInfo)

	signalChan := make(chan *dbus.Signal, 50)
	conn.Signal(signalChan)

	// We want to get NameOwnerChanged signals to detect new/removed players
	conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',interface='org.freedesktop.DBus',member='NameOwnerChanged'")

	addMatch := func(rule string) {
		// avoid duplicates
		for _, r := range activeMatches {
			if r == rule {
				return
			}
		}
		conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule)
		activeMatches = append(activeMatches, rule)
	}

	removeAllMatches := func() {
		for _, r := range activeMatches {
			conn.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, r)
		}
		activeMatches = activeMatches[:0]
	}

	findPlayer := func() string {
		players, _ := getMprisPlayers(conn)
		if len(players) > 0 {
			return players[0]
		}
		return ""
	}

	setPlayer := func(newPlayer string) {
		mtx.Lock()
		defer mtx.Unlock()
		if newPlayer == player {
			return
		}
		// remove previous matches to avoid stale events
		removeAllMatches()

		if newPlayer == "" {
			player = ""
			obj = nil
			songInfo = emptySongInfo()
			lastUpdate = time.Now()
			updateEwwFromSong(songInfo)
			return
		}

		player = newPlayer
		obj = conn.Object(player, "/org/mpris/MediaPlayer2")
		songInfo = getSongInfo(obj)
		lastUpdate = time.Now()
		updateEwwFromSong(songInfo)

		// Add a match for PropertiesChanged only from this sender
		propsRule := fmt.Sprintf("type='signal',interface='org.freedesktop.DBus.Properties',sender='%s'", player)
		addMatch(propsRule)
		// Add a match for Seeked signals
		seekRule := fmt.Sprintf("type='signal',interface='org.mpris.MediaPlayer2.Player',member='Seeked',sender='%s'", player)
		addMatch(seekRule)
	}

	setPlayer(findPlayer())

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mtx.Lock()
			// Advance elapsed if playing, based on lastUpdate
			now := time.Now()
			if songInfo.Playing {
				delta := now.Sub(lastUpdate)
				// add microseconds
				add := int64(delta.Seconds() * 1_000_000)
				songInfo.Elapsed += add
				// protect against negative or absurd values
				if songInfo.Elapsed < 0 {
					songInfo.Elapsed = 0
				}
				if songInfo.Total > 0 && songInfo.Elapsed > songInfo.Total+5_000_000 { // allow small overshoot
					songInfo.Elapsed = songInfo.Total
				}
				songInfo.Normalized = formatNormalized(songInfo.Elapsed, songInfo.Total)
				updateEwwFromSong(songInfo)
			}
			lastUpdate = now
			mtx.Unlock()

			// try to find a player if none present
			mtx.Lock()
			curPlayer := player
			mtx.Unlock()
			if curPlayer == "" {
				setPlayer(findPlayer())
			}

		case sig := <-signalChan:
			if sig == nil || len(sig.Body) == 0 {
				continue
			}
			// NameOwnerChanged: player started or stopped
			if sig.Name == "org.freedesktop.DBus.NameOwnerChanged" {
				if len(sig.Body) >= 3 {
					name, _ := sig.Body[0].(string)
					oldOwner, _ := sig.Body[1].(string)
					newOwner, _ := sig.Body[2].(string)
					if strings.HasPrefix(name, "org.mpris.MediaPlayer2.") {
						if newOwner != "" && player == "" {
							// new player started and we don't have one
							setPlayer(name)
						} else if name == player && newOwner == "" && oldOwner != "" {
							// current player disappeared
							setPlayer("")
						}
					}
				}
				continue
			}

			if strings.HasSuffix(sig.Name, "PropertiesChanged") {
				// check if signal is from our current player
				mtx.Lock()
				curObj := obj
				mtx.Unlock()
				if curObj == nil {
					continue
				}
				if len(sig.Body) >= 2 {
					changed, ok := sig.Body[1].(map[string]dbus.Variant)
					if ok {
						mtx.Lock()
						if s, exists := changed["PlaybackStatus"]; exists {
							status := toString(s.Value())
							if status == "Playing" && !songInfo.Playing {
								lastUpdate = time.Now()
							}
							songInfo.Status = status
							songInfo.Playing = status == "Playing"
							songInfo.Paused = status == "Paused"
							songInfo.Stopped = status == "Stopped"
						}
						// If Position reported in changed props (some players set it here), update
						if p, exists := changed["Position"]; exists {
							songInfo.Elapsed = toInt64Micros(p.Value())
							lastUpdate = time.Now()
						}
						// If Metadata changed, re-query metadata to handle numeric type differences and art changes
						if _, exists := changed["Metadata"]; exists {
							newInfo := getSongInfo(curObj)
							// keep elapsed if newInfo.Elapsed == 0 (some players don't send pos in metadata)
							if newInfo.Elapsed != 0 {
								songInfo.Elapsed = newInfo.Elapsed
							}
							// update total if present
							if newInfo.Total != 0 {
								songInfo.Total = newInfo.Total
							}
							// update title/artist/album/art when changed
							if newInfo.Title != "" {
								songInfo.Title = newInfo.Title
							}
							if newInfo.Artist != "" {
								songInfo.Artist = newInfo.Artist
							}
							if newInfo.Album != "" {
								songInfo.Album = newInfo.Album
							}
							if newInfo.ArtUrl != "" && newInfo.ArtUrl != songInfo.ArtUrl {
								songInfo.ArtUrl = newInfo.ArtUrl
							}
							// if new playback status present, update it too
							if newInfo.Status != "" {
								songInfo.Status = newInfo.Status
								songInfo.Playing = newInfo.Playing
								songInfo.Paused = newInfo.Paused
								songInfo.Stopped = newInfo.Stopped
							}
						}
						// recompute normalized
						songInfo.Normalized = formatNormalized(songInfo.Elapsed, songInfo.Total)
						updateEwwFromSong(songInfo)
						mtx.Unlock()
					}
				}
			}

			// Seeked signal: body[0] is new position in microseconds (int64)
			if strings.HasSuffix(sig.Name, "Seeked") {
				if len(sig.Body) > 0 {
					mtx.Lock()
					if pos, ok := sig.Body[0].(int64); ok {
						songInfo.Elapsed = pos
					} else {
						// try numeric conversion
						songInfo.Elapsed = toInt64Micros(sig.Body[0])
					}
					lastUpdate = time.Now()
					songInfo.Normalized = formatNormalized(songInfo.Elapsed, songInfo.Total)
					updateEwwFromSong(songInfo)
					mtx.Unlock()
				}
			}
		}
	}
}
