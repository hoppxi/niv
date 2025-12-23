package subscribe

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type LEDsEventChannels struct {
	CapsOn    chan struct{}
	CapsOff   chan struct{}
	NumOn     chan struct{}
	NumOff    chan struct{}
	ScrollOn  chan struct{}
	ScrollOff chan struct{}
}

func LEDsEvents() *LEDsEventChannels {
	ev := &LEDsEventChannels{
		CapsOn:    make(chan struct{}),
		CapsOff:   make(chan struct{}),
		NumOn:     make(chan struct{}),
		NumOff:    make(chan struct{}),
		ScrollOn:  make(chan struct{}),
		ScrollOff: make(chan struct{}),
	}

	go watchLockLEDs(ev)
	return ev
}

func discoverLockLEDs() map[string]string {
	leds := make(map[string]string)

	entries, err := os.ReadDir("/sys/class/leds")
	if err != nil {
		return leds
	}

	for _, e := range entries {
		name := e.Name()
		switch {
		case strings.HasSuffix(name, "::capslock"):
			leds["caps"] = filepath.Join("/sys/class/leds", name, "brightness")
		case strings.HasSuffix(name, "::numlock"):
			leds["num"] = filepath.Join("/sys/class/leds", name, "brightness")
		case strings.HasSuffix(name, "::scrolllock"):
			leds["scroll"] = filepath.Join("/sys/class/leds", name, "brightness")
		}
	}

	return leds
}

func readBrightness(path string) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	v, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return false, err
	}

	return v == 1, nil
}

func watchLockLEDs(ev *LEDsEventChannels) {
	leds := discoverLockLEDs()
	prev := make(map[string]bool)

	for k, p := range leds {
		if v, err := readBrightness(p); err == nil {
			prev[k] = v
		}
	}

	for {
		time.Sleep(100 * time.Millisecond)

		for k, p := range leds {
			cur, err := readBrightness(p)
			if err != nil {
				continue
			}

			if cur != prev[k] {
				switch k {
				case "caps":
					if cur {
						ev.CapsOn <- struct{}{}
					} else {
						ev.CapsOff <- struct{}{}
					}
				case "num":
					if cur {
						ev.NumOn <- struct{}{}
					} else {
						ev.NumOff <- struct{}{}
					}
				case "scroll":
					if cur {
						ev.ScrollOn <- struct{}{}
					} else {
						ev.ScrollOff <- struct{}{}
					}
				}
				prev[k] = cur
			}
		}
	}
}
