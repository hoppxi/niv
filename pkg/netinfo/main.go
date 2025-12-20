package netinfo

import (
	"encoding/json"

	"github.com/godbus/dbus/v5"
)

type Network struct {
	ID    string `json:"id"`
	SSID  string `json:"ssid"`
	Type  string `json:"type"`  // "wifi" or "ethernet"
	Speed uint32 `json:"speed"` // connection speed in Mbps
}

type CurrentConnection struct {
	SSID  string `json:"ssid"`
	Type  string `json:"type"`  // "wifi" or "ethernet"
	Speed uint32 `json:"speed"` // connection speed in Mbps
}

type NetworkInfo struct {
	Enabled           bool              `json:"enabled"`
	Connected         bool              `json:"connected"`
	CurrentConnection CurrentConnection `json:"current_connection"`
	Networks          []Network         `json:"networks"`
}

// GetNetworkInfo retrieves WiFi + Ethernet info using NetworkManager (no sudo).
func GetNetworkInfo() (*NetworkInfo, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}

	nm := conn.Object("org.freedesktop.NetworkManager", "/org/freedesktop/NetworkManager")

	var devices []dbus.ObjectPath
	err = nm.Call("org.freedesktop.NetworkManager.GetDevices", 0).Store(&devices)
	if err != nil {
		return nil, err
	}

	info := &NetworkInfo{
		Enabled:  true,
		Networks: make([]Network, 0),
	}

	for _, devPath := range devices {
		devObj := conn.Object("org.freedesktop.NetworkManager", devPath)

		// device type
		var dtype uint32
		err := devObj.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.NetworkManager.Device",
			"DeviceType",
		).Store(&dtype)
		if err != nil {
			continue
		}

		// 2 = Wi-Fi, 1 = Ethernet
		switch dtype {
		case 2: // WiFi
			handleWiFiDevice(conn, devPath, info)
		case 1: // Ethernet
			handleEthernetDevice(conn, devPath, info)
		}
	}

	return info, nil
}

func handleWiFiDevice(conn *dbus.Conn, devPath dbus.ObjectPath, info *NetworkInfo) {
	devObj := conn.Object("org.freedesktop.NetworkManager", devPath)

	// Active Access Point
	var activeAP dbus.ObjectPath
	err := devObj.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.NetworkManager.Device.Wireless",
		"ActiveAccessPoint",
	).Store(&activeAP)
	if err == nil && activeAP != "/" {
		apObj := conn.Object("org.freedesktop.NetworkManager", activeAP)

		// SSID is a byte array
		var ssidRaw []byte
		apObj.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.NetworkManager.AccessPoint",
			"Ssid",
		).Store(&ssidRaw)

		ssid := string(ssidRaw)
		speed := uint32(0)
		apObj.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.NetworkManager.AccessPoint",
			"MaxBitrate",
		).Store(&speed)

		if ssid != "" {
			info.Connected = true
			info.CurrentConnection = CurrentConnection{
				SSID:  ssid,
				Type:  "wifi",
				Speed: speed,
			}
		}

		// Add to networks list
		info.Networks = append(info.Networks, Network{
			ID:    string(activeAP),
			SSID:  ssid,
			Type:  "wifi",
			Speed: speed,
		})
	}

	// Scanned networks
	var aps []dbus.ObjectPath
	err = devObj.Call("org.freedesktop.NetworkManager.Device.Wireless.GetAccessPoints", 0).Store(&aps)
	if err != nil {
		return
	}

	for _, ap := range aps {
		apObj := conn.Object("org.freedesktop.NetworkManager", ap)
		var ssidRaw []byte
		var speed uint32

		err := apObj.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.NetworkManager.AccessPoint",
			"Ssid",
		).Store(&ssidRaw)
		if err != nil {
			continue
		}
		ssid := string(ssidRaw)

		apObj.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.NetworkManager.AccessPoint",
			"MaxBitrate",
		).Store(&speed)

		info.Networks = append(info.Networks, Network{
			ID:    string(ap),
			SSID:  ssid,
			Type:  "wifi",
			Speed: speed,
		})
	}
}

func handleEthernetDevice(conn *dbus.Conn, devPath dbus.ObjectPath, info *NetworkInfo) {
	devObj := conn.Object("org.freedesktop.NetworkManager", devPath)

	// Check connection state
	var activeConn dbus.ObjectPath
	err := devObj.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.NetworkManager.Device",
		"ActiveConnection",
	).Store(&activeConn)
	if err == nil && activeConn != "/" {
		info.Connected = true
		info.CurrentConnection = CurrentConnection{
			SSID:  "Ethernet",
			Type:  "ethernet",
			Speed: 1000, // placeholder, can try reading "Speed" property if available
		}
	}

	info.Networks = append(info.Networks, Network{
		ID:    string(devPath),
		SSID:  "Ethernet",
		Type:  "ethernet",
		Speed: 1000, // placeholder
	})
}

// Optional JSON helper
func GetNetworkInfoJSON() ([]byte, error) {
	info, err := GetNetworkInfo()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(info, "", "  ")
}
