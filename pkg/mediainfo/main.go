package mediainfo

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

// MediaInfo contains the extracted and formatted data about the currently playing media.
type MediaInfo struct {
	Title                  string  `json:"title"`
	Artist                 string  `json:"artist"`
	Album                  string  `json:"album"`
	Art                    string  `json:"art"`
	Status                 string  `json:"status"`
	Playing                bool    `json:"playing"`
	Paused                 bool    `json:"paused"`
	Stopped                bool    `json:"stopped"`
	SeekPositionElapsed    string  `json:"seek_position_elapsed"`
	SeekPositionTotal      string  `json:"seek_position_total"`
	SeekPositionNormalized float64 `json:"seek_position_normalized"` // 0.0–1.0
	PositionRaw            int64   `json:"-"`                      // Ignore in JSON (microseconds)
	LengthRaw              int64   `json:"-"`                      // Ignore in JSON (microseconds)
}

// Global variables for configuration
var httpClient = &http.Client{Timeout: 8 * time.Second}
var debug = false // set to true to log errors

// ------------------- Helpers -------------------

func emptyMediaInfo() *MediaInfo {
	return &MediaInfo{
		Title:                  "No media playing",
		Artist:                 "--",
		Album:                  "--",
		Art:                    "./assets/icons/music-2.svg",
		Status:                 "Stopped",
		Playing:                false,
		Paused:                 false,
		Stopped:                true,
		SeekPositionElapsed:    "00:00",
		SeekPositionTotal:      "00:00",
		SeekPositionNormalized: 0,
	}
}

