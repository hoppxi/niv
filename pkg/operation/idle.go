package operation

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

type IdleController struct{}

var Idle = &IdleController{}

func (i *IdleController) Inhibit() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	obj := conn.Object("org.freedesktop.ScreenSaver", "/org/freedesktop/ScreenSaver")
	var cookie uint32
	err = obj.Call("org.freedesktop.ScreenSaver.Inhibit", 0, "wigo", "Doing something important").Store(&cookie)
	if err != nil {
		return err
	}

	fmt.Println("Idle inhibition enabled via D-Bus")
	return nil
}

func (i *IdleController) UnInhibit() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	obj := conn.Object("org.freedesktop.ScreenSaver", "/org/freedesktop/ScreenSaver")
	err = obj.Call("org.freedesktop.ScreenSaver.UnInhibit", 0, uint32(0)).Store()
	if err != nil {
		return err
	}

	fmt.Println("Idle inhibition disabled via D-Bus")
	return nil
}

func (i *IdleController) Toggle() error {
	if err := i.UnInhibit(); err != nil {
		return i.Inhibit()
	}
	return nil
}
