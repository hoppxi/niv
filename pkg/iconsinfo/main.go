package iconsinfo

import (
	"encoding/json"

	"github.com/hoppxi/wigo/pkg/audioinfo"
	"github.com/hoppxi/wigo/pkg/btinfo"
	"github.com/hoppxi/wigo/pkg/netinfo"
)

type IconsInfo struct {
	InputVolume  string `json:"input-volume"`
	OutputVolume string `json:"output-volume"`
	Network      string `json:"network"`
	Bluetooth    string `json:"bluetooth"`
}

func GetIcons() IconsInfo {
	audio, _ := audioinfo.GetAudioInfo()
	net, _ := netinfo.GetNetworkInfo()
	bt, _ := btinfo.GetBluetoothInfo()

	return IconsInfo{
		InputVolume:  mapInputVolume(audio),
		OutputVolume: mapOutputVolume(audio),
		Network:      mapNetwork(net),
		Bluetooth:    mapBluetooth(bt),
	}
}

func GetIconsJSON() ([]byte, error) {
	info := GetIcons()
	return json.MarshalIndent(info, "", "  ")
}

func mapInputVolume(a *audioinfo.AudioInfo) string {
	if a.Input.Muted {
		return "mic_off"
	}
	return "mic"
}

func mapOutputVolume(a *audioinfo.AudioInfo) string {
	if a.Output.Muted {
		return "volume_off"
	}
	switch {
	case a.Output.Level >= 70:
		return "volume_up"
	case a.Output.Level >= 40:
		return "volume_down"
	case a.Output.Level >= 0:
		return "volume_mute"
	default:
		return "volume_off"
	}
}

func mapNetwork(n *netinfo.NetworkInfo) string {
	if !n.Enabled {
		return "wifi_off"
	}

	if n.CurrentConnection.Type == "ethernet" {
		return "settings_ethernet"
	}

	if !n.Connected {
		return "wifi_off"
	}

	switch {
	case n.CurrentConnection.Speed >= 70:
		return "wifi"
	case n.CurrentConnection.Speed >= 40:
		return "wifi_2_bar"
	case n.CurrentConnection.Speed >= 1:
		return "wifi_1_bar"
	default:
		return "wifi"
	}
}

func mapBluetooth(b *btinfo.BluetoothInfo) string {
	if b.Enabled {
		return "bluetooth"
	}
	return "bluetooth_disabled"
}
