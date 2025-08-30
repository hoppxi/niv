package main

import (
    "bufio"
    "bytes"
    "fmt"
    "os"
    "os/exec"
    "strconv"
    "strings"
    "time"
)

func runEwwUpdate(varName, value string) {
    cmd := exec.Command("bash", "-c", fmt.Sprintf("eww update %s=%s", varName, value))
    if err := cmd.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "eww update failed for %s=%s: %v\n", varName, value, err)
    }
}
func getNetworkStatus() string {
    out, err := exec.Command("nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "device").Output()
    if err != nil {
        return "wifi-off"
    }

    scanner := bufio.NewScanner(bytes.NewReader(out))
    var wifiDevice string
    var ethernetConnected bool
    for scanner.Scan() {
        line := scanner.Text()
        parts := strings.Split(line, ":")
        if len(parts) < 3 {
            continue
        }
        device, typ, state := parts[0], parts[1], parts[2]

        if typ == "ethernet" && state == "connected" {
            ethernetConnected = true
        }
        if typ == "wifi" && state == "connected" {
            wifiDevice = device
        }
    }

    if ethernetConnected {
        return "ethernet-port"
    }

    if wifiDevice == "" {
        return "wifi-off"
    }

    outWifi, err := exec.Command("nmcli", "-f", "IN-USE,SIGNAL", "device", "wifi").Output()
    if err != nil {
        return "wifi-off"
    }

    scanner = bufio.NewScanner(bytes.NewReader(outWifi))
    for scanner.Scan() {
        line := scanner.Text()
        fields := strings.Fields(line)
        if len(fields) < 2 {
            continue
        }
        inUse, signalStr := fields[0], fields[1]

        if inUse == "*" {
            sig, err := strconv.Atoi(signalStr)
            if err != nil {
                return "wifi-off"
            }
            switch {
            case sig > 80:
                return "wifi"
            case sig > 60:
                return "wifi-high"
            case sig > 40:
                return "wifi-low"
            case sig > 20:
                return "wifi-zero"
            default:
                return "wifi"
            }
        }
    }

    return "wifi-off"
}

// Volume status using pamixer
func getVolumeStatus() string {
    mutedOut, err := exec.Command("pamixer", "--get-mute").Output()
    if err != nil {
        return "volume-x"
    }
    muted := strings.TrimSpace(string(mutedOut)) == "true"
    if muted {
        return "volume-x"
    }

    volOut, err := exec.Command("pamixer", "--get-volume").Output()
    if err != nil {
        return "volume-x"
    }
    volStr := strings.TrimSpace(string(volOut))
    vol, err := strconv.Atoi(volStr)
    if err != nil {
        return "volume-x"
    }

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

// Battery status using upower
func getBatteryStatus() string {
    out, err := exec.Command("upower", "-i", "/org/freedesktop/UPower/devices/battery_BAT0").Output()
    if err != nil {
        return "battery-warning"
    }

    scanner := bufio.NewScanner(bytes.NewReader(out))
    var state string
    var capacity int
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if strings.HasPrefix(line, "state:") {
            state = strings.TrimSpace(strings.TrimPrefix(line, "state:"))
        }
        if strings.HasPrefix(line, "percentage:") {
            percentStr := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "percentage:"), "%"))
            capacity, _ = strconv.Atoi(percentStr)
        }
    }

    if state == "charging" || state == "fully-charged" {
        return "battery-charging"
    }

    if capacity >= 75 {
        return "battery-full"
    } else if capacity >= 50 {
        return "battery-medium"
    } else if capacity >= 25 {
        return "battery-low"
    } else if capacity >= 15 {
        return "battery"
    } 
    return "battery-warning"
}

func getNotificationStatus() string {
    return "bell-off"
}

func main() {

    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        network := getNetworkStatus()
        volume := getVolumeStatus()
        battery := getBatteryStatus()
        notification := getNotificationStatus()

        runEwwUpdate("NETWORK_ICON", network)
        runEwwUpdate("VOLUME_ICON", volume)
        runEwwUpdate("BATTERY_ICON", battery)
        runEwwUpdate("NOTIFICATION_ICON", notification)

        <-ticker.C
    }
}
