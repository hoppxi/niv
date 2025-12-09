package watchers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/hoppxi/niv/internal/subscribe"
)

var (
	historyPath        = filepath.Join(os.Getenv("HOME"), ".local/share/niv/notification-history.jsonl")
	dndPath            = filepath.Join(os.Getenv("HOME"), ".local/share/niv/dnd.jsonl")
	execArgsGlobal     []string
	activeNotifications = struct {
		mu   sync.Mutex
		data map[uint32]subscribe.Notification
	}{data: make(map[uint32]subscribe.Notification)}
)

type notificationHelper struct{}

var NotificationHelper = &notificationHelper{}

func (h *notificationHelper) SetDNDState(state string) error {
	dir := filepath.Dir(dndPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Println("failed to create DND directory:", err)
		return err
	}

	switch state {
	case "on":
		return os.WriteFile(dndPath, []byte("on"), 0644)
	case "off":
		return os.WriteFile(dndPath, []byte("off"), 0644)
	default:
		return errors.New("use --dnd on|off")
	}
}

func (h *notificationHelper) IsDND() bool {
	data, err := os.ReadFile(dndPath)
	return err == nil && string(data) == "on"
}

func (h *notificationHelper) GetDNDState() bool {
	return h.IsDND()
}

func (h *notificationHelper) ClearHistory() error {
	if err := os.Remove(historyPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	runEWWUpdate("NOTIFICATION_HISTORY", []subscribe.Notification{})
	return nil
}

func (h *notificationHelper) ToggleDND() bool {
	path := filepath.Join(os.Getenv("HOME"), ".local/share/niv/dnd.jsonl")
	state := false
	data, err := os.ReadFile(path)
	if err == nil && string(data) == "on" {
		state = true
	}

	newState := "off"
	if !state {
		newState = "on"
	}

	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)
	os.WriteFile(path, []byte(newState), 0644)

	return newState == "on"
}

func (h *notificationHelper) GetHistoryCount() int {
	path := filepath.Join(os.Getenv("HOME"), ".local/share/niv/notification-history.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	lines := bytes.Split(data, []byte("\n"))
	count := 0
	for _, l := range lines {
		if len(l) > 0 {
			count++
		}
	}
	return count
}

func (h *notificationHelper) CloseNotificationByID(id uint32) {
	activeNotifications.mu.Lock()
	delete(activeNotifications.data, id)

	snapshot := make([]subscribe.Notification, 0, len(activeNotifications.data))
	for _, n := range activeNotifications.data {
		snapshot = append(snapshot, n)
	}
	activeNotifications.mu.Unlock()

	runEWWUpdate("NOTIFICATION", snapshot)

	history := loadHistory()
	newHistory := make([]subscribe.Notification, 0)
	for _, n := range history {
		if n.ID != id {
			newHistory = append(newHistory, n)
		}
	}

	dir := filepath.Dir(historyPath)
	os.MkdirAll(dir, 0755)
	f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error updating history:", err)
		return
	}
	defer f.Close()
	for _, n := range newHistory {
		data, _ := json.Marshal(n)
		f.Write(data)
		f.Write([]byte("\n"))
	}

	runEWWUpdate("NOTIFICATION_HISTORY", newHistory)
}

func (h *notificationHelper) CloseNotificationViewOnly(id uint32) {
	activeNotifications.mu.Lock()
	delete(activeNotifications.data, id)

	snapshot := make([]subscribe.Notification, 0, len(activeNotifications.data))
	for _, n := range activeNotifications.data {
		snapshot = append(snapshot, n)
	}
	activeNotifications.mu.Unlock()

	runEWWUpdate("NOTIFICATION", snapshot)
}

func (h *notificationHelper) InvokeAction(id uint32, action string) error {
    conn, err := dbus.ConnectSessionBus()
    if err != nil {
        return err
    }
    defer conn.Close()

    // Send ActionInvoked signal
    return conn.Emit(
        "/org/freedesktop/Notifications",
        "org.freedesktop.Notifications.ActionInvoked",
        id,
        action,
    )
}


type NotificationDaemon struct{}

func StartNotificationWatcher(execArgs []string) {
	execArgsGlobal = execArgs

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatal("DBus connect error:", err)
	}
	defer conn.Close()

	reply, err := conn.RequestName("org.freedesktop.Notifications", dbus.NameFlagDoNotQueue)
	if err != nil {
		log.Fatal(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Fatal("Another notification daemon is running. Stop it first.")
	}

	daemon := &NotificationDaemon{}
	conn.ExportAll(daemon, "/org/freedesktop/Notifications", "org.freedesktop.Notifications")

	history := loadHistory()
	if len(history) > 0 {
		runEWWUpdate("NOTIFICATION_HISTORY", history)
	}

	log.Println("Notification daemon started.")
	select {}
}

