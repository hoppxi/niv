package operation

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// audio represents the audio subsystem.
type audio struct{}

// Audio is the exported instance.
var Audio audio

// --- Public API ---

func (a *audio) SetOutputLevel(lvl int) error {
	return setDeviceVolume(true, lvl)
}

func (a *audio) MuteOutput() error {
	return setDeviceMute(true, true)
}

func (a *audio) UnmuteOutput() error {
	return setDeviceMute(true, false)
}

func (a *audio) ToggleMuteOutput() error {
	return toggleDeviceMute(true)
}

func (a *audio) SetInputLevel(lvl int) error {
	return setDeviceVolume(false, lvl)
}

func (a *audio) MuteInput() error {
	return setDeviceMute(false, true)
}

func (a *audio) UnmuteInput() error {
	return setDeviceMute(false, false)
}

func (a *audio) ToggleMuteInput() error {
	return toggleDeviceMute(false)
}



// --- Internal Logic (pactl) ---

func setDeviceVolume(isOutput bool, lvl int) error {
	if lvl < 0 {
		lvl = 0
	}
	if lvl > 100 {
		lvl = 100
	}

	deviceType := "sink"
	if !isOutput {
		deviceType = "source"
	}

	device, err := getDefaultDevice(deviceType)
	if err != nil {
		return err
	}

	volArg := strconv.Itoa(lvl) + "%"
	cmd := exec.Command("pactl", "set-"+deviceType+"-volume", device, volArg)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set volume: %v", err)
	}

	// If volume > 0, unmute
	if lvl > 0 {
		_ = setDeviceMute(isOutput, false)
	}

	return nil
}

func setDeviceMute(isOutput bool, mute bool) error {
	deviceType := "sink"
	if !isOutput {
		deviceType = "source"
	}

	device, err := getDefaultDevice(deviceType)
	if err != nil {
		return err
	}

	muteArg := "0"
	if mute {
		muteArg = "1"
	}

	cmd := exec.Command("pactl", "set-"+deviceType+"-mute", device, muteArg)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set mute: %v", err)
	}
	return nil
}

func toggleDeviceMute(isOutput bool) error {
	deviceType := "sink"
	if !isOutput {
		deviceType = "source"
	}

	device, err := getDefaultDevice(deviceType)
	if err != nil {
		return err
	}

	cmd := exec.Command("pactl", "get-"+deviceType+"-mute", device)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get mute state: %v", err)
	}

	state := strings.Contains(string(out), "yes")
	var newMute string
	if state {
		newMute = "0"
	} else {
		newMute = "1"
	}

	cmd = exec.Command("pactl", "set-"+deviceType+"-mute", device, newMute)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to toggle mute: %v", err)
	}

	return nil
}

func getDefaultDevice(deviceType string) (string, error) {
	cmd := exec.Command("pactl", "get-default-"+deviceType)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default %s: %v", deviceType, err)
	}
	return strings.TrimSpace(string(out)), nil
}
