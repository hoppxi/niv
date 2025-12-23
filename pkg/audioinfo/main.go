package audioinfo

import (
	"encoding/json"
	"fmt"

	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
)

type AudioDevice struct {
	Name  string `json:"name"`
	Level int    `json:"level"` // percent 0-100
	Muted bool   `json:"muted"`
}

type AudioInfo struct {
	Output AudioDevice `json:"output"`
	Input  AudioDevice `json:"input"`
}

func channelVolumesToPercent(cv proto.ChannelVolumes) int {
	if len(cv) == 0 {
		return 100
	}
	var sum float64
	for _, v := range cv {
		sum += float64(v) / float64(proto.VolumeNorm) * 100.0
	}
	pct := int(sum/float64(len(cv)) + 0.5)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct
}

func getDeviceInfo(c *pulse.Client, isSink bool) (AudioDevice, error) {
	var dev AudioDevice
	if isSink {
		s, err := c.DefaultSink()
		if err != nil {
			return dev, fmt.Errorf("failed to get default sink: %w", err)
		}
		id := s.ID()
		name := s.Name()
		var reply proto.GetSinkInfoReply
		req := proto.GetSinkInfo{SinkIndex: proto.Undefined, SinkName: id}
		if err := c.RawRequest(&req, &reply); err != nil {
			return dev, fmt.Errorf("failed to request sink info: %w", err)
		}
		dev = AudioDevice{
			Name:  name,
			Level: channelVolumesToPercent(reply.ChannelVolumes),
			Muted: reply.Mute,
		}
		return dev, nil
	}

	src, err := c.DefaultSource()
	if err != nil {
		return dev, fmt.Errorf("failed to get default source: %w", err)
	}
	id := src.ID()
	name := src.Name()
	var reply proto.GetSourceInfoReply
	req := proto.GetSourceInfo{SourceIndex: proto.Undefined, SourceName: id}
	if err := c.RawRequest(&req, &reply); err != nil {
		return dev, fmt.Errorf("failed to request source info: %w", err)
	}
	dev = AudioDevice{
		Name:  name,
		Level: channelVolumesToPercent(reply.ChannelVolumes),
		Muted: reply.Mute,
	}
	return dev, nil
}

func GetAudioInfo() (*AudioInfo, error) {
	c, err := pulse.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create pulse client: %w", err)
	}
	defer c.Close()

	out, err := getDeviceInfo(c, true)
	if err != nil {
		return nil, err
	}
	in, err := getDeviceInfo(c, false)
	if err != nil {
		return nil, err
	}
	info := &AudioInfo{
		Output: out,
		Input:  in,
	}
	return info, nil
}

func GetAudioInfoJSON() ([]byte, error) {
	info, err := GetAudioInfo()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(info, "", "  ")
}
