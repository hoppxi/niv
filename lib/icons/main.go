/*
A Go program that keeps track of system status and update their corresponding icons in the eww status bar.
It monitors network status, battery level, Bluetooth connectivity, screen brightness, nightlight status,
volume level, notifications, and microphone status using D-Bus and file system notifications.
*/
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/godbus/dbus/v5"
)

const (
	volumePollInterval       = 3 * time.Second
	notificationPollInterval = 5 * time.Second
	ewwBatchDelay            = 80 * time.Millisecond
)

var (
	stateMu sync.Mutex
	state   = map[string]string{
		"NETWORK_ICON":      "wifi-off",
		"VOLUME_ICON":       "volume-x",
		"BATTERY_ICON":      "battery-warning",
		"BLUETOOTH_ICON":    "bluetooth-off",
		"NIGHTLIGHT_ICON":   "eclipse",
		"BRIGHTNESS_ICON":   "sun-dim",
		"AIRPLANE_ICON":     "plane",
		"NOTIFICATION_ICON": "bell-off",
		"MIC_ICON":          "mic-off",
	}

	pending      = map[string]string{}
	pendingTimer *time.Timer
)

func scheduleUpdate(key, val string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	if prev, ok := state[key]; ok && prev == val {
		return
	}
	pending[key] = val
	if pendingTimer == nil {
		pendingTimer = time.AfterFunc(ewwBatchDelay, flushPending)
	} else {
		pendingTimer.Reset(ewwBatchDelay)
	}
}

// flushPending runs with NO lock held (it will copy under lock then run)
func flushPending() {
	stateMu.Lock()
	updates := make([]string, 0, len(pending))
	for k, v := range pending {
		updates = append(updates, fmt.Sprintf("%s=%s", k, escapeArg(v)))
		state[k] = v
	}
	pending = map[string]string{}
	pendingTimer = nil
	stateMu.Unlock()

	// run eww outside lock (in goroutine so it doesn't block event loop)
	if len(updates) > 0 {
		go runEwwUpdates(updates)
	}
}

func runEwwUpdates(kv []string) {
	if len(kv) == 0 {
		return
	}
	args := append([]string{"update"}, kv...)
	cmd := exec.Command("eww", args...)
	_ = cmd.Run() // ignore errors
}

func escapeArg(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "=", " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return "''"
	}
	return s
}

// NetworkManager
func watchNetwork(sys *dbus.Conn) {
	if sys == nil {
		fmt.Fprintln(os.Stderr, "watchNetwork: no system bus connection")
		return
	}

	if call := sys.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',sender='org.freedesktop.NetworkManager'"); call.Err != nil {
		fmt.Fprintln(os.Stderr, "watchNetwork AddMatch error:", call.Err)
	}

	c := make(chan *dbus.Signal, 32)
	sys.Signal(c)

	st := getNetworkStatusDBus(sys)
	scheduleUpdate("NETWORK_ICON", st)

	go func() {
		for sig := range c {
			if sig == nil {
				continue
			}
			if sig.Sender == "org.freedesktop.NetworkManager" {
				st := getNetworkStatusDBus(sys)
				scheduleUpdate("NETWORK_ICON", st)
			}
		}
	}()
}

