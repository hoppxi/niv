package btinfo

import (
	"encoding/json"
	"errors"

	"github.com/godbus/dbus/v5"
)

type BluetoothDevice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Battery int `json:"battery"`
}

type CurrentConnection struct {
	Name    string `json:"name"`
	Battery int    `json:"battery"`
}

type BluetoothInfo struct {
	Enabled           bool               `json:"enabled"`
	Connected         bool               `json:"connected"`
	CurrentConnection CurrentConnection  `json:"current_connection"`
	Devices           []BluetoothDevice  `json:"devices"`
}

func GetBluetoothInfo() (*BluetoothInfo, error) {
    conn, err := dbus.SystemBus()
    if err != nil {
        return nil, err
    }

    adapters, err := getAdaptersFromManager(conn)
    if err != nil {
        return nil, err
    }

    info := &BluetoothInfo{
        Enabled: true,
        Devices: make([]BluetoothDevice, 0),
    }

    for _, adPath := range adapters {
        devices, err := listDevices(conn, adPath)
        if err != nil {
            continue
        }
        for _, d := range devices {
            info.Devices = append(info.Devices, d)
            // check if connected
            if d.Name != "" { // or better: check Connected property
                info.Connected = true
                info.CurrentConnection = CurrentConnection{
                    Name:    d.Name,
                    Battery: 100, // could query Battery1 interface here
                }
            }
        }
    }

    return info, nil
}


func getAdaptersFromManager(conn *dbus.Conn) ([]dbus.ObjectPath, error) {
	obj := conn.Object("org.bluez", "/")
	var managedObjects map[dbus.ObjectPath]map[string]map[string]dbus.Variant

	err := obj.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).
		Store(&managedObjects)
	if err != nil {
		return nil, err
	}

	adapters := []dbus.ObjectPath{}
	for path, ifaces := range managedObjects {
		if _, ok := ifaces["org.bluez.Adapter1"]; ok {
			adapters = append(adapters, path)
		}
	}

	if len(adapters) == 0 {
		return nil, errors.New("no bluetooth adapters found")
	}

	return adapters, nil
}

func listDevices(conn *dbus.Conn, adapterPath dbus.ObjectPath) ([]BluetoothDevice, error) {
	obj := conn.Object("org.bluez", "/")
	var managedObjects map[dbus.ObjectPath]map[string]map[string]dbus.Variant

	err := obj.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).
		Store(&managedObjects)
	if err != nil {
		return nil, err
	}

	devices := []BluetoothDevice{}

	for path, ifaces := range managedObjects {
		if dev, ok := ifaces["org.bluez.Device1"]; ok {
			if len(path) > len(adapterPath) && string(path[:len(adapterPath)]) == string(adapterPath) {
				id := string(path)
				name := ""
				battery := 100
				if n, ok := dev["Name"]; ok {
					name = n.Value().(string)
				}
				if b, ok := dev["BatteryPercentage"]; ok {
					battery = int(b.Value().(uint8))
				}
				devices = append(devices, BluetoothDevice{
					ID:   id,
					Name: name,
					Battery: battery,
				})
			}
		}
	}

	return devices, nil
}

func GetBluetoothInfoJSON() ([]byte, error) {
	info, err := GetBluetoothInfo()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(info, "", "  ")
}
