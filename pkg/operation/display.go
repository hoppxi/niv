package operation

import (
	"fmt"
	"os/exec"
)

type display struct{}

// Display is the exported instance.
var Display display

// SetBrightness sets the screen brightness level (0-100).
func (d *display) SetBrightness(brightness string) error {
	cmd := exec.Command("brightnessctl", "set", brightness)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}
	return nil
}