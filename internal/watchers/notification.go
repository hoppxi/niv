package watchers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/hoppxi/wigo/internal/subscribe"
)

var (
	historyPath    = filepath.Join(os.Getenv("HOME"), ".local/share/wigo/notification-history.jsonl")
	dndPath        = filepath.Join(os.Getenv("HOME"), ".local/share/wigo/dnd.jsonl")
	cacheDir       = filepath.Join(os.Getenv("HOME"), ".cache/wigo/images")
	execArgsGlobal []string

	// Global DBus connection to ensure signals are emitted from the owner
	dbusConn *dbus.Conn

	activeNotifications = struct {
		mu   sync.Mutex
		data map[uint32]subscribe.Notification
	}{data: make(map[uint32]subscribe.Notification)}
)

// Ensure cache directory exists
func init() {
	os.MkdirAll(cacheDir, 0755)
}

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
	path := dndPath
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
	data, err := os.ReadFile(historyPath)
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

// CloseNotificationByID closes it, updates history, and updates UI
func (h *notificationHelper) CloseNotificationByID(id uint32) {
	activeNotifications.mu.Lock()
	delete(activeNotifications.data, id)
	snapshot := getSortedSnapshot() // Use sorted snapshot
	activeNotifications.mu.Unlock()

	runEWWUpdate("NOTIFICATION", snapshot)

	// Remove from persistence/history
	history := loadHistory()
	newHistory := make([]subscribe.Notification, 0)
	for _, n := range history {
		if n.ID != id {
			newHistory = append(newHistory, n)
		}
	}
	saveHistoryList(newHistory)
	runEWWUpdate("NOTIFICATION_HISTORY", newHistory)
}

// CloseNotificationViewOnly removes from active view but keeps in history
func (h *notificationHelper) CloseNotificationViewOnly(id uint32) {
	activeNotifications.mu.Lock()
	delete(activeNotifications.data, id)
	snapshot := getSortedSnapshot() // Use sorted snapshot
	activeNotifications.mu.Unlock()

	// This sends the updated list (minus the closed one) to EWW.
	// Because it's sorted and complete, it won't clear others.
	runEWWUpdate("NOTIFICATION", snapshot)
}

// InvokeAction is called by the UI (EWW) to trigger an action on the app
func (h *notificationHelper) InvokeAction(id uint32, actionKey string) error {
	if dbusConn == nil {
		return errors.New("dbus connection not established")
	}

	log.Printf("Invoking action: ID=%d, Key=%s", id, actionKey)

	// We must emit the signal on the existing connection that owns the name
	// org.freedesktop.Notifications
	err := dbusConn.Emit(
		"/org/freedesktop/Notifications",
		"org.freedesktop.Notifications.ActionInvoked",
		id,
		actionKey,
	)

	if err != nil {
		log.Printf("Failed to emit ActionInvoked: %v", err)
		return err
	}

	// Usually, after an action is invoked, the notification should close
	h.CloseNotificationViewOnly(id)

	// Also signal that the notification is closed due to user action (Reason 2)
	dbusConn.Emit(
		"/org/freedesktop/Notifications",
		"org.freedesktop.Notifications.NotificationClosed",
		id,
		uint32(2),
	)

	return nil
}

// SendTestNotification sends a robust notification with images and actions for testing
func (h *notificationHelper) SendTestNotification() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")

	hints := map[string]dbus.Variant{
		"urgency":  dbus.MakeVariant(byte(2)), // Critical
		"category": dbus.MakeVariant("test.email"),
	}

	call := obj.Call("org.freedesktop.Notifications.Notify", 0,
		"Wigo Tester",
		uint32(0),
		"mail-unread", // Icon name
		"Test Notification",
		"This is a <b>bold</b> body with <a href='https://google.com'>links</a>.\nIt is critical urgency.",
		[]string{"default", "Reply", "delete", "Delete Email"}, // Actions pairs: key, label
		hints,
		int32(0), // No timeout
	)

	if call.Err != nil {
		return call.Err
	}
	return nil
}

