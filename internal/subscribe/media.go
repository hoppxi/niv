package subscribe

import (
	"log"
	"strings"

	"github.com/godbus/dbus/v5"
)

// MediaEvents returns a channel of MediaEvent whenever any MPRIS player's
// properties change. Fully parses signal body and identifies player.
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

		// Listen for all PropertiesChanged signals on the bus
		call1 := conn.BusObject().Call(
			"org.freedesktop.DBus.AddMatch", 0,
			"type='signal',interface='org.freedesktop.DBus.Properties'",
		)
		if call1.Err != nil {
			log.Printf("MediaEvents: PropertiesChanged AddMatch error: %v", call1.Err)
		}
		// Also listen for Seeked signals globally
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

			// Filter only MPRIS players
			if !strings.HasPrefix(string(sig.Path), "/org/mpris/MediaPlayer2") {
				continue
			}

			// Player is derived from the D-Bus Object Path
			player := string(sig.Path)

			// Use a switch to handle different signal names
			switch sig.Name {
			case "org.freedesktop.DBus.Properties.PropertiesChanged":
				// --- ROBUST LOGIC FOR PropertiesChanged ---
				// Signal signature: (s a{sv} as)
				// 0: Interface Name (string)
				// 1: Changed Properties (map[string]dbus.Variant)
				// 2: Invalidated Properties ([]string)
				
				// Need at least 3 arguments
				if len(sig.Body) < 3 {
					continue
				}

				// 1. Check if the interface is the MPRIS Player interface
				if iface, ok := sig.Body[0].(string); !ok || (iface != "org.mpris.MediaPlayer2.Player" && iface != "org.mpris.MediaPlayer2") {
					continue // Ignore if not a recognized MPRIS interface
				}

				// 2. sig.Body[1] is a map[string]dbus.Variant with changed properties
				if props, ok := sig.Body[1].(map[string]dbus.Variant); ok {
					for prop, val := range props {
						out <- MediaEvent{
							Player:   player,
							Property: prop,
							Value:    val.Value(),
						}
					}
				}

				// 3. sig.Body[2] is an array of invalidated property names
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
				// --- Seeked SIGNAL ---
				// This signal means a seek just occurred.
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