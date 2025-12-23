package netinfo

import (
	"encoding/json"

	"github.com/godbus/dbus/v5"
)

type Network struct {
	ID       string `json:"id"`
	SSID     string `json:"ssid"`
	Type     string `json:"type"` // wifi | ethernet
	Strength uint8  `json:"strength"`
	Active   bool   `json:"active"`
}

type CurrentConnection struct {
	SSID     string `json:"ssid"`
	Type     string `json:"type"`
	Strength uint8  `json:"strength"`
}

type SavedConnection struct {
	Name string `json:"name"`
	Type string `json:"type"` // wifi, ethernet, etc
	UUID string `json:"uuid"`
}

type NetworkInfo struct {
	Enabled           bool               `json:"enabled"`   // WiFi radio
	Connected         bool               `json:"connected"` // any active connection
	CurrentConnection *CurrentConnection `json:"current_connection,omitempty"`
	Networks          []Network          `json:"networks"`
	SavedConnections  []SavedConnection  `json:"saved_connections"`
}

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

	var wifiEnabled bool
	err = nm.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.NetworkManager", "WirelessEnabled",
	).Store(&wifiEnabled)
	if err != nil {
		return nil, err
	}

	info := &NetworkInfo{
		Enabled:  wifiEnabled,
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

	saved, err := getSavedConnections(conn)
	if err == nil {
		info.SavedConnections = saved
	} else {
		info.SavedConnections = []SavedConnection{}
	}

	return info, nil
}

func handleWiFiDevice(conn *dbus.Conn, devPath dbus.ObjectPath, info *NetworkInfo) {
	devObj := conn.Object("org.freedesktop.NetworkManager", devPath)

	netMap := map[string]Network{}
	var activeSSID string

	// Active AP
	var activeAP dbus.ObjectPath
	_ = devObj.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.NetworkManager.Device.Wireless",
		"ActiveAccessPoint",
	).Store(&activeAP)

	if activeAP != "/" {
		apObj := conn.Object("org.freedesktop.NetworkManager", activeAP)

		var ssidRaw []byte
		var strength uint8

		apObj.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.NetworkManager.AccessPoint", "Ssid",
		).Store(&ssidRaw)

		apObj.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.NetworkManager.AccessPoint", "Strength",
		).Store(&strength)

		ssid := string(ssidRaw)
		activeSSID = ssid

		info.Connected = true
		info.CurrentConnection = &CurrentConnection{
			SSID:     ssid,
			Type:     "wifi",
			Strength: strength,
		}

		netMap[ssid] = Network{
			ID:       string(activeAP),
			SSID:     ssid,
			Type:     "wifi",
			Strength: strength,
			Active:   true,
		}
	}

	// Scan APs
	var aps []dbus.ObjectPath
	if err := devObj.Call(
		"org.freedesktop.NetworkManager.Device.Wireless.GetAccessPoints", 0,
	).Store(&aps); err != nil {
		return
	}

	for _, ap := range aps {
		apObj := conn.Object("org.freedesktop.NetworkManager", ap)

		var ssidRaw []byte
		var strength uint8

		if err := apObj.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.NetworkManager.AccessPoint", "Ssid",
		).Store(&ssidRaw); err != nil {
			continue
		}

		apObj.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.NetworkManager.AccessPoint", "Strength",
		).Store(&strength)

		ssid := string(ssidRaw)
		if ssid == "" {
			continue
		}

		// Skip duplicate active AP
		if ssid == activeSSID {
			continue
		}

		netMap[ssid] = Network{
			ID:       string(ap),
			SSID:     ssid,
			Type:     "wifi",
			Strength: strength,
			Active:   false,
		}
	}

	for _, n := range netMap {
		info.Networks = append(info.Networks, n)
	}
}

func handleEthernetDevice(conn *dbus.Conn, devPath dbus.ObjectPath, info *NetworkInfo) {
	devObj := conn.Object("org.freedesktop.NetworkManager", devPath)

	var activeConn dbus.ObjectPath
	err := devObj.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.NetworkManager.Device", "ActiveConnection",
	).Store(&activeConn)

	active := err == nil && activeConn != "/"

	if active {
		info.Connected = true
		info.CurrentConnection = &CurrentConnection{
			SSID:     "Ethernet",
			Type:     "ethernet",
			Strength: 100,
		}
	}

	info.Networks = append(info.Networks, Network{
		ID:       string(devPath),
		SSID:     "Ethernet",
		Type:     "ethernet",
		Strength: 100,
		Active:   active,
	})
}

func getSavedConnections(conn *dbus.Conn) ([]SavedConnection, error) {
	nmSettings := conn.Object("org.freedesktop.NetworkManager", "/org/freedesktop/NetworkManager/Settings")

	var paths []dbus.ObjectPath
	err := nmSettings.Call("org.freedesktop.NetworkManager.Settings.ListConnections", 0).Store(&paths)
	if err != nil {
		return nil, err
	}

	saved := make([]SavedConnection, 0)
	for _, p := range paths {
		connObj := conn.Object("org.freedesktop.NetworkManager", p)

		var settings map[string]map[string]dbus.Variant
		err := connObj.Call("org.freedesktop.NetworkManager.Settings.Connection.GetSettings", 0).Store(&settings)
		if err != nil {
			continue
		}

		id, ctype, uuid := "", "", ""
		if c, ok := settings["connection"]; ok {
			if v, ok := c["id"]; ok {
				id = v.Value().(string)
			}
			if v, ok := c["type"]; ok {
				ctype = v.Value().(string)
			}
			if v, ok := c["uuid"]; ok {
				uuid = v.Value().(string)
			}
		}

		saved = append(saved, SavedConnection{
			Name: id,
			Type: ctype,
			UUID: uuid,
		})
	}

	return saved, nil
}

func GetNetworkInfoJSON() ([]byte, error) {
	info, err := GetNetworkInfo()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(info, "", "  ")
}
