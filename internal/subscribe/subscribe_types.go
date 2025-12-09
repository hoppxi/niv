package subscribe

import (
	"github.com/godbus/dbus/v5"
)

type Notification struct {
	ID            uint32                  `json:"id"`
	AppName    		string                  `json:"app_name"`
	ReplacesID 		uint32                  `json:"replaces_id"`
	AppIcon    		string                  `json:"app_icon"`
	Summary    		string                  `json:"summary"`
	Body       		string                  `json:"body"`
	Actions    		[]string                `json:"actions"`
	Hints      		map[string]dbus.Variant `json:"hints"`
	Timeout    		int32                   `json:"timeout"`
	ExpireTimeout int32								 	  `json:"expire_timeout"`
	Timestamp 		int64								 	  `json:"timestamp"`
}

type NetworkEvent struct{}
type BluetoothEvent struct{}
type AudioEvent struct{}
// MediaEvent represents a single MPRIS property change.
type MediaEvent struct {
	Player   string      // DBus object path of the media player
	Property string      // Property name that changed (e.g., PlaybackStatus, Metadata)
	Value    interface{} // New value of the property
}