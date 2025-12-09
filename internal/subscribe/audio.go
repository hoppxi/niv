package subscribe

import (
	"bufio"
	"log"
	"strings"

	"github.com/hoppxi/niv/internal/manager"
)

func AudioEvents() <-chan AudioEvent {
  out := make(chan AudioEvent, 16)

  // 1. CREATE the command (DOES NOT START YET)
  cmd, cancel := manager.NewCmd("pactl", "subscribe")
  
  // 2. SETUP the pipe BEFORE starting the process
  stdout, err := cmd.StdoutPipe()
  if err != nil {
    log.Printf("subscribe: failed to get stdout pipe: %v", err)
    cancel() // Clean up context if pipe fails
    close(out)
    return out
  }

  // 3. START the command and track it in the manager
  if manager.StartTrackedCmd(cmd, cancel) == nil {
      close(out)
      return out
  }
  
  // 4. Start the scanner routine
  go func() {
    defer close(out)
    // The cmd.Wait() is handled by manager.StopAll()
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
    // Scanner will block until stdout pipe is closed (when pactl process is killed)
  }()

  return out
}
// isAudioChange filters events we care about
func isAudioChange(line string) bool {
	line = strings.ToLower(line)
	return strings.Contains(line, "sink") ||
		strings.Contains(line, "source") ||
		strings.Contains(line, "server") ||
		strings.Contains(line, "card")
}
