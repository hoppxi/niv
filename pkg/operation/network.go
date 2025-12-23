package operation

import (
	"errors"
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
)

type network struct{}

var Network network

const (
	nmBus                  = "org.freedesktop.NetworkManager"
	nmPath                 = "/org/freedesktop/NetworkManager"
	nmInterface            = "org.freedesktop.NetworkManager"
	wifiInterface          = "org.freedesktop.NetworkManager.Device.Wireless"
	networkDeviceInterface = "org.freedesktop.NetworkManager.Device"
	settingsInterface      = "org.freedesktop.NetworkManager.Settings"

	// NM_DEVICE_TYPE_WIFI = 2
	nmDeviceTypeWifi = 2
)

func (n *network) Connect(ssid, password string) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	devicePath, err := getWifiDevice(conn)
	if err != nil {
		return err
	}

	devObj := conn.Object(nmBus, devicePath)

	var apPaths []dbus.ObjectPath
	if err := devObj.Call(wifiInterface+".GetAccessPoints", 0).Store(&apPaths); err != nil {
		return fmt.Errorf("failed to get AP list: %v", err)
	}

	var apPath dbus.ObjectPath
	for _, ap := range apPaths {
		apObj := conn.Object(nmBus, ap)

		prop, err := apObj.GetProperty("org.freedesktop.NetworkManager.AccessPoint.Ssid")
		if err != nil {
			continue
		}

		ssidBytes := prop.Value().([]byte)
		if string(ssidBytes) == ssid {
			apPath = ap
			break
		}
	}

	if apPath == "" {
		return fmt.Errorf("AP (radio) with SSID %s not found", ssid)
	}

	connectionSettings := map[string]map[string]dbus.Variant{
		"connection": {
			"id":   dbus.MakeVariant(ssid),
			"type": dbus.MakeVariant("802-11-wireless"),
			"uuid": dbus.MakeVariant(uuid.New().String()),
		},
		"802-11-wireless": {
			"ssid": dbus.MakeVariant([]byte(ssid)),
			"mode": dbus.MakeVariant("infrastructure"),
		},
		"802-11-wireless-security": {
			"key-mgmt": dbus.MakeVariant("wpa-psk"),
			"psk":      dbus.MakeVariant(password),
		},
		"ipv4": {
			"method": dbus.MakeVariant("auto"),
		},
		"ipv6": {
			"method": dbus.MakeVariant("auto"),
		},
	}

	nmObj := conn.Object(nmBus, dbus.ObjectPath(nmPath))

	var path, active dbus.ObjectPath
	call := nmObj.Call("org.freedesktop.NetworkManager.AddAndActivateConnection", 0,
		connectionSettings,
		devicePath,
		apPath,
	)
	if call.Err != nil {
		return fmt.Errorf("AddAndActivateConnection failed: %v", call.Err)
	}

	if err := call.Store(&path, &active); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	return nil
}

func (n *network) AirplaneMode() error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	nmObj := conn.Object(nmBus, dbus.ObjectPath(nmPath))

	err = nmObj.SetProperty(nmInterface+".WirelessEnabled", false)
	if err != nil {
		return fmt.Errorf("failed to disable wireless: %v", err)
	}

	_ = nmObj.SetProperty(nmInterface+".WwanEnabled", false)

	return nil
}

func (n *network) DisableWiFi() error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	nmObj := conn.Object(nmBus, dbus.ObjectPath(nmPath))
	return nmObj.SetProperty(nmInterface+".WirelessEnabled", false)
}

func (n *network) EnableWiFi() error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	nmObj := conn.Object(nmBus, dbus.ObjectPath(nmPath))
	return nmObj.SetProperty(nmInterface+".WirelessEnabled", true)
}

func (n *network) ScanNetworks() ([]string, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	devicePath, err := getWifiDevice(conn)
	if err != nil {
		return nil, err
	}

	deviceObj := conn.Object(nmBus, devicePath)

	options := make(map[string]dbus.Variant)
	call := deviceObj.Call(wifiInterface+".RequestScan", 0, options)
	if call.Err != nil {
	} else {
		time.Sleep(2 * time.Second)
	}

	var apPaths []dbus.ObjectPath
	err = deviceObj.Call(wifiInterface+".GetAllAccessPoints", 0).Store(&apPaths)
	if err != nil {
		variant, pErr := deviceObj.GetProperty(wifiInterface + ".AccessPoints")
		if pErr != nil {
			return nil, fmt.Errorf("failed to get AP list: %v", pErr)
		}
		apPaths = variant.Value().([]dbus.ObjectPath)
	}

	seen := make(map[string]bool)
	var ssids []string

	for _, apPath := range apPaths {
		apObj := conn.Object(nmBus, apPath)

		v, err := apObj.GetProperty("org.freedesktop.NetworkManager.AccessPoint.Ssid")
		if err != nil {
			continue
		}

		ssidBytes := v.Value().([]byte)
		ssidStr := string(ssidBytes)

		if ssidStr != "" && !seen[ssidStr] {
			ssids = append(ssids, ssidStr)
			seen[ssidStr] = true
		}
	}

	return ssids, nil
}

func getWifiDevice(conn *dbus.Conn) (dbus.ObjectPath, error) {
	nmObj := conn.Object(nmBus, dbus.ObjectPath(nmPath))

	var devicePaths []dbus.ObjectPath
	err := nmObj.Call(nmInterface+".GetDevices", 0).Store(&devicePaths)
	if err != nil {
		return "", fmt.Errorf("failed to list devices: %v", err)
	}

	for _, path := range devicePaths {
		dObj := conn.Object(nmBus, path)

		v, err := dObj.GetProperty(networkDeviceInterface + ".DeviceType")
		if err != nil {
			continue
		}

		if v.Value().(uint32) == nmDeviceTypeWifi {
			return path, nil
		}
	}

	return "", errors.New("no WiFi device found")
}
