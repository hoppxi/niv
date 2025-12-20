package subscribe

import (
	"log"

	"github.com/godbus/dbus/v5"
)

func NotificationChannel() chan Notification {
    out := make(chan Notification, 10)

    go func() {
				conn, err := dbus.ConnectSessionBus()
				if err != nil { log.Fatal(err) }

				reply, err := conn.RequestName("org.freedesktop.Notifications",
						dbus.NameFlagDoNotQueue)
				if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
						log.Fatal("Cannot claim org.freedesktop.Notifications. Another daemon is running.")
				}


        // Add match
        rule := "type='method_call',interface='org.freedesktop.Notifications',member='Notify'"
        call := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule)
        if call.Err != nil {
            log.Fatal("Failed AddMatch:", call.Err)
        }

        ch := make(chan *dbus.Message, 10)
        conn.Eavesdrop(ch)

        for msg := range ch {
            iface, _ := msg.Headers[dbus.FieldInterface].Value().(string)
            member, _ := msg.Headers[dbus.FieldMember].Value().(string)

            if iface != "org.freedesktop.Notifications" || member != "Notify" {
                continue
            }

            // Extract notification arguments
            var (
                appName    string
                replacesID uint32
                appIcon    string
                summary    string
                body       string
                actions    []string
                hints      map[string]dbus.Variant
                timeout    int32
            )

            err := dbus.Store(msg.Body,
                &appName, &replacesID, &appIcon, &summary, &body,
                &actions, &hints, &timeout,
            )
            if err != nil {
                log.Println("Failed to parse Notify():", err)
                continue
            }

            out <- Notification{
                AppName:    appName,
                ReplacesID: replacesID,
                AppIcon:    appIcon,
                Summary:    summary,
                Body:       body,
                Actions:    actions,
                Hints:      hints,
                Timeout:    timeout,
            }
        }
    }()

    return out
}
