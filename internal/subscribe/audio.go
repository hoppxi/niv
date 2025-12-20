package subscribe

import (
	"bufio"
	"log"
	"strings"

	"github.com/hoppxi/wigo/internal/manager"
)

func AudioEvents() <-chan AudioEvent {
	out := make(chan AudioEvent, 16)
	cmd, cancel := manager.NewCmd("pactl", "subscribe")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("subscribe: failed to get stdout pipe: %v", err)
		cancel() // Clean up context if pipe fails
		close(out)
		return out
	}

	if manager.StartTrackedCmd(cmd, cancel) == nil {
		close(out)
		return out
	}

	go func() {
		defer close(out)
		scanner := bufio.NewScanner(stdout)

		for scanner.Scan() {
			line := scanner.Text()
			if isAudioChange(line) {
				select {
				case out <- AudioEvent{}:
				default:
				}
			}
		}
	}()

	return out
}

func isAudioChange(line string) bool {
	line = strings.ToLower(line)
	return strings.Contains(line, "sink") ||
		strings.Contains(line, "source") ||
		strings.Contains(line, "server") ||
		strings.Contains(line, "card")
}