func getNetworkStatusDBus(sys *dbus.Conn) string {
	nm := sys.Object("org.freedesktop.NetworkManager", dbus.ObjectPath("/org/freedesktop/NetworkManager"))
	var devPaths []dbus.ObjectPath
	if err := nm.Call("org.freedesktop.NetworkManager.GetDevices", 0).Store(&devPaths); err != nil {
		return "wifi-off"
	}
	ethernetUp := false
	var wifiDevice string
	for _, p := range devPaths {
		dev := sys.Object("org.freedesktop.NetworkManager", p)
		var devType uint32
		_ = dev.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.NetworkManager.Device", "DeviceType").Store(&devType)
		var state uint32
		_ = dev.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.NetworkManager.Device", "State").Store(&state)
		if devType == 1 && state == 100 {
			ethernetUp = true
		}
		if devType == 2 && state == 100 {
			var iface string
			_ = dev.Call("org.freedesktop.DBus.Properties.Get", 0,
				"org.freedesktop.NetworkManager.Device", "Interface").Store(&iface)
			wifiDevice = iface
		}
	}
	if ethernetUp {
		return "ethernet-port"
	}
	if wifiDevice == "" {
		return "wifi-off"
	}
	// try strength if possible
	for _, p := range devPaths {
		dev := sys.Object("org.freedesktop.NetworkManager", p)
		var devType uint32
		_ = dev.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.NetworkManager.Device", "DeviceType").Store(&devType)
		if devType != 2 {
			continue
		}
		var activeAp dbus.ObjectPath
		_ = dev.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.NetworkManager.Device.Wireless", "ActiveAccessPoint").Store(&activeAp)
		if activeAp != "" {
			ap := sys.Object("org.freedesktop.NetworkManager", activeAp)
			var strength uint8
			_ = ap.Call("org.freedesktop.DBus.Properties.Get", 0,
				"org.freedesktop.NetworkManager.AccessPoint", "Strength").Store(&strength)
			switch {
			case strength > 80:
				return "wifi"
			case strength > 60:
				return "wifi-high"
			case strength > 40:
				return "wifi-low"
			case strength > 20:
				return "wifi-zero"
			default:
				return "wifi"
			}
		}
	}
	return "wifi"
}

// UPower - used for battery status
func watchUPower(sys *dbus.Conn) {
	if sys == nil {
		fmt.Fprintln(os.Stderr, "watchUPower: no system bus connection")
		return
	}

	if call := sys.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',sender='org.freedesktop.UPower'"); call.Err != nil {
		fmt.Fprintln(os.Stderr, "watchUPower AddMatch error:", call.Err)
	}

	c := make(chan *dbus.Signal, 16)
	sys.Signal(c)

	scheduleUpdate("BATTERY_ICON", getBatteryDBus(sys))

	go func() {
		for sig := range c {
			if sig == nil || sig.Sender != "org.freedesktop.UPower" {
				continue
			}
			scheduleUpdate("BATTERY_ICON", getBatteryDBus(sys))
		}
	}()
}

func getBatteryDBus(sys *dbus.Conn) string {
	up := sys.Object("org.freedesktop.UPower", dbus.ObjectPath("/org/freedesktop/UPower"))
	var devPaths []dbus.ObjectPath
	if err := up.Call("org.freedesktop.UPower.EnumerateDevices", 0).Store(&devPaths); err != nil {
		return "battery-warning"
	}
	best := "battery-warning"
	for _, p := range devPaths {
		dev := sys.Object("org.freedesktop.UPower", p)
		var r map[string]dbus.Variant
		if err := dev.Call("org.freedesktop.DBus.Properties.GetAll", 0, "org.freedesktop.UPower.Device").Store(&r); err != nil {
			continue
		}
		if typVar, ok := r["Type"]; ok {
			if typ, ok := typVar.Value().(uint32); ok && typ == 2 {
				var state uint32
				var percent float64
				if sv, ok := r["State"]; ok {
					if v, ok := sv.Value().(uint32); ok {
						state = v
					}
				}
				if pv, ok := r["Percentage"]; ok {
					if pf, ok := pv.Value().(float64); ok {
						percent = pf
					}
				}
				if state == 1 || state == 4 {
					best = "battery-charging"
				} else {
					switch {
					case percent >= 75:
						best = "battery-full"
					case percent >= 50:
						best = "battery-medium"
					case percent >= 25:
						best = "battery-low"
					case percent >= 15:
						best = "battery"
					default:
						best = "battery-warning"
					}
				}
				return best
			}
		}
	}
	return best
}

// BlueZ - used for bluetooth status
func watchBlueZ(sys *dbus.Conn) {
	if sys == nil {
		fmt.Fprintln(os.Stderr, "watchBlueZ: no system bus connection")
		return
	}

	if call := sys.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',sender='org.bluez'"); call.Err != nil {
		fmt.Fprintln(os.Stderr, "watchBlueZ AddMatch error:", call.Err)
	}

	c := make(chan *dbus.Signal, 16)
	sys.Signal(c)

	scheduleUpdate("BLUETOOTH_ICON", getBluetoothDBus(sys))

	go func() {
		for sig := range c {
			if sig == nil || sig.Sender != "org.bluez" {
				continue
			}
			scheduleUpdate("BLUETOOTH_ICON", getBluetoothDBus(sys))
		}
	}()
}

func getBluetoothDBus(sys *dbus.Conn) string {
	om := sys.Object("org.bluez", dbus.ObjectPath("/"))
	var managed map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	if err := om.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&managed); err != nil {
		return "bluetooth-off"
	}
	for _, ifs := range managed {
		if props, ok := ifs["org.bluez.Adapter1"]; ok {
			if p, ok := props["Powered"]; ok {
				if powered, ok := p.Value().(bool); ok && powered {
					return "bluetooth"
				}
			}
		}
		if props, ok := ifs["org.bluez.Device1"]; ok {
			if p, ok := props["Connected"]; ok {
				if connected, ok := p.Value().(bool); ok && connected {
					return "bluetooth"
				}
			}
		}
	}
	return "bluetooth-off"
}

// Brightness
func watchBrightness() {
	basedirs, _ := filepath.Glob("/sys/class/backlight/*")
	if len(basedirs) == 0 {
		scheduleUpdate("BRIGHTNESS_ICON", "sun-dim")
		return
	}
	target := filepath.Join(basedirs[0], "brightness")
	maxPath := filepath.Join(basedirs[0], "max_brightness")

	updateFromFiles := func() {
		cur := readIntFile(target)
		max := readIntFile(maxPath)
		if max <= 0 {
			scheduleUpdate("BRIGHTNESS_ICON", "sun-dim")
			return
		}
		p := (cur * 100) / max
		switch {
		case p >= 75:
			scheduleUpdate("BRIGHTNESS_ICON", "sun")
		case p >= 40:
			scheduleUpdate("BRIGHTNESS_ICON", "sun-medium")
		default:
			scheduleUpdate("BRIGHTNESS_ICON", "sun-dim")
		}
	}

	updateFromFiles()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		go func() {
			for range time.Tick(10 * time.Second) {
				updateFromFiles()
			}
		}()
		return
	}
	if err = watcher.Add(target); err != nil {
		_ = watcher.Close()
		go func() {
			for range time.Tick(10 * time.Second) {
				updateFromFiles()
			}
		}()
		return
	}

	go func() {
		defer watcher.Close()
		for ev := range watcher.Events {
			if ev.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				updateFromFiles()
			}
		}
	}()
}

func readIntFile(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(b))
	v, _ := strconv.Atoi(s)
	return v
}

// Nightlight - uses hyprctl and hyprsunset;
// currenly, it does not work as the code assumes that hyprctl monitors output contains nightlight status, which it does not.
func watchNightlight() {
	go func() {
		update := func() {
			out, err := exec.Command("hyprctl", "monitors").Output()
			if err != nil {
				scheduleUpdate("NIGHTLIGHT_ICON", "eclipse")
				return
			}
			if bytes.Contains(out, []byte("nightlight: 1")) || bytes.Contains(out, []byte("night_light: 1")) {
				scheduleUpdate("NIGHTLIGHT_ICON", "moon")
			} else {
				scheduleUpdate("NIGHTLIGHT_ICON", "eclipse")
			}
		}
		update()
		for range time.Tick(30 * time.Second) {
			update()
		}
	}()
}

