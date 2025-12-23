package miscinfo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"
)

// MiscInfo struct for JSON
type MiscInfo struct {
	Name   string `json:"name"`
	OS     string `json:"os"`
	OSIcon string `json:"osicon"`
	Uptime string `json:"uptime"`
}

// GetMisc gathers misc information
func GetMisc() MiscInfo {
	var info MiscInfo

	// username@hostname
	currentUser, err := user.Current()
	username := "unknown"
	if err == nil {
		username = currentUser.Username
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	info.Name = fmt.Sprintf("%s@%s", username, hostname)

	// OS info
	osName, osIcon := getLinuxDistro()
	info.OS = osName
	info.OSIcon = osIcon

	// Uptime
	info.Uptime = GetUptime()

	return info
}

func GetMiscJSON() ([]byte, error) {
	info := GetMisc()
	return json.MarshalIndent(info, "", "  ")
}

func getLinuxDistro() (name, icon string) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "Linux", "linux"
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var prettyName, id string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			prettyName = strings.Trim(line[len("PRETTY_NAME="):], `"`)
		}
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(line[len("ID="):], `"`)
		}
	}

	if prettyName == "" {
		prettyName = "Linux"
	}
	if id == "" {
		id = "linux"
	}

	return prettyName, "distributor-logo-" + id
}

func GetUptime() string {
	content, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "unknown"
	}

	var uptimeSeconds float64
	fmt.Sscanf(string(content), "%f", &uptimeSeconds)
	d := time.Duration(uptimeSeconds) * time.Second

	h := int(d.Hours())
	m := int(d.Minutes()) % 60

	if h > 0 {
		if m > 0 {
			return fmt.Sprintf("%d hour%s %d minute%s", h, plural(h), m, plural(m))
		}
		return fmt.Sprintf("%d hour%s", h, plural(h))
	}
	return fmt.Sprintf("%d minute%s", m, plural(m))
}

func plural(n int) string {
	if n != 1 {
		return "s"
	}
	return ""
}
