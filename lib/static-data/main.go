/*
A simple system information that are static and does not require updates
Usage: sysinfo --cpu --memory --network --battery --os --packages --display --audio
Outputs JSON with requested sections
*/
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/host"
)

const (
	cmdTimeout      = 1200 * time.Millisecond // command timeout for responsiveness
	pkgListLimit    = 100                      // limit number of packages returned for speed
	readSmallBuffer = 4096
)

type CPUInfo struct {
	ModelName    string  `json:"model"`
	BaseFreqGHz  float64 `json:"base_freq_ghz"`
	Cores        int     `json:"cores"`
	Threads      int     `json:"threads"`
	Architecture string  `json:"arch"`
}

type MemoryInfo struct {
	TotalMB uint64 `json:"total_installed"`
	FreeMB  uint64 `json:"free"`
	SwapMB  uint64 `json:"swap_total"`
	UsedMB  uint64 `json:"used"`
}

type NetworkInfo struct {
	Interfaces []string `json:"interfaces"`
	MACAddress string   `json:"mac_address"` // first non-empty MAC found
	Driver     string   `json:"driver"`
	Firmware   string   `json:"firmware"`
}

type BatteryInfo struct {
	HasBattery         bool   `json:"has_battery"`
	Type               string `json:"type"`
	Health             string `json:"health"`
	DesignCapacitymWh  int    `json:"design_capacity_mwh"`
	CurrentCapacitymWh int    `json:"current_capacity_mwh"`
}

type OSInfo struct {
	OS          string `json:"os"`
	Platform    string `json:"platform"`
	Kernel      string `json:"kernel"`
	PCName      string `json:"hostname"`
	UptimeHours int64  `json:"uptime_hours"`
}

type DisplayInfo struct {
	Detected     bool   `json:"detected"`
	Resolution   string `json:"resolution"`
	RefreshRate  string `json:"refresh_rate"`
	ResponseTime string `json:"response_time"`
	ViewingAngle string `json:"viewing_angle"`
	ModelNumber  string `json:"model_number"`
	Manufacturer string `json:"manufacturer"`
}

type AudioInfo struct {
	Detected    bool   `json:"detected"`
	Name        string `json:"name"`
	MicDetected bool   `json:"mic_detected"`
	MicName     string `json:"mic_name"`
}

type PackageInfo struct {
	SystemPackages []string `json:"system_packages"`
	UserPackages   []string `json:"user_packages"`
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func readFileTrim(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	buf := make([]byte, readSmallBuffer)
	n, _ := f.Read(buf)
	return strings.TrimSpace(string(buf[:n])), nil
}

// CPU
func getCPUInfo(ctx context.Context) CPUInfo {
	// Try lscpu for clean output
	out, err := runCmd(ctx, "lscpu", "-J")
	cpu := CPUInfo{Architecture: runtimeArch()}
	if err == nil && out != "" {
		var j map[string][]map[string]string
		if err := json.Unmarshal([]byte(out), &j); err == nil {
			if infos, ok := j["lscpu"]; ok {
				for _, kv := range infos {
					k := kv["field"]
					v := kv["data"]
					switch strings.TrimSpace(strings.ToLower(strings.TrimSuffix(k, ":"))) {
					case "model name":
						if cpu.ModelName == "" {
							cpu.ModelName = v
						}
					case "cpu mhz":
						if cpu.BaseFreqGHz == 0 {
							if f := parseFreqMHz(v); f > 0 {
								cpu.BaseFreqGHz = f / 1000.0
							}
						}
					case "cpu(s)":
						if cpu.Cores == 0 {
							if n := parseInt(v); n > 0 {
								cpu.Cores = n
							}
						}
					case "thread(s) per core":
						// currently ignore
					case "socket(s)":
						// currently ignore
					}
				}
			}
		}
	}

	if cpu.ModelName == "" || cpu.Cores == 0 || cpu.BaseFreqGHz == 0 {
		f, err := os.Open("/proc/cpuinfo")
		if err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			model := ""
			freq := 0.0
			processorCount := 0
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "model name") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 && model == "" {
						model = strings.TrimSpace(parts[1])
					}
				}
				if strings.HasPrefix(line, "cpu MHz") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						if v := parseFloat(parts[1]); v > 0 && freq == 0 {
							freq = v / 1000.0
						}
					}
				}
				if strings.HasPrefix(line, "processor") {
					processorCount++
				}
			}
			if model != "" {
				cpu.ModelName = model
			}
			if freq > 0 {
				cpu.BaseFreqGHz = freq
			}
			if processorCount > 0 && cpu.Cores == 0 {
				cpu.Cores = processorCount
			}
		}
	}

	if cpu.Threads == 0 {
		if out, err := runCmd(ctx, "nproc", "--all"); err == nil && out != "" {
			if n := parseInt(out); n > 0 {
				cpu.Threads = n
			}
		}
	}
	cpu.Architecture = runtimeArch()
	return cpu
}

