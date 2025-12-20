package operation

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

type battery struct{}

var Battery battery

func (b *battery) SetPowerMode(mode string) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	// The object path and bus name for the daemon
	obj := conn.Object("org.freedesktop.PowerProfiles", "/org/freedesktop/PowerProfiles")

	// We use the Properties.Set method because ActiveProfile is a property
	err = obj.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		"org.freedesktop.PowerProfiles",
		"ActiveProfile",
		dbus.MakeVariant(mode),
	).Store()

	if err != nil {
		return fmt.Errorf("failed to set power mode to %s: %w", mode, err)
	}

	return nil
}