func (n *NotificationDaemon) Notify(appName string, replacesID uint32, appIcon, summary, body string,
	actions []string, hints map[string]dbus.Variant, expireTimeout int32) uint32 {

	nextNotificationID := uint32(time.Now().UnixNano() & 0x7fffffff)
	id := atomic.AddUint32(&nextNotificationID, 1)
	if id == 0 {
		id = atomic.AddUint32(&nextNotificationID, 1)
	}

	notif := subscribe.Notification{
		ID:            id,
		AppName:       appName,
		ReplacesID:    replacesID,
		AppIcon:       appIcon,
		Summary:       summary,
		Body:          body,
		Actions:       actions,
		Hints:         hints,
		ExpireTimeout: expireTimeout,
		Timestamp:     time.Now().Unix(),
	}

	activeNotifications.mu.Lock()
	activeNotifications.data[id] = notif

	// Take snapshot of active notifications for NOTIFICATION
	snapshot := make([]subscribe.Notification, 0, len(activeNotifications.data))
	for _, n := range activeNotifications.data {
		snapshot = append(snapshot, n)
	}
	activeNotifications.mu.Unlock()

	// Append to history
	appendHistory(notif)

	// Update full history to EWW
	history := loadHistory()
	runEWWUpdate("NOTIFICATION_HISTORY", history)

	// Update current notifications if DND is off
	if !NotificationHelper.IsDND() {
		runEWWUpdate("NOTIFICATION", snapshot)
	}

	// Schedule auto-clear
	go scheduleAutoClear(id, notif)

	// Optionally run external command
	if len(execArgsGlobal) > 0 {
		go runCommand(execArgsGlobal, notif)
	}

	return id
}

func (n *NotificationDaemon) GetServerInformation() (string, string, string, string) {
	return "niv", "hoppxi", "1.0", "1.2"
}

func (n *NotificationDaemon) GetCapabilities() []string {
	return []string{"body", "body-markup", "actions", "icon-static"}
}

func (n *NotificationDaemon) CloseNotification(id uint32) {
	activeNotifications.mu.Lock()
	delete(activeNotifications.data, id)
	snapshot := make([]subscribe.Notification, 0, len(activeNotifications.data))
	for _, n := range activeNotifications.data {
		snapshot = append(snapshot, n)
	}
	activeNotifications.mu.Unlock()

	runEWWUpdate("NOTIFICATION", snapshot)
}

func scheduleAutoClear(id uint32, notif subscribe.Notification) {
	timeout := notif.ExpireTimeout
	if timeout <= 0 {
		timeout = 10000
	}
	time.Sleep(time.Duration(timeout) * time.Millisecond)

	activeNotifications.mu.Lock()
	delete(activeNotifications.data, id)
	snapshot := make([]subscribe.Notification, 0, len(activeNotifications.data))
	for _, n := range activeNotifications.data {
		snapshot = append(snapshot, n)
	}
	activeNotifications.mu.Unlock()

	if !NotificationHelper.IsDND() {
		runEWWUpdate("NOTIFICATION", snapshot)
	}
}

func appendHistory(n subscribe.Notification) {
	dir := filepath.Dir(historyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Println("failed to create history directory:", err)
		return
	}

	f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("history write error:", err)
		return
	}
	defer f.Close()

	data, _ := json.Marshal(n)
	f.Write(data)
	f.Write([]byte("\n"))
}

func loadHistory() []subscribe.Notification {
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return nil
	}

	var history []subscribe.Notification
	lines := bytes.SplitSeq(data, []byte("\n"))
	for l := range lines {
		if len(l) == 0 {
			continue
		}
		var notif subscribe.Notification
		if err := json.Unmarshal(l, &notif); err == nil {
			history = append(history, notif)
		}
	}
	return history
}

func runEWWUpdate(varName string, data any) {
	jsonBytes, _ := json.Marshal(data)
	cmd := exec.Command("eww", "update", varName+"="+string(jsonBytes))
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("eww update error: %v", err)
	}
	if len(out) > 0 {
		log.Printf("eww update output: %s", string(out))
	}
}

// Optionally run external command
func runCommand(args []string, n subscribe.Notification) {
	jsonBytes, _ := json.Marshal(n)
	jsonArg := "NOTIFICATION=" + string(jsonBytes)
	cmd := exec.Command(args[0], append(args[1:], jsonArg)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("exec error: %v", err)
	}
	if len(out) > 0 {
		log.Printf("exec output: %s", string(out))
	}
}