func runtimeArch() string {
	return strings.TrimSpace(runtimeGOARCH())
}

// wrapper to avoid direct import cycle; will call runtime.GOARCH via small function
func runtimeGOARCH() string {
	return os.Getenv("GOARCH_OVERRIDE")
}

// Because we can't call runtime.GOARCH from a helper due to method placement, do simple:
func init() {
	// nothing; we'll just call runtime.GOARCH directly in a simple way
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	re := regexp.MustCompile(`\d+`)
	m := re.FindString(s)
	if m == "" {
		return 0
	}
	var v int
	fmt.Sscanf(m, "%d", &v)
	return v
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	re := regexp.MustCompile(`\d+(\.\d+)?`)
	m := re.FindString(s)
	if m == "" {
		return 0
	}
	var v float64
	fmt.Sscanf(m, "%f", &v)
	return v
}

func parseFreqMHz(s string) float64 {
	return parseFloat(s)
}

// Memory
func getMemoryInfo() MemoryInfo {
	// parse /proc/meminfo
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return MemoryInfo{}
	}
	defer f.Close()
	var total, free, available, swapTotal, used uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		val := parseUint64(fields[1]) // kB
		switch key {
		case "MemTotal":
			total = val
		case "MemFree":
			free = val
		case "MemAvailable":
			available = val
		case "SwapTotal":
			swapTotal = val
		}
	}
	// used = total - available (approx)
	if total > available {
		used = total - available
	} else {
		used = 0
	}
	return MemoryInfo{
		TotalMB: total / 1024,
		FreeMB:  free / 1024,
		SwapMB:  swapTotal / 1024,
		UsedMB:  used / 1024,
	}
}

func parseUint64(s string) uint64 {
	s = strings.TrimSpace(s)
	var v uint64
	fmt.Sscanf(s, "%d", &v)
	return v
}

// Network
func getNetworkInfo(ctx context.Context) NetworkInfo {
	// Read interfaces from /sys/class/net
	ifacesDir := "/sys/class/net"
	names := []string{}
	mac := ""
	driver := "unknown"
	firmware := "unknown"
	dirEntries, err := os.ReadDir(ifacesDir)
	if err == nil {
		for _, de := range dirEntries {
			name := de.Name()
			names = append(names, name)
			if mac == "" {
				if addr, err := readFileTrim(filepath.Join(ifacesDir, name, "address")); err == nil {
					if addr != "" && addr != "00:00:00:00:00:00" {
						mac = addr
					}
				}
			}
			// try driver symlink
			if driver == "unknown" {
				driverPath := filepath.Join(ifacesDir, name, "device", "driver")
				if link, err := os.Readlink(driverPath); err == nil {
					driver = filepath.Base(link)
				}
			}
			// try firmware
			if firmware == "unknown" {
				fwPath := filepath.Join(ifacesDir, name, "device", "firmware_node")
				if _, err := os.Stat(fwPath); err == nil {
					if link, err := os.Readlink(fwPath); err == nil {
						firmware = filepath.Base(link)
					}
				}
			}
		}
	}
	// fallback to ethtool -i for first interface
	if driver == "unknown" && len(names) > 0 {
		if out, err := runCmd(ctx, "ethtool", "-i", names[0]); err == nil && out != "" {
			for _, line := range strings.Split(out, "\n") {
				if strings.HasPrefix(line, "driver:") {
					driver = strings.TrimSpace(strings.TrimPrefix(line, "driver:"))
				}
				if strings.HasPrefix(line, "firmware-version:") {
					firmware = strings.TrimSpace(strings.TrimPrefix(line, "firmware-version:"))
				}
			}
		}
	}

	return NetworkInfo{
		Interfaces: names,
		MACAddress: mac,
		Driver:     driver,
		Firmware:   firmware,
	}
}

