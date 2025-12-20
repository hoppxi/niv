package batteryinfo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type BatteryInfo struct {
	Voltage         string `json:"voltage"`
	DesignCapacity  string `json:"design_capacity"`
	CurrentCapacity string `json:"current_capacity"`
}

type BatteryDynamicInfo struct {
	Level         int64  `json:"level"`
	Status        string `json:"status"`
	TimeRemaining string `json:"time_remaining"`
	PowerMode     string `json:"power_mode"`
	Source        string `json:"source"`
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func getBatteryPath() (string, error) {
	matches, _ := filepath.Glob("/sys/class/power_supply/BAT*")
	if len(matches) == 0 {
		return "", errors.New("no battery found")
	}
	return matches[0], nil
}

func formatCapacity(basePath string, fileName string) string {
	raw := readFile(filepath.Join(basePath, fileName))
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil || val == 0 {
		return "Unknown"
	}

	// If the filename contains "energy", unit is microWatt-hours (uWh) -> convert to Wh
	if strings.Contains(fileName, "energy") {
		return fmt.Sprintf("%.2f Wh", val/1000000.0)
	}
	// If the filename contains "charge", unit is microAmp-hours (uAh) -> convert to mAh
	if strings.Contains(fileName, "charge") {
		return fmt.Sprintf("%d mAh", int(val/1000.0))
	}
	return raw
}

func GetBatteryInfo() (*BatteryInfo, error) {
	base, err := getBatteryPath()
	if err != nil {
		return nil, err
	}

	// 1. Format Voltage (microvolts to Volts)
	vRaw := readFile(filepath.Join(base, "voltage_now"))
	vVal, _ := strconv.ParseFloat(vRaw, 64)
	voltageStr := fmt.Sprintf("%.2fV", vVal/1000000.0)

	// 2. Handle Design/Current Capacity (Check Energy then Charge)
	design := formatCapacity(base, "energy_full_design")
	if design == "Unknown" {
		design = formatCapacity(base, "charge_full_design")
	}

	current := formatCapacity(base, "energy_full")
	if current == "Unknown" {
		current = formatCapacity(base, "charge_full")
	}

	// 3. Detect Source (AC vs Battery)

	return &BatteryInfo{
		Voltage:         voltageStr,
		DesignCapacity:  design,
		CurrentCapacity: current,
	}, nil
}

func GetBatteryDynamicInfo() (*BatteryDynamicInfo, error) {
	base, err := getBatteryPath()
	if err != nil {
		return nil, err
	}

	// 1. Level and Status
	status := readFile(filepath.Join(base, "status"))
	capStr := readFile(filepath.Join(base, "capacity"))
	level, _ := strconv.ParseInt(capStr, 10, 64)

	// 2. Power Mode (Reads from Linux Power Profiles Daemon)
	// Typical values: 'performance', 'balanced', 'power-saver'
	pMode := readFile("/sys/firmware/acpi/platform_profile")
	if pMode == "" {
		pMode = "balanced" // Fallback
	}

	// 3. Time Remaining Calculation
	timeRem := "Calculating..."
	if status == "Discharging" {
		// Use Energy (mWh) / Power (mW)
		eNow, _ := strconv.ParseFloat(readFile(filepath.Join(base, "energy_now")), 64)
		pNow, _ := strconv.ParseFloat(readFile(filepath.Join(base, "power_now")), 64)

		// Fallback to Charge (mAh) / Current (mA)
		if eNow == 0 {
			eNow, _ = strconv.ParseFloat(readFile(filepath.Join(base, "charge_now")), 64)
			pNow, _ = strconv.ParseFloat(readFile(filepath.Join(base, "current_now")), 64)
		}

		if pNow > 0 {
			hours := eNow / pNow
			mins := int((hours - float64(int(hours))) * 60)
			timeRem = fmt.Sprintf("%dh %dm", int(hours), mins)
		}
	} else if status == "Full" || status == "Charging" {
		timeRem = "N/A"
	}
	source := "Battery"
	if readFile("/sys/class/power_supply/AC/online") == "1" ||
		readFile("/sys/class/power_supply/ACAD/online") == "1" {
		source = "AC"
	}

	return &BatteryDynamicInfo{
		Level:         level,
		Status:        status,
		TimeRemaining: timeRem,
		PowerMode:     pMode,
		Source:        source,
	}, nil
}

func GetBatteryInfoJSON() ([]byte, error) {
	info, err := GetBatteryInfo()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(info, "", "  ")
}

func GetBatteryDynamicInfoJSON() ([]byte, error) {
	info, err := GetBatteryDynamicInfo()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(info, "", "  ")
}
