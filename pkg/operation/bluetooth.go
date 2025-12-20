package operation

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

type bluetooth struct{}

// Bluetooth is the exported instance.
var Bluetooth bluetooth

const (
	bluezBus    = "org.bluez"
	adapterPath = "/org/bluez/hci0" // Default adapter path (hci0)
	adapterInterface = "org.bluez.Adapter1"
	deviceInterface = "org.bluez.Device1"
)

// Disable turns off the Bluetooth radio by setting the Adapter's Powered property to false.
func (b *bluetooth) Disable() error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("dbus connection error: %v", err)
	}
	defer conn.Close()

	adapterObj := conn.Object(bluezBus, dbus.ObjectPath(adapterPath))

	// Set the Powered property on the Adapter interface to false
	err = adapterObj.SetProperty(adapterInterface+".Powered", dbus.MakeVariant(false))
	if err != nil {
		return fmt.Errorf("failed to disable bluetooth: %v", err)
	}

	return nil
}

// Connect attempts to connect to a specific Bluetooth device given its D-Bus path.
// This requires the device to be paired first, which is often done outside this scope.
// 'device' is the MAC address, and we try to find the corresponding D-Bus path.
func (b *bluetooth) Connect(macAddress string) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("dbus connection error: %v", err)
	}
	defer conn.Close()

	// 1. Convert MAC to BlueZ device path format
	// e.g., 1A:2B:3C:4D:5E:6F -> /org/bluez/hci0/dev_1A_2B_3C_4D_5E_6F
	devicePath, err := findDevicePathByMAC(conn, macAddress)
	if err != nil {
		return err
	}

	// 2. Get the device object and call the Connect method
	deviceObj := conn.Object(bluezBus, devicePath)
	
	// Ensure the device is powered on and accessible
	if !isAdapterPowered(conn) {
		return errors.New("bluetooth adapter is not powered on")
	}

	// Call the Connect() method on the Device1 interface
	call := deviceObj.Call(deviceInterface+".Connect", 0)
	if call.Err != nil {
		return fmt.Errorf("failed to connect to device %s: %v", macAddress, call.Err)
	}

	return nil
}

// Scan triggers a Bluetooth device discovery process.
// Note: BlueZ scanning is asynchronous. We start the scan and return success.
func (b *bluetooth) Scan() error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("dbus connection error: %v", err)
	}
	defer conn.Close()

	adapterObj := conn.Object(bluezBus, dbus.ObjectPath(adapterPath))
	
	// Ensure adapter is powered on before scanning
	if !isAdapterPowered(conn) {
		// Attempt to turn it on
		err = adapterObj.SetProperty(adapterInterface+".Powered", dbus.MakeVariant(true))
		if err != nil {
			return fmt.Errorf("failed to power on adapter for scan: %v", err)
		}
		// Give it a moment to initialize
		time.Sleep(1 * time.Second) 
	}

	// Call the StartDiscovery method on the Adapter1 interface
	call := adapterObj.Call(adapterInterface+".StartDiscovery", 0)
	if call.Err != nil {
		return fmt.Errorf("failed to start bluetooth scanning: %v", call.Err)
	}

	return nil
}

// --- Helpers ---

// isAdapterPowered checks the current state of the default adapter.
func isAdapterPowered(conn *dbus.Conn) bool {
	adapterObj := conn.Object(bluezBus, dbus.ObjectPath(adapterPath))
	
	var powered bool
	err := adapterObj.StoreProperty(adapterInterface + ".Powered", &powered)
	if err != nil {
		return false // Assume off or error
	}
	return powered
}

// findDevicePathByMAC attempts to locate the D-Bus path for a given device MAC address.
func findDevicePathByMAC(conn *dbus.Conn, macAddress string) (dbus.ObjectPath, error) {
	// Format MAC for D-Bus path (e.g., 1A:2B:3C:4D:5E:6F -> dev_1A_2B_3C_4D_5E_6F)
	pathSuffix := "dev_" + strings.ReplaceAll(macAddress, ":", "_")
	
	// BlueZ devices are children of the adapter path
	devicePath := dbus.ObjectPath(adapterPath + "/" + pathSuffix)

	// Check if this path exists and is a BlueZ device
	// We call Introspect to see if the interface is present
	obj := conn.Object(bluezBus, devicePath)
	node, err := introspect.Call(obj)

	if err != nil {
		// Device path calculated but introspection failed (device not visible/paired/found)
		return "", fmt.Errorf("device with MAC %s not found or accessible via D-Bus: %v", macAddress, err)
	}

	// Check if the expected Device interface is present
	for _, iface := range node.Interfaces {
		if iface.Name == deviceInterface {
			return devicePath, nil
		}
	}

	return "", fmt.Errorf("device path found but it does not implement the %s interface", deviceInterface)
}