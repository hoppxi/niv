package subscribe

import (
	"log"

	"github.com/jfreymuth/pulse/proto"
)

type AudioEvent struct {
	Facility proto.SubscriptionEventType
	Index    uint32
}

func AudioEvents() <-chan AudioEvent {
	out := make(chan AudioEvent, 16)

	client, conn, err := proto.Connect("")
	if err != nil {
		log.Printf("failed to connect to pulse server: %v", err)
		close(out)
		return out
	}
	go func() {
		defer conn.Close()
		ch := make(chan struct{}, 1)

		client.Callback = func(val any) {
			switch val := val.(type) {
			case *proto.SubscribeEvent:
				if val.Event.GetType() == proto.EventChange {
					select {
					case ch <- struct{}{}:
					default:
					}
				}
			}
		}

		err := client.Request(&proto.SetClientName{}, nil)
		if err != nil {
			log.Printf("SetClientName failed: %v", err)
			close(out)
			return
		}

		err = client.Request(&proto.Subscribe{Mask: proto.SubscriptionMaskAll}, nil)
		if err != nil {
			log.Printf("Subscribe failed: %v", err)
			close(out)
			return
		}

		var defaultSinkName string
		serverInfo := proto.GetServerInfoReply{}
		err = client.Request(&proto.GetServerInfo{}, &serverInfo)
		if err != nil {
			log.Printf("GetServerInfo failed: %v", err)
			close(out)
			return
		}
		defaultSinkName = serverInfo.DefaultSinkName

		for range ch {
			repl := proto.GetSinkInfoReply{}
			err = client.Request(&proto.GetSinkInfo{SinkIndex: proto.Undefined, SinkName: defaultSinkName}, &repl)
			if err != nil {
				log.Printf("GetSinkInfo failed: %v", err)
				continue
			}
			var acc int64
			for _, vol := range repl.ChannelVolumes {
				acc += int64(vol)
			}
			acc /= int64(len(repl.ChannelVolumes))
			select {
			case out <- AudioEvent{Facility: proto.EventSink, Index: 0}:
			default:
			}
		}
	}()

	return out
}
