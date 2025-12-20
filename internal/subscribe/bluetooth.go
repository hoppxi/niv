package subscribe

import (
	"log"
	"strings"

	"github.com/godbus/dbus/v5"
)

func BluetoothEvents() <-chan BluetoothEvent {
	events := make(chan BluetoothEvent, 10)

	go func() {
		conn, err := dbus.SystemBus()
		if err != nil {
			log.Printf("BluetoothEvents: failed to connect to system bus: %v", err)
			return
		}

		signals := make(chan *dbus.Signal, 32)
		conn.Signal(signals)

		// Match signals for PropertiesChanged from BlueZ
		rule := "type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path_namespace='/org/bluez'"
		if err := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule).Err; err != nil {
			log.Printf("BluetoothEvents: AddMatch failed: %v", err)
		}

		for sig := range signals {
			if sig == nil || sig.Name != "org.freedesktop.DBus.Properties.PropertiesChanged" {
				continue
			}
			if len(sig.Body) < 2 {
				continue
			}

			iface, ok := sig.Body[0].(string)
			if !ok {
				continue
			}

			changed, ok := sig.Body[1].(map[string]dbus.Variant)
			if !ok {
				continue
			}

			// Adapter properties
			if strings.HasPrefix(iface, "org.bluez.Adapter1") {
				if _, ok := changed["Powered"]; ok {
					select {
					case events <- BluetoothEvent{}:
					default:
					}
					continue
				}
			}

			// Device properties
			if strings.HasPrefix(iface, "org.bluez.Device1") {
				if _, ok := changed["Connected"]; ok {
					select {
					case events <- BluetoothEvent{}:
					default:
					}
					continue
				}
			}
		}
	}()

	return events
}
