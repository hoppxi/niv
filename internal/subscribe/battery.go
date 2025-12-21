package subscribe

import (
	"log"
	"strconv"
	"strings"
	"syscall"
)

type BatteryEventsT struct {
	BatteryLow20     <-chan struct{}
	BatteryLow5      <-chan struct{}
	BatteryFull      <-chan struct{}
	ChargerPlugged   <-chan struct{}
	ChargerUnplugged <-chan struct{}
	DynamicChange    <-chan struct{}
}

func BatteryEvents() BatteryEventsT {
	low20 := make(chan struct{}, 1)
	low5 := make(chan struct{}, 1)
	full := make(chan struct{}, 1)
	plugged := make(chan struct{}, 1)
	unplugged := make(chan struct{}, 1)
	dynamic := make(chan struct{}, 1)

	go func() {
		fd, err := syscall.Socket(
			syscall.AF_NETLINK,
			syscall.SOCK_RAW,
			syscall.NETLINK_KOBJECT_UEVENT,
		)
		if err != nil {
			log.Printf("battery: socket error: %v", err)
			return
		}
		defer syscall.Close(fd)

		addr := &syscall.SockaddrNetlink{
			Family: syscall.AF_NETLINK,
			Groups: 1,
		}
		if err := syscall.Bind(fd, addr); err != nil {
			log.Printf("battery: bind error: %v", err)
			return
		}

		buf := make([]byte, 4096)

		for {
			n, _, err := syscall.Recvfrom(fd, buf, 0)
			if err != nil {
				log.Printf("battery: recv error: %v", err)
				continue
			}

			msg := string(buf[:n])

			if !strings.Contains(msg, "SUBSYSTEM=power_supply") {
				continue
			}

			if strings.Contains(msg, "ACTION=change") {
				nonBlock(dynamic)
			}

			if cap := extract(msg, "POWER_SUPPLY_CAPACITY="); cap != "" {
				if p, err := strconv.Atoi(cap); err == nil {
					switch {
					case p <= 5:
						nonBlock(low5)
					case p == 20:
						nonBlock(low20)
					case p == 100:
						nonBlock(full)
					}
				}
			}

			if online := extract(msg, "POWER_SUPPLY_ONLINE="); online != "" {
				switch online {
				case "1":
					nonBlock(plugged)
				case "0":
					nonBlock(unplugged)
				}
			}
		}
	}()

	return BatteryEventsT{
		BatteryLow20:     low20,
		BatteryLow5:      low5,
		BatteryFull:      full,
		ChargerPlugged:   plugged,
		ChargerUnplugged: unplugged,
		DynamicChange:    dynamic,
	}
}

func extract(msg, key string) string {
	// Uevents are null-terminated strings
	strings.SplitSeq(msg, "\x00")
	for part := range strings.SplitSeq(msg, "\x00") {
		if mg, suc := strings.CutPrefix(part, key); suc {
			return mg
		}
	}
	return ""
}

func nonBlock(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}
