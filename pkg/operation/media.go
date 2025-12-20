package operation

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/hoppxi/wigo/pkg/mediainfo"
)

type media struct{}

var Media media

var (
	dbusConn *dbus.Conn
	connOnce sync.Once
)

// getDBusConnection initializes and returns DBus connection.
func getDBusConnection() (*dbus.Conn, error) {
	var err error
	connOnce.Do(func() {
		dbusConn, err = dbus.ConnectSessionBus()
		if err != nil {
			dbusConn = nil
			return
		}
	})

	if dbusConn == nil {
		return nil, fmt.Errorf("dbus connection error: %v", err)
	}
	return dbusConn, nil
}

// ControlMedia controls media playback (play, pause, next, previous, stop, play-pause)
func (m *media) ControlMedia(action string) error {
	conn, err := getDBusConnection()
	if err != nil {
		return err
	}

	method := mapActionToMethod(action)
	if method == "" {
		return fmt.Errorf("unknown action: %s", action)
	}

	player := mediainfo.FindActivePlayer(conn)

	obj := conn.Object(player, "/org/mpris/MediaPlayer2")

	// Call the method directly without status checking, relying on MPRIS spec.
	// NOTE: If the media player seeks 5 seconds back here, it is the player's internal setting,
	// not this code.
	call := obj.Call("org.mpris.MediaPlayer2.Player."+method, 0)
	return call.Err
}

// SetMediaPosition seeks the current media to a specific percentage (0-100).
func (m *media) SetMediaPosition(percent float64) error {
	// 1. Validate and clamp input percentage
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	conn, err := getDBusConnection()
	if err != nil {
		return err
	}

	player := mediainfo.FindActivePlayer(conn)

	obj := conn.Object(player, "/org/mpris/MediaPlayer2")
	propsIface := "org.freedesktop.DBus.Properties"

	// 2. Get all required properties in a single call: Metadata and Position
	call := obj.Call(propsIface+".GetAll", 0, "org.mpris.MediaPlayer2.Player")
	if call.Err != nil {
		return fmt.Errorf("GetAll call error: %v", call.Err)
	}

	rawProps, ok := call.Body[0].(map[string]dbus.Variant)
	if !ok {
		return errors.New("invalid properties format")
	}

	var length int64 = 0
	var trackID dbus.ObjectPath = "/org/mpris/MediaPlayer2/Track/1" // Default/Fallback Track ID

	if metaVar, ok := rawProps["Metadata"]; ok {
		if meta, ok := metaVar.Value().(map[string]dbus.Variant); ok {
			// 3. Extract Length from Metadata (in microseconds)
			if v, ok := meta["mpris:length"]; ok {
				length = mediainfo.ToInt64Micros(v.Value())
			}
			// 4. Extract Track ID (Required for SetPosition)
			if v, ok := meta["mpris:trackid"]; ok {
				if id, ok := v.Value().(dbus.ObjectPath); ok {
					trackID = id
				}
			}
		}
	}

	if length <= 0 {
		return errors.New("cannot seek: media has no valid duration")
	}

	// 5. Calculate Target Position in microseconds
	// This position is absolute, from the start of the track.
	targetPos := int64((percent / 100.0) * float64(length))

	// 6. Call SetPosition(TrackId, Position)
	// This is the correct MPRIS method for absolute seeking (going to a specific time/percentage).
	// Arguments: Track ID (ObjectPath) and Position (int64 in microseconds).
	call = obj.Call("org.mpris.MediaPlayer2.Player.SetPosition", 0, trackID, targetPos)
	if call.Err != nil {
		return fmt.Errorf("failed to set absolute position to %d: %v", targetPos, call.Err)
	}
	return nil
}

func mapActionToMethod(action string) string {
	switch strings.ToLower(action) {
	case "play":
		return "Play"
	case "pause":
		return "Pause"
	case "next":
		return "Next"
	case "previous", "prev":
		return "Previous"
	case "stop":
		return "Stop"
	case "play-pause":
		return "PlayPause"
	default:
		return ""
	}
}