type NotificationDaemon struct{}

func StartNotificationWatcher(execArgs []string) {
	execArgsGlobal = execArgs

	var err error
	dbusConn, err = dbus.ConnectSessionBus()
	if err != nil {
		log.Fatal("DBus connect error:", err)
	}
	// Do not defer close here, this runs forever.

	reply, err := dbusConn.RequestName("org.freedesktop.Notifications", dbus.NameFlagDoNotQueue)
	if err != nil {
		log.Fatal(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Fatal("Another notification daemon is running. Stop it first.")
	}

	daemon := &NotificationDaemon{}
	dbusConn.ExportAll(daemon, "/org/freedesktop/Notifications", "org.freedesktop.Notifications")

	history := loadHistory()
	if len(history) > 0 {
		runEWWUpdate("NOTIFICATION_HISTORY", history)
	}

	log.Println("Notification daemon started on org.freedesktop.Notifications")
	select {}
}

func (n *NotificationDaemon) Notify(appName string, replacesID uint32, appIcon, summary, body string,
	actions []string, hints map[string]dbus.Variant, expireTimeout int32) uint32 {

	var id uint32
	if replacesID != 0 {
		id = replacesID
	} else {
		nextNotificationID := uint32(time.Now().UnixNano() & 0x7fffffff)
		id = atomic.AddUint32(&nextNotificationID, 1)
		if id == 0 {
			id = atomic.AddUint32(&nextNotificationID, 1)
		}
	}

	// Urgency: 0=low, 1=normal, 2=critical
	urgency := byte(1) // default normal
	if u, ok := hints["urgency"].Value().(byte); ok {
		urgency = u
	}

	imagePath := appIcon
	processedImg, err := extractImageFromHints(hints, id)
	if err == nil && processedImg != "" {
		imagePath = processedImg
	}

	finalTimeout := expireTimeout
	if finalTimeout == -1 {
		switch urgency {
		case 0: // Low
			finalTimeout = 5000
		case 1: // Normal
			finalTimeout = 10000
		case 2: // Critical
			finalTimeout = 0 // Never expire
		}
	} else if urgency == 2 {
		finalTimeout = 0
	}

	notif := subscribe.Notification{
		ID:            id,
		AppName:       appName,
		ReplacesID:    replacesID,
		AppIcon:       imagePath,
		Summary:       summary,
		Body:          body,
		Actions:       actions,
		Hints:         hints,
		ExpireTimeout: finalTimeout,
		Timestamp:     time.Now().Unix(),
	}

	activeNotifications.mu.Lock()
	activeNotifications.data[id] = notif
	snapshot := getSortedSnapshot()
	activeNotifications.mu.Unlock()

	appendHistory(notif)
	runEWWUpdate("NOTIFICATION_HISTORY", loadHistory())

	if !NotificationHelper.IsDND() {
		runEWWUpdate("NOTIFICATION", snapshot)
	}

	if finalTimeout > 0 {
		go scheduleAutoClear(id, finalTimeout)
	}

	if len(execArgsGlobal) > 0 {
		go runCommand(execArgsGlobal, notif)
	}

	return id
}

func getSortedSnapshot() []subscribe.Notification {
	snapshot := make([]subscribe.Notification, 0, len(activeNotifications.data))
	for _, n := range activeNotifications.data {
		snapshot = append(snapshot, n)
	}

	sort.Slice(snapshot, func(i, j int) bool {
		return snapshot[i].ID > snapshot[j].ID
	})

	return snapshot
}

func extractImageFromHints(hints map[string]dbus.Variant, id uint32) (string, error) {

	if path, ok := hints["image-path"].Value().(string); ok && path != "" {
		return path, nil
	}
	if path, ok := hints["image_path"].Value().(string); ok && path != "" {
		return path, nil
	}

	var imgDataStruct []any

	if val, ok := hints["image-data"]; ok {
		if s, ok := val.Value().([]any); ok {
			imgDataStruct = s
		}
	} else if val, ok := hints["icon_data"]; ok {

		if s, ok := val.Value().([]any); ok {
			imgDataStruct = s
		}
	}

	if len(imgDataStruct) == 7 {
		width := imgDataStruct[0].(int32)
		height := imgDataStruct[1].(int32)
		// rowstride := imgDataStruct[2].(int32)
		hasAlpha := imgDataStruct[3].(bool)
		// bitsPerSample := imgDataStruct[4].(int32) // mostly 8
		channels := imgDataStruct[5].(int32)
		pixelData := imgDataStruct[6].([]byte)

		img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))

		ptr := 0
		for y := 0; y < int(height); y++ {
			for x := 0; x < int(width); x++ {
				if ptr+int(channels) > len(pixelData) {
					break
				}

				r := pixelData[ptr]
				g := pixelData[ptr+1]
				b := pixelData[ptr+2]
				a := uint8(255)
				if hasAlpha && channels > 3 {
					a = pixelData[ptr+3]
				}

				img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: a})
				ptr += int(channels)
			}
			// Skip rowstride padding if any (rowstride is bytes per row)
			// ptr should align with rowstride, but simple iteration often works for packed data
			// A strict implementation would jump ptr based on rowstride
		}

		// Save to cache
		fileName := filepath.Join(cacheDir, fmt.Sprintf("%d.png", id))
		f, err := os.Create(fileName)
		if err != nil {
			return "", err
		}
		defer f.Close()

		if err := png.Encode(f, img); err != nil {
			return "", err
		}
		return fileName, nil
	}

	return "", nil
}

func (n *NotificationDaemon) GetServerInformation() (string, string, string, string) {
	return "wigo", "hoppxi", "1.0", "1.2"
}

func (n *NotificationDaemon) GetCapabilities() []string {
	return []string{"body", "body-markup", "actions", "icon-static", "persistence"}
}

func (n *NotificationDaemon) CloseNotification(id uint32) {
	activeNotifications.mu.Lock()
	delete(activeNotifications.data, id)
	snapshot := getSortedSnapshot()
	activeNotifications.mu.Unlock()

	runEWWUpdate("NOTIFICATION", snapshot)

	if dbusConn != nil {
		dbusConn.Emit("/org/freedesktop/Notifications", "org.freedesktop.Notifications.NotificationClosed", id, uint32(3)) // 3 = closed by call
	}
}

func scheduleAutoClear(id uint32, timeout int32) {
	time.Sleep(time.Duration(timeout) * time.Millisecond)

	activeNotifications.mu.Lock()
	if _, exists := activeNotifications.data[id]; !exists {
		activeNotifications.mu.Unlock()
		return
	}
	delete(activeNotifications.data, id)
	snapshot := getSortedSnapshot()
	activeNotifications.mu.Unlock()

	if !NotificationHelper.IsDND() {
		runEWWUpdate("NOTIFICATION", snapshot)
	}

	if dbusConn != nil {
		dbusConn.Emit("/org/freedesktop/Notifications", "org.freedesktop.Notifications.NotificationClosed", id, uint32(1))
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

func saveHistoryList(history []subscribe.Notification) {
	dir := filepath.Dir(historyPath)
	os.MkdirAll(dir, 0755)
	f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error updating history:", err)
		return
	}
	defer f.Close()
	for _, n := range history {
		data, _ := json.Marshal(n)
		f.Write(data)
		f.Write([]byte("\n"))
	}
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

	sort.Slice(history, func(i, j int) bool { return history[i].ID > history[j].ID })

	return history
}

func runEWWUpdate(varName string, data any) {
	jsonBytes, _ := json.Marshal(data)
	cmd := exec.Command("eww", "update", varName+"="+string(jsonBytes))
	if err := cmd.Run(); err != nil {
		log.Printf("eww update error: %v", err)
	}
}

func runCommand(args []string, n subscribe.Notification) {
	jsonBytes, _ := json.Marshal(n)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(os.Environ(), "NOTIFICATION="+string(jsonBytes))

	if err := cmd.Run(); err != nil {
		log.Printf("exec error: %v", err)
	}
}
