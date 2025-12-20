package audioinfo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type AudioDevice struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
	Muted bool   `json:"muted"`
}

type AudioInfo struct {
	Output AudioDevice `json:"output"`
	Input  AudioDevice `json:"input"`
}

func parseVolume(line string) int {
	re := regexp.MustCompile(`(\d+)%`)
	matches := re.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return 100
	}
	var sum int
	for _, m := range matches {
		v, _ := strconv.Atoi(m[1])
		sum += v
	}
	return sum / len(matches)
}

func parseMuted(line string) bool {
	return strings.Contains(strings.ToLower(line), "yes")
}

func getDefaultDeviceName(kind string) (string, error) {
	cmd := exec.Command("pactl", "info")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	prefix := ""
	if kind == "sink" {
		prefix = "Default Sink: "
	} else {
		prefix = "Default Source: "
	}
	for _, line := range lines {
		if after, found := strings.CutPrefix(line, prefix); found {
			return after, nil
		}
	}
	return "", fmt.Errorf("default %s not found", kind)
}

func getDeviceFromPactl(kind string) (AudioDevice, error) {
	name, err := getDefaultDeviceName(kind)
	if err != nil {
		return AudioDevice{}, err
	}

	cmd := exec.Command("pactl", "list", kind+"s")
	out, err := cmd.Output()
	if err != nil {
		return AudioDevice{}, err
	}

	devices := bytes.SplitSeq(out, []byte("\n\n"))
	for block := range devices {
		if !bytes.Contains(block, []byte(name)) {
			continue
		}
		lines := bytes.Split(block, []byte("\n"))
		dev := AudioDevice{Name: name}
		for _, l := range lines {
			line := strings.TrimSpace(string(l))
			if strings.HasPrefix(line, "Mute:") {
				dev.Muted = parseMuted(line)
			}
			if strings.HasPrefix(line, "Volume:") {
				dev.Level = parseVolume(line)
			}
		}
		return dev, nil
	}

	// fallback
	return AudioDevice{Name: name, Level: 100, Muted: false}, nil
}

func GetAudioInfo() (*AudioInfo, error) {
	out, err := getDeviceFromPactl("sink")
	if err != nil {
		return nil, fmt.Errorf("failed to get output: %v", err)
	}

	in, err := getDeviceFromPactl("source")
	if err != nil {
		return nil, fmt.Errorf("failed to get input: %v", err)
	}

	info := &AudioInfo{}
	info.Output = out
	info.Input = in
	return info, nil
}

func GetAudioInfoJSON() ([]byte, error) {
	info, err := GetAudioInfo()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(info, "", "  ")
}
