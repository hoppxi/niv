package btinfo

import (
	"encoding/json"
	"errors"

	"github.com/godbus/dbus/v5"
)

type BluetoothDevice struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Paired           bool   `json:"paired"`
	Connected        bool   `json:"connected"`
	RSSI             int    `json:"rssi"`
	Battery          int    `json:"battery"`
	BatteryAvailable bool   `json:"battery_available"`
}

type CurrentConnection struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Battery   int    `json:"battery"`
	Transport string `json:"transport"`
	Profile   string `json:"profile_uuid"`
	Connected bool   `json:"connected"`
}

type BluetoothInfo struct {
	Enabled           bool              `json:"enabled"`
	Connected         bool              `json:"connected"`
	CurrentConnection CurrentConnection `json:"current_connection"`
	ConnectedDevices  []BluetoothDevice `json:"connected_devices"`
	AvailableDevices  []BluetoothDevice `json:"available_devices"`
}

func asString(v dbus.Variant) (string, bool) {
	s, ok := v.Value().(string)
	return s, ok
}

func asBool(v dbus.Variant) (bool, bool) {
	b, ok := v.Value().(bool)
	return b, ok
}

func asInt(v dbus.Variant) (int, bool) {
	switch t := v.Value().(type) {
	case int:
		return t, true
	case int16:
		return int(t), true
	case int32:
		return int(t), true
	case uint8:
		return int(t), true
	case uint16:
		return int(t), true
	case uint32:
		return int(t), true
	}
	return 0, false
}

func startDiscovery(conn *dbus.Conn, adapter dbus.ObjectPath) {
	obj := conn.Object("org.bluez", adapter)
	_ = obj.Call("org.bluez.Adapter1.StartDiscovery", 0).Err
}

func GetBluetoothInfo() (*BluetoothInfo, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}

	obj := conn.Object("org.bluez", "/")
	var managed map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	if err := obj.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&managed); err != nil {
		return nil, err
	}

	info := &BluetoothInfo{
		Enabled:           false,
		Connected:         false,
		CurrentConnection: CurrentConnection{},
		ConnectedDevices:  []BluetoothDevice{},
		AvailableDevices:  []BluetoothDevice{},
	}

	adapters := []dbus.ObjectPath{}
	for path, ifaces := range managed {
		if a, ok := ifaces["org.bluez.Adapter1"]; ok {
			adapters = append(adapters, path)
			if p, ok := a["Powered"]; ok {
				if b, ok := asBool(p); ok && b {
					info.Enabled = true
				}
			}
		}
	}

	if len(adapters) == 0 {
		return info, errors.New("no bluetooth adapters")
	}

	for _, ad := range adapters {
		startDiscovery(conn, ad)
	}

	var activeDevice dbus.ObjectPath
	var activeTransport dbus.ObjectPath
	activeProfile := ""

	for path, ifaces := range managed {
		if mt, ok := ifaces["org.bluez.MediaTransport1"]; ok {
			if s, ok := mt["State"]; ok {
				if st, ok := asString(s); ok && st == "active" {
					if d, ok := mt["Device"]; ok {
						if dp, ok := d.Value().(dbus.ObjectPath); ok {
							activeDevice = dp
							activeTransport = path
						}
					}
					if u, ok := mt["UUID"]; ok {
						activeProfile, _ = asString(u)
					}
					break
				}
			}
		}
	}

	for path, ifaces := range managed {
		dev, ok := ifaces["org.bluez.Device1"]
		if !ok {
			continue
		}

		id := string(path)
		name := ""
		if n, ok := dev["Name"]; ok {
			name, _ = asString(n)
		}

		paired := false
		if p, ok := dev["Paired"]; ok {
			paired, _ = asBool(p)
		}

		connected := false
		if c, ok := dev["Connected"]; ok {
			connected, _ = asBool(c)
		}

		rssi := 0
		if r, ok := dev["RSSI"]; ok {
			if v, ok := asInt(r); ok {
				rssi = v
			}
		}

		batteryAvailable := false
		batteryValue := -1

		if b, ok := dev["BatteryPercentage"]; ok {
			if v, ok := asInt(b); ok {
				batteryAvailable = true
				batteryValue = v
			}
		}

		if bat, ok := ifaces["org.bluez.Battery1"]; ok {
			if p, ok := bat["Percentage"]; ok {
				if v, ok := asInt(p); ok {
					batteryAvailable = true
					batteryValue = v
				}
			}
		}

		device := BluetoothDevice{
			ID:               id,
			Name:             name,
			Paired:           paired,
			Connected:        connected,
			RSSI:             rssi,
			Battery:          -1,
			BatteryAvailable: batteryAvailable,
		}

		if activeDevice != "" && path == activeDevice {
			info.CurrentConnection = CurrentConnection{
				ID:        id,
				Name:      name,
				Battery:   batteryValue,
				Transport: string(activeTransport),
				Profile:   activeProfile,
				Connected: true,
			}
			info.Connected = true
			info.ConnectedDevices = append(info.ConnectedDevices, device)
			continue
		}

		if paired {
			info.ConnectedDevices = append(info.ConnectedDevices, device)
		} else if rssi != 0 {
			info.AvailableDevices = append(info.AvailableDevices, device)
		}
	}

	return info, nil
}

func GetBluetoothInfoJSON() ([]byte, error) {
	info, err := GetBluetoothInfo()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(info, "", "  ")
}