func FormatDurationMicros(micros int64) string {
	if micros <= 0 {
		return "00:00"
	}
	secs := micros / 1_000_000
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
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

func FormatNormalized(elapsed, total int64) float64 {
	if total <= 0 {
		return 0
	}
	// Calculate as 0-100%
	val := float64(elapsed) / float64(total) * 100
	return clampNormalized(val)
}

func cleanArtURL(raw string) string {
	raw = strings.TrimSpace(raw)
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

func fetchArt(artURL string) string {
	artURL = cleanArtURL(artURL)
	if artURL == "" {
		return ""
	}

	// 1. Local file path
	if strings.HasPrefix(artURL, "file://") {
		path := strings.TrimPrefix(artURL, "file://")
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			abs, _ := filepath.Abs(path)
			return "file://" + abs
		}
		return ""
	}

	// 2. HTTP/HTTPS URL (with caching)
	if strings.HasPrefix(artURL, "http://") || strings.HasPrefix(artURL, "https://") {
		h := sha1.Sum([]byte(artURL))
		hash := hex.EncodeToString(h[:])
		
		ext := ".img" // Default extension
		if u, err := url.Parse(artURL); err == nil {
			if p := filepath.Ext(u.Path); p != "" {
				ext = p // Preserve extension if present in URL path
			}
		}

		tmpPath := filepath.Join(os.TempDir(), "mpris_art_"+hash+ext)

		// Check cache
		if fi, err := os.Stat(tmpPath); err == nil && !fi.IsDir() {
			abs, _ := filepath.Abs(tmpPath)
			return "file://" + abs
		}

		// Download
		resp, err := httpClient.Get(artURL)
		if err != nil {
			if debug {
				fmt.Printf("fetchArt: failed to GET %s: %v\n", artURL, err)
			}
			return ""
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			if debug {
				fmt.Printf("fetchArt: non-200 status %d for %s\n", resp.StatusCode, artURL)
			}
			return ""
		}

		out, err := os.Create(tmpPath)
		if err != nil {
			if debug {
				fmt.Printf("fetchArt: failed to create temp file %s: %v\n", tmpPath, err)
			}
			return ""
		}
		defer out.Close()

		if _, err := io.Copy(out, resp.Body); err != nil {
			if debug {
				fmt.Printf("fetchArt: failed to copy HTTP body: %v\n", err)
			}
			out.Close()
			os.Remove(tmpPath)
			return ""
		}

		abs, _ := filepath.Abs(tmpPath)
		return "file://" + abs
	}

	return artURL
}

// ------------------- DBus Helpers -------------------

// ToInt64Micros attempts to convert a DBus value to int64 microseconds.
func ToInt64Micros(v any) int64 {
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

// toString converts common DBus types to string.
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

// ------------------- Player Discovery (Logic from second script) -------------------

// FindActivePlayer selects the best available MPRIS player name.
// It prioritizes a player whose status is 'Playing'. If none is playing, it returns the first one found.
func FindActivePlayer(conn *dbus.Conn) string {
	var names []string
	err := conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names)
	if err != nil {
		if debug {
			fmt.Printf("ListNames error: %v\n", err)
		}
		return ""
	}

	var allPlayers []string

	for _, n := range names {
		if strings.HasPrefix(n, "org.mpris.MediaPlayer2.") {
			allPlayers = append(allPlayers, n)
			
			// Check if player is currently playing
			obj := conn.Object(n, "/org/mpris/MediaPlayer2")
			call := obj.Call("org.freedesktop.DBus.Properties.Get", 0, "org.mpris.MediaPlayer2.Player", "PlaybackStatus")
			if call.Err == nil {
				if status, ok := call.Body[0].(string); ok && status == "Playing" {
					return n // Return 'Playing' player immediately
				}
			}
		}
	}

	// Fallback: return first available player if no one is playing
	if len(allPlayers) > 0 {
		return allPlayers[0]
	}

	return "" // No players found
}

// ------------------- Main Extract -------------------

// GetMediaInfo connects to DBus, finds the active player, and extracts its media info.
func GetMediaInfo() (*MediaInfo, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		if debug {
			fmt.Printf("DBus connect error: %v\n", err)
		}
		return emptyMediaInfo(), err
	}
	defer conn.Close()

	player := FindActivePlayer(conn)
	if player == "" {
		return emptyMediaInfo(), nil
	}

	obj := conn.Object(player, "/org/mpris/MediaPlayer2")
	propsIface := "org.freedesktop.DBus.Properties"

	call := obj.Call(propsIface+".GetAll", 0, "org.mpris.MediaPlayer2.Player")
	if call.Err != nil {
		if debug {
			fmt.Printf("GetAll call error: %v\n", call.Err)
		}
		return emptyMediaInfo(), nil
	}

	rawProps, ok := call.Body[0].(map[string]dbus.Variant)
	if !ok {
		return emptyMediaInfo(), nil
	}

	// 1. Player Properties
	status := ""
	if s, ok := rawProps["PlaybackStatus"]; ok {
		status = toString(s.Value())
	} else {
		status = "Stopped" // Default status if not present
	}

	pos := int64(0)
	if p, ok := rawProps["Position"]; ok {
		pos = ToInt64Micros(p.Value())
	}

	// 2. Metadata
	var title, artist, album, artURL string
	var length int64 = 0

	if metaVar, ok := rawProps["Metadata"]; ok {
		if meta, ok := metaVar.Value().(map[string]dbus.Variant); ok {
			if t, ok := meta["xesam:title"]; ok {
				title = toString(t.Value())
			}
			if a, ok := meta["xesam:artist"]; ok {
				// Artist can be []string or []interface{} (using the robust logic from second script)
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
				artURL = toString(art.Value())
			}
			if l, ok := meta["mpris:length"]; ok {
				length = ToInt64Micros(l.Value())
			}
		}
	}
	
	// If title is empty, use the art URL placeholder
	if title == "" {
		title = "---"
	}
	// If artist is empty, use the art URL placeholder
	if artist == "" {
		artist = "--"
	}

	return &MediaInfo{
		Title:                  title,
		Artist:                 artist,
		Album:                  album,
		Art:                    fetchArt(artURL),
		Status:                 status,
		Playing:                status == "Playing",
		Paused:                 status == "Paused",
		Stopped:                status == "Stopped",
		SeekPositionElapsed:    FormatDurationMicros(pos),
		SeekPositionTotal:      FormatDurationMicros(length),
		SeekPositionNormalized: FormatNormalized(pos, length),
		PositionRaw:            pos,
		LengthRaw:              length,
	}, nil
}