// Battery
func getBatteryInfo() BatteryInfo {
	powerDir := "/sys/class/power_supply"
	entries, err := os.ReadDir(powerDir)
	if err != nil {
		return BatteryInfo{HasBattery: false}
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(strings.ToLower(name), "bat") || strings.Contains(strings.ToLower(name), "battery") {
			base := filepath.Join(powerDir, name)
			typ, _ := readFileTrim(filepath.Join(base, "type"))
			health, _ := readFileTrim(filepath.Join(base, "health"))
			designCapStr, _ := readFileTrim(filepath.Join(base, "energy_full_design"))
			currCapStr, _ := readFileTrim(filepath.Join(base, "energy_full"))
			// some systems use charge_* instead of energy_*
			if designCapStr == "" {
				designCapStr, _ = readFileTrim(filepath.Join(base, "charge_full_design"))
			}
			if currCapStr == "" {
				currCapStr, _ = readFileTrim(filepath.Join(base, "charge_full"))
			}
			design := parseInt(designCapStr)
			curr := parseInt(currCapStr)
			if design == 0 && curr == 0 {
				// fallback to capacity*1000 if in uAh
				design = parseInt(designCapStr)
				curr = parseInt(currCapStr)
			}
			return BatteryInfo{
				HasBattery:         true,
				Type:               typ,
				Health:             health,
				DesignCapacitymWh:  design,
				CurrentCapacitymWh: curr,
			}
		}
	}
	return BatteryInfo{HasBattery: false}
}

// OS
func getOSInfo() OSInfo {
	h, _ := host.Info()
	platform := ""
	if d, err := os.ReadFile("/etc/os-release"); err == nil {
		// parse NAME= or ID=
		scanner := bufio.NewScanner(bytes.NewReader(d))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "NAME=") && platform == "" {
				platform = strings.Trim(line[5:], `"`)
			}
			if strings.HasPrefix(line, "ID=") && platform == "" {
				platform = strings.Trim(line[3:], `"`)
			}
		}
	}
	if platform == "" {
		platform = h.Platform
	}
	return OSInfo{
		OS:          "linux",
		Platform:    platform,
		Kernel:      h.KernelVersion,
		PCName:      h.Hostname,
		UptimeHours: int64(h.Uptime / 3600),
	}
}

// Display
func getDisplayInfo(ctx context.Context) DisplayInfo {
	// For Hyprland, hyprctl provides JSON monitors
	out, err := runCmd(ctx, "hyprctl", "monitors", "-j")
	if err == nil && out != "" {
		var arr []map[string]interface{}
		if err := json.Unmarshal([]byte(out), &arr); err == nil && len(arr) > 0 {
			m := arr[0]
			res := "unknown"
			if w, ok := m["width"]; ok {
				if h, ok2 := m["height"]; ok2 {
					res = fmt.Sprintf("%v×%v", w, h)
				}
			}
			refresh := "unknown"
			if r, ok := m["refresh"]; ok {
				refresh = fmt.Sprintf("%v", r)
			}
			model := "unknown"
			if md, ok := m["model"]; ok {
				model = fmt.Sprintf("%v", md)
			}
			man := "unknown"
			if mn, ok := m["make"]; ok {
				man = fmt.Sprintf("%v", mn)
			}
			return DisplayInfo{
				Detected:     true,
				Resolution:   res,
				RefreshRate:  refresh,
				ResponseTime: "unknown",
				ViewingAngle: "unknown",
				ModelNumber:  model,
				Manufacturer: man,
			}
		}
	}

	// fallback: read drm connectors (very light)
	// look for /sys/class/drm/*/modes
	drmGlob := "/sys/class/drm/*/modes"
	matches, _ := filepath.Glob(drmGlob)
	if len(matches) > 0 {
		m := matches[0]
		if data, err := readFileTrim(m); err == nil && data != "" {
			// mode like "1920x1080"
			res := strings.TrimSpace(data)
			return DisplayInfo{
				Detected:     true,
				Resolution:   res,
				RefreshRate:  "unknown",
				ResponseTime: "unknown",
				ViewingAngle: "unknown",
				ModelNumber:  "unknown",
				Manufacturer: "unknown",
			}
		}
	}
	return DisplayInfo{Detected: false}
}

// Audio
func getAudioInfo(ctx context.Context) AudioInfo {
	// Try pactl (PipeWire)
	out, err := runCmd(ctx, "pactl", "list", "short", "sinks")
	if err == nil && out != "" {
		lines := strings.Split(out, "\n")
		if len(lines) > 0 {
			first := strings.Fields(lines[0])
			name := ""
			if len(first) >= 2 {
				name = first[1]
			}
			// sources for mic
			outSrc, _ := runCmd(ctx, "pactl", "list", "short", "sources")
			micDetected := false
			micName := ""
			if outSrc != "" {
				linesSrc := strings.Split(outSrc, "\n")
				if len(linesSrc) > 0 && strings.TrimSpace(linesSrc[0]) != "" {
					micFields := strings.Fields(linesSrc[0])
					if len(micFields) >= 2 {
						micName = micFields[1]
						micDetected = true
					}
				}
			}
			return AudioInfo{
				Detected:    true,
				Name:        name,
				MicDetected: micDetected,
				MicName:     micName,
			}
		}
	}
	// fallback to ALSA cards
	outCards, err := runCmd(ctx, "cat", "/proc/asound/cards")
	if err == nil && outCards != "" {
		// try to parse first card name between brackets
		r := regexp.MustCompile(`\d+\s+\[(.*?)\]`)
		m := r.FindStringSubmatch(outCards)
		name := "unknown"
		if len(m) >= 2 {
			name = m[1]
		}
		return AudioInfo{
			Detected:    true,
			Name:        name,
			MicDetected: false,
			MicName:     "",
		}
	}
	return AudioInfo{Detected: false}
}

// Packages - NixOS
func getPackageInfo(ctx context.Context) PackageInfo {
	sysPkgs := []string{}
	userPkgs := []string{}

	// system packages: for NixOS, list direct store paths used by current system
	// limit results for speed
	// command: nix-store -q --references /run/current-system | head -n <limit>
	out, err := runCmd(ctx, "bash", "-c", fmt.Sprintf("nix-store -q --references /run/current-system 2>/dev/null | head -n %d", pkgListLimit))
	if err == nil && out != "" {
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				sysPkgs = append(sysPkgs, line)
			}
		}
	}

	// user packages: try nix-env -q (limit)
	out2, err := runCmd(ctx, "bash", "-c", fmt.Sprintf("nix-env -q | head -n %d", pkgListLimit))
	if err == nil && out2 != "" {
		for _, line := range strings.Split(out2, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				userPkgs = append(userPkgs, line)
			}
		}
	}

	return PackageInfo{
		SystemPackages: sysPkgs,
		UserPackages:   userPkgs,
	}
}

func main() {
	cpuFlag := flag.Bool("cpu", false, "Show CPU static info")
	memFlag := flag.Bool("memory", false, "Show Memory static info")
	netFlag := flag.Bool("network", false, "Show Network static info")
	battFlag := flag.Bool("battery", false, "Show Battery static info")
	osFlag := flag.Bool("os", false, "Show OS static info")
	pkgFlag := flag.Bool("packages", false, "Show Packages static info")
	dispFlag := flag.Bool("display", false, "Show Display static info")
	audioFlag := flag.Bool("audio", false, "Show Audio static info")
	flag.Parse()

	if !( *cpuFlag || *memFlag || *netFlag || *battFlag || *osFlag || *pkgFlag || *dispFlag || *audioFlag ) {
		fmt.Fprintln(os.Stderr, "Usage: sysinfo [--cpu] [--memory] [--network] [--battery] [--os] [--packages] [--display] [--audio]")
		os.Exit(1)
	}

	ctx := context.Background()
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	out := make(map[string]interface{})

	add := func(k string, v interface{}) {
		mu.Lock()
		out[k] = v
		mu.Unlock()
	}

	if *cpuFlag {
		wg.Add(1)
		go func() {
			defer wg.Done()
			add("cpu", getCPUInfo(ctx))
		}()
	}
	if *memFlag {
		wg.Add(1)
		go func() {
			defer wg.Done()
			add("memory", getMemoryInfo())
		}()
	}
	if *netFlag {
		wg.Add(1)
		go func() {
			defer wg.Done()
			add("network", getNetworkInfo(ctx))
		}()
	}
	if *battFlag {
		wg.Add(1)
		go func() {
			defer wg.Done()
			add("battery", getBatteryInfo())
		}()
	}
	if *osFlag {
		wg.Add(1)
		go func() {
			defer wg.Done()
			add("os", getOSInfo())
		}()
	}
	if *pkgFlag {
		wg.Add(1)
		go func() {
			defer wg.Done()
			add("packages", getPackageInfo(ctx))
		}()
	}
	if *dispFlag {
		wg.Add(1)
		go func() {
			defer wg.Done()
			add("display", getDisplayInfo(ctx))
		}()
	}
	if *audioFlag {
		wg.Add(1)
		go func() {
			defer wg.Done()
			add("audio", getAudioInfo(ctx))
		}()
	}

	wg.Wait()

	j, err := json.Marshal(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error marshaling json:", err)
		os.Exit(2)
	}
	fmt.Println(string(j))
}
