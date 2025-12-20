package subscribe

import (
	"log"
	"strings"

	"github.com/godbus/dbus/v5"
)

// any MPRIS player
func MediaEvents() <-chan MediaEvent {
	out := make(chan MediaEvent, 20)

	go func() {
		conn, err := dbus.ConnectSessionBus()
		if err != nil {
			log.Printf("MediaEvents: failed to connect to session bus: %v", err)
			close(out)
			return
		}

		signalChan := make(chan *dbus.Signal, 50)
		conn.Signal(signalChan)

		call1 := conn.BusObject().Call(
			"org.freedesktop.DBus.AddMatch", 0,
			"type='signal',interface='org.freedesktop.DBus.Properties'",
		)
		if call1.Err != nil {
			log.Printf("MediaEvents: PropertiesChanged AddMatch error: %v", call1.Err)
		}

		call2 := conn.BusObject().Call(
			"org.freedesktop.DBus.AddMatch", 0,
			"type='signal',interface='org.mpris.MediaPlayer2.Player',member='Seeked'",
		)
		if call2.Err != nil {
			log.Printf("MediaEvents: Seeked AddMatch error: %v", call2.Err)
		}

		for sig := range signalChan {
			if sig == nil || sig.Path == "" {
				continue
			}

			if !strings.HasPrefix(string(sig.Path), "/org/mpris/MediaPlayer2") {
				continue
			}

			player := string(sig.Path)

			switch sig.Name {
			case "org.freedesktop.DBus.Properties.PropertiesChanged":

				// Need at least 3 arguments
				if len(sig.Body) < 3 {
					continue
				}

				if iface, ok := sig.Body[0].(string); !ok || (iface != "org.mpris.MediaPlayer2.Player" && iface != "org.mpris.MediaPlayer2") {
					continue // Ignore if not a recognized MPRIS interface
				}

				if props, ok := sig.Body[1].(map[string]dbus.Variant); ok {
					for prop, val := range props {
						out <- MediaEvent{
							Player:   player,
							Property: prop,
							Value:    val.Value(),
						}
					}
				}

				if invalidated, ok := sig.Body[2].([]string); ok {
					for _, prop := range invalidated {
						out <- MediaEvent{
							Player:   player,
							Property: prop,
							Value:    nil, // property invalidated
						}
					}
				}

			case "org.mpris.MediaPlayer2.Player.Seeked":

				out <- MediaEvent{
					Player:   player,
					Property: "Position", // Trigger property
					Value:    nil,
				}
			}
		}
	}()

	return out
}
