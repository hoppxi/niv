package subscribe

import (
	"log"
	"math"

	"github.com/godbus/dbus/v5"
)

func NetworkEvents() <-chan NetworkEvent {
	events := make(chan NetworkEvent, 10)

	go func() {
		conn, err := dbus.SystemBus()
		if err != nil {
			log.Printf("NetworkEvents: failed to connect to system bus: %v", err)
			return
		}

		signals := make(chan *dbus.Signal, 32)
		conn.Signal(signals)

		rule := "type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path_namespace='/org/freedesktop/NetworkManager'"
		if err := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule).Err; err != nil {
			log.Printf("NetworkEvents: AddMatch failed: %v", err)
		}

		prevSpeed := 0

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

			// Wi-Fi enabled/disabled
			if iface == "org.freedesktop.NetworkManager" {
				if _, ok := changed["WirelessEnabled"]; ok {
					select {
					case events <- NetworkEvent{}:
					default:
					}
					continue
				}
			}

			if iface == "org.freedesktop.NetworkManager.Device.Wireless" || iface == "org.freedesktop.NetworkManager.Device" {
				if apVar, ok := changed["ActiveAccessPoint"]; ok {
					var activeAP string
					if apVar.Value() != nil {
						activeAP = string(apVar.Value().(dbus.ObjectPath)) // convert ObjectPath → string
					}
					if activeAP == "" || activeAP == "/" {
						// Disconnected
						select {
						case events <- NetworkEvent{}:
						default:
						}
					} else {
						// Connected to new AP
						select {
						case events <- NetworkEvent{}:
						default:
						}
					}
				}

				// Speed changes
				if speedVar, ok := changed["Speed"]; ok {
					speed, ok := speedVar.Value().(uint32)
					if !ok {
						continue
					}

					if math.Abs(float64(int(speed)-prevSpeed)) >= 10 {
						prevSpeed = int(speed)
						select {
						case events <- NetworkEvent{}:
						default:
						}
					}
				}
			}
		}
	}()

	return events
}