// Volume - fallbacks to wpctl, then pamixer
func watchVolume() {
	go func() {
		update := func() {
			out, err := exec.Command("wpctl", "get-volume", "@DEFAULT_SINK@").Output()
			if err != nil || len(out) == 0 {
				out2, err2 := exec.Command("pamixer", "--get-volume").Output()
				if err2 != nil {
					scheduleUpdate("VOLUME_ICON", "volume-x")
					return
				}
				vol, _ := strconv.Atoi(strings.TrimSpace(string(out2)))
				scheduleUpdate("VOLUME_ICON", volumeIconFromInt(vol))
				return
			}
			s := strings.TrimSpace(string(out))
			num := extractFirstInt(s)
			if num < 0 {
				scheduleUpdate("VOLUME_ICON", "volume-x")
				return
			}
			muteOut, err := exec.Command("wpctl", "get-mute", "@DEFAULT_SINK@").Output()
			if err == nil && strings.Contains(string(muteOut), "yes") {
				scheduleUpdate("VOLUME_ICON", "volume-x")
				return
			}
			scheduleUpdate("VOLUME_ICON", volumeIconFromInt(num))
		}

		update()
		for range time.Tick(volumePollInterval) {
			update()
		}
	}()
}

func volumeIconFromInt(vol int) string {
	switch {
	case vol >= 75:
		return "volume-2"
	case vol >= 50:
		return "volume-1"
	case vol >= 25:
		return "volume"
	case vol == 0:
		return "volume-x"
	default:
		return "volume-x"
	}
}

func extractFirstInt(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			j := i
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			n, _ := strconv.Atoi(s[i:j])
			return n
		}
	}
	return -1
}

// Notifications - uses swaync
func watchNotifications() {
	go func() {
		update := func() {
			if err := exec.Command("pgrep", "swaync").Run(); err != nil {
				scheduleUpdate("NOTIFICATION_ICON", "bell-off")
				return
			}
			if out, err := exec.Command("swaync", "dnd").Output(); err == nil {
				s := strings.TrimSpace(string(out))
				if s == "true" || s == "1" {
					scheduleUpdate("NOTIFICATION_ICON", "bell-minus")
					return
				}
			}
			out, err := exec.Command("swaync", "list").Output()
			if err != nil {
				scheduleUpdate("NOTIFICATION_ICON", "bell")
				return
			}
			if len(bytes.TrimSpace(out)) > 0 {
				scheduleUpdate("NOTIFICATION_ICON", "bell-dot")
			} else {
				scheduleUpdate("NOTIFICATION_ICON", "bell")
			}
		}
		update()
		for range time.Tick(notificationPollInterval) {
			update()
		}
	}()
}

// Mic - uses wpctl
func watchMic() {
	go func() {
		update := func() {
			out, err := exec.Command("wpctl", "get-volume", "@DEFAULT_SOURCE@").Output()
			if err != nil {
				scheduleUpdate("MIC_ICON", "mic-off")
				return
			}
			if strings.Contains(string(out), "Volume:") {
				scheduleUpdate("MIC_ICON", "mic-off")
			} else {
				scheduleUpdate("MIC_ICON", "mic")
			}
		}
		update()
		for range time.Tick(4 * time.Second) {
			update()
		}
	}()
}

func main() {
	sys, sysErr := dbus.ConnectSystemBus()
	if sysErr != nil {
		fmt.Fprintln(os.Stderr, "system bus connect error:", sysErr)
		sys = nil
	}

	// Start watchers only when their bus is available
	if sys != nil {
		watchNetwork(sys)
		watchUPower(sys)
		watchBlueZ(sys)
	}
	watchBrightness()
	watchNightlight()
	watchVolume()
	watchNotifications()
	watchMic()

	// initial push
	stateMu.Lock()
	inits := make([]string, 0, len(state))
	for k, v := range state {
		inits = append(inits, fmt.Sprintf("%s=%s", k, escapeArg(v)))
	}
	stateMu.Unlock()
	runEwwUpdates(inits)

	select {}
}
