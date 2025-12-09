package subscribe

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func runtimeDir() string {
	return os.Getenv("XDG_RUNTIME_DIR")
}

func instanceSig() string {
	return os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
}

func eventSocket() string {
	return fmt.Sprintf("%s/hypr/%s/.socket2.sock", runtimeDir(), instanceSig())
}

type Event struct {
	Name    string
	Payload string
	Raw     string
}

func SubscribeEvents() (<-chan Event, error) {
	path := eventSocket()
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, fmt.Errorf("failed to connect event socket %s: %w", path, err)
	}

	out := make(chan Event, 128)

	go func() {
		defer close(out)
		defer conn.Close()

		sc := bufio.NewScanner(conn)
		for sc.Scan() {
			line := sc.Text()
			name, payload := SplitEvent(line)
			out <- Event{
				Name:    name,
				Payload: payload,
				Raw:     line,
			}
		}
	}()

	return out, nil
}

func SplitEvent(line string) (string, string) {
	parts := strings.SplitN(line, ">>", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return line, ""
}
