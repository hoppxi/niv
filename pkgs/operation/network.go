package operation

import (
	"errors"
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
)

type network struct{}

// Network is the exported instance.
var Network network

const (
	nmBus        = "org.freedesktop.NetworkManager"
	nmPath       = "/org/freedesktop/NetworkManager"
	nmInterface  = "org.freedesktop.NetworkManager"
	wifiInterface = "org.freedesktop.NetworkManager.Device.Wireless"
	networkDeviceInterface = "org.freedesktop.NetworkManager.Device"
	settingsInterface = "org.freedesktop.NetworkManager.Settings"
	
	// NM_DEVICE_TYPE_WIFI = 2
	nmDeviceTypeWifi = 2
)

// Connect connects to a WiFi network. 
// It creates a new connection profile on the fly and activates it.
func (n *network) Connect(ssid, password string) error {
	conn, err := dbus.ConnectSystemBus() // NM is on System Bus
	if err != nil {
		return err
	}
	defer conn.Close()

	// 1. Find the WiFi Device
	devicePath, err := getWifiDevice(conn)
	if err != nil {
		return err
	}

	// 2. Prepare Connection Settings (D-Bus Map of Maps)
	// Structure: [connection][id/type/uuid], [802-11-wireless][ssid/mode], [802-11-wireless-security][key-mgmt/auth-alg/psk]
	
	ssidBytes := []byte(ssid)
	
	connectionSettings := make(map[string]map[string]dbus.Variant)

	// 'connection' section
	connectionSettings["connection"] = map[string]dbus.Variant{
		"id":   dbus.MakeVariant("Auto " + ssid),
		"type": dbus.MakeVariant("802-11-wireless"),
	}

	// '802-11-wireless' section
	connectionSettings["802-11-wireless"] = map[string]dbus.Variant{
		"ssid": dbus.MakeVariant(ssidBytes),
		"mode": dbus.MakeVariant("infrastructure"),
	}

	// '802-11-wireless-security' section (only if password provided)
	if password != "" {
		connectionSettings["802-11-wireless-security"] = map[string]dbus.Variant{
			"key-mgmt": dbus.MakeVariant("wpa-psk"),
			"psk":      dbus.MakeVariant(password),
		}
	}

	// 3. Call AddAndActivateConnection
	// Method: AddAndActivateConnection(a{sa{sv}} connection, o device, o specific_object) -> (o path, o active_connection)
	nmObj := conn.Object(nmBus, dbus.ObjectPath(nmPath))
	
	var path, activeConn dbus.ObjectPath
	
	// Note: We use the main NM object to add the connection
	call := nmObj.Call("org.freedesktop.NetworkManager.AddAndActivateConnection", 0, 
		connectionSettings, 
		devicePath, 
		dbus.ObjectPath("/"), // No specific object (AP)
	)

	if call.Err != nil {
		return fmt.Errorf("failed to add/activate connection: %v", call.Err)
	}

	// Parse response to ensure it worked
	if err := call.Store(&path, &activeConn); err != nil {
		return fmt.Errorf("connection initiated but failed to parse response: %v", err)
	}

	return nil
}

// AirplaneMode turns off ALL networking radios (WiFi, WWAN, etc).
func (n *network) AirplaneMode() error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	nmObj := conn.Object(nmBus, dbus.ObjectPath(nmPath))
	
	// Set "WirelessEnabled" to false
	// Set "WwanEnabled" to false
	// Or use global "Enable" (NetworkingEnabled)
	
	// Let's just kill the radios specifically, as "Enable(false)" kills the daemon logic sometimes.
	// Actually, standard Airplane mode logic is turning off WirelessEnabled.
	
	err = nmObj.SetProperty(nmInterface + ".WirelessEnabled", false)
	if err != nil {
		return fmt.Errorf("failed to disable wireless: %v", err)
	}
	
	// Try to disable WWAN (Cellular) too, ignore error if device doesn't have it
	_ = nmObj.SetProperty(nmInterface + ".WwanEnabled", false)

	return nil
}

// DisableWiFi turns off only the WiFi radio.
func (n *network) DisableWiFi() error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	nmObj := conn.Object(nmBus, dbus.ObjectPath(nmPath))
	return nmObj.SetProperty(nmInterface + ".WirelessEnabled", false)
}

// EnableWiFi turns on the WiFi radio (Helper, needed for scanning).
func (n *network) EnableWiFi() error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	nmObj := conn.Object(nmBus, dbus.ObjectPath(nmPath))
	return nmObj.SetProperty(nmInterface + ".WirelessEnabled", true)
}

// ScanNetworks re-initiates a scan and returns found SSIDs.
func (n *network) ScanNetworks() ([]string, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 1. Get WiFi Device
	devicePath, err := getWifiDevice(conn)
	if err != nil {
		return nil, err
	}
	
	deviceObj := conn.Object(nmBus, devicePath)

	// 2. Request Scan
	// RequestScan(a{sv} options). Options can be empty.
	options := make(map[string]dbus.Variant)
	call := deviceObj.Call(wifiInterface + ".RequestScan", 0, options)
	if call.Err != nil {
		// If a scan was recently requested, NM might error. We continue to read cached APs.
		// fmt.Println("Scan request warning:", call.Err) 
	} else {
		// Wait briefly for scan to populate (D-Bus is async, strictly we should listen to signals, 
		// but a short sleep is the simple synchronous way)
		time.Sleep(2 * time.Second)
	}

	// 3. Get Access Points
	// GetAllAccessPoints() -> [o]
	var apPaths []dbus.ObjectPath
	err = deviceObj.Call(wifiInterface + ".GetAllAccessPoints", 0).Store(&apPaths)
	if err != nil {
		// Fallback: Read property if method fails
		variant, pErr := deviceObj.GetProperty(wifiInterface + ".AccessPoints")
		if pErr != nil {
			return nil, fmt.Errorf("failed to get AP list: %v", pErr)
		}
		apPaths = variant.Value().([]dbus.ObjectPath)
	}

	// 4. Resolve SSIDs
	seen := make(map[string]bool)
	var ssids []string

	for _, apPath := range apPaths {
		apObj := conn.Object(nmBus, apPath)
		
		// Read SSID property
		v, err := apObj.GetProperty("org.freedesktop.NetworkManager.AccessPoint.Ssid")
		if err != nil {
			continue
		}
		
		// SSID is returned as []byte
		ssidBytes := v.Value().([]byte)
		ssidStr := string(ssidBytes)

		if ssidStr != "" && !seen[ssidStr] {
			ssids = append(ssids, ssidStr)
			seen[ssidStr] = true
		}
	}

	return ssids, nil
}

// --- Helpers ---

func getWifiDevice(conn *dbus.Conn) (dbus.ObjectPath, error) {
	nmObj := conn.Object(nmBus, dbus.ObjectPath(nmPath))

	// Get list of all devices
	var devicePaths []dbus.ObjectPath
	err := nmObj.Call(nmInterface + ".GetDevices", 0).Store(&devicePaths)
	if err != nil {
		return "", fmt.Errorf("failed to list devices: %v", err)
	}

	for _, path := range devicePaths {
		dObj := conn.Object(nmBus, path)
		
		// Check Device Type
		v, err := dObj.GetProperty(networkDeviceInterface + ".DeviceType")
		if err != nil {
			continue
		}
		
		// NM_DEVICE_TYPE_WIFI = 2
		if v.Value().(uint32) == nmDeviceTypeWifi {
			return path, nil
		}
	}

	return "", errors.New("no WiFi device found")
}