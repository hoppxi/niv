package iconsinfo

import (
	"encoding/json"

	"github.com/hoppxi/niv/pkgs/audioinfo"
	"github.com/hoppxi/niv/pkgs/btinfo"
	"github.com/hoppxi/niv/pkgs/netinfo"
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
		return "mic-off"
	}
	return "mic"
}

func mapOutputVolume(a *audioinfo.AudioInfo) string {
	if a.Output.Muted {
		return "volume-x"
	}
	switch {
	case a.Output.Level >= 70:
		return "volume-2"
	case a.Output.Level >= 40 && a.Output.Level < 70:
		return "volume-1"
	case a.Output.Level >= 0 && a.Output.Level < 40:
		return "volume"
	default:
		return "Volume"
	}
}

func mapNetwork(n *netinfo.NetworkInfo) string {
	if !n.Enabled {
		return "wifi-off"
	}
	if n.CurrentConnection.Type == "ethernet" {
		return "ethernet-port"
	}
	if !n.Connected {
		return "wifi-off"
	}
	switch {
	case n.CurrentConnection.Speed >= 70:
		return "wifi-high"
	case n.CurrentConnection.Speed >= 40 && n.CurrentConnection.Speed < 70:
		return "wifi-medium"
	case n.CurrentConnection.Speed >= 1 && n.CurrentConnection.Speed < 40:
		return "wifi-low"
	default:
		return "wifi"
	}
}

func mapBluetooth(b *btinfo.BluetoothInfo) string {
	if b.Enabled {
		return "bluetooth"
	}
	return "bluetooth-off"
}
