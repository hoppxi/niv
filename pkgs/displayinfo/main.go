package displayinfo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type DisplayInfo struct {
	Level int `json:"level"`
}

func readInt(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(data))
	return strconv.Atoi(s)
}

func GetDisplayInfo() (*DisplayInfo, error) {
	paths, err := filepath.Glob("/sys/class/backlight/*")
	if err != nil || len(paths) == 0 {
		return nil, errors.New("no backlight devices found")
	}

	device := paths[0] // You might need to select a different device
	current, err := readInt(filepath.Join(device, "brightness"))
	if err != nil {
		return nil, err
	}

	maxVal, err := readInt(filepath.Join(device, "max_brightness"))
	if err != nil {
		return nil, err
	}

	if maxVal <= 0 {
		return nil, errors.New("invalid max_brightness value")
	}

	percent := int(float64(current) / float64(maxVal) * 100.0)
	if percent < 0 {
		percent = 0
	} else if percent > 100 {
		percent = 100
	}

	return &DisplayInfo{
		Level: percent,
	}, nil
}

func GetDisplayInfoJSON() ([]byte, error) {
	info, err := GetDisplayInfo()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(info, "", "  ")
}
