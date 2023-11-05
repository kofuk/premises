package streaming

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	entity "github.com/kofuk/premises/common/entity/web"
)

type Streaming struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Streaming {
	return &Streaming{
		rdb: rdb,
	}
}

type Stream struct {
	runnerID   int
	streamType StreamType
}

type StreamType int

const (
	StandardStream StreamType = iota
	ErrorStream
	SysstatStream
)

func (self *Streaming) GetStream(streamType StreamType) *Stream {
	return &Stream{
		streamType: streamType,
	}
}

func (self Stream) GetChannelID() string {
	return fmt.Sprintf("%d:%d", self.runnerID, self.streamType)
}

type Message any

func NewStandardMessage(eventCode entity.EventCode, pageCode entity.PageCode) Message {
	return &entity.StandardMessage{
		EventCode: eventCode,
		PageCode:  pageCode,
	}
}

func NewStandardMessageWithProgress(eventCode entity.EventCode, progress int, pageCode entity.PageCode) Message {
	return &entity.StandardMessage{
		EventCode: eventCode,
		Progress:  progress,
		PageCode:  pageCode,
	}
}

func NewErrorMessage(eventCode entity.ErrorCode) Message {
	return &entity.ErrorMessage{
		ErrorCode: eventCode,
	}
}

func NewSysstatMessage(cpuUsage float64) Message {
	return &entity.SysstatMessage{
		CPUUsage: cpuUsage,
	}
}

func (self *Streaming) PublishEvent(ctx context.Context, stream *Stream, message Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	if _, err := self.rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
		channelID := stream.GetChannelID()
		p.Set(ctx, "last-status:"+channelID, data, -1)
		p.Publish(ctx, "status:"+channelID, data)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (self *Streaming) SubscribeEvent(ctx context.Context, stream *Stream) (*redis.PubSub, []byte, error) {
	channelID := stream.GetChannelID()
	lastStatus, err := self.rdb.Get(ctx, fmt.Sprintf("last-status:"+channelID)).Result()
	if err != nil && err != redis.Nil {
		return nil, nil, err
	}

	subscription := self.rdb.Subscribe(ctx, "status:"+channelID)

	return subscription, []byte(lastStatus), nil
}
