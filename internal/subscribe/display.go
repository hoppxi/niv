package subscribe

import (
	"log"
	"strings"
	"syscall"
)

func DisplayEvents() <-chan struct{} {
	events := make(chan struct{}, 1)

	go func() {
		fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_KOBJECT_UEVENT)
		if err != nil {
			log.Printf("subscribe: failed to open netlink socket: %v", err)
			return
		}
		defer syscall.Close(fd)

		addr := &syscall.SockaddrNetlink{
			Family: syscall.AF_NETLINK,
			Groups: 1, // listen to broadcast uevents
		}
		if err := syscall.Bind(fd, addr); err != nil {
			log.Printf("subscribe: failed to bind netlink socket: %v", err)
			return
		}

		buf := make([]byte, 4096)
		for {
			n, _, err := syscall.Recvfrom(fd, buf, 0)
			if err != nil {
				log.Printf("subscribe: netlink recv error: %v", err)
				continue
			}

			msg := string(buf[:n])
			if strings.Contains(msg, "SUBSYSTEM=backlight") && strings.Contains(msg, "ACTION=change") {
				select {
				case events <- struct{}{}:
				default:
				}
			}
		}
	}()

	return events
}
