package streaming

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/kofuk/premises/common/entity"
	webEntity "github.com/kofuk/premises/common/entity/web"
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

type StreamType struct {
	id           int
	historyCount int
}

var (
	StandardStream = StreamType{
		id:           1,
		historyCount: 1,
	}
	InfoStream = StreamType{
		id:           2,
		historyCount: 0,
	}
	SysstatStream = StreamType{
		id:           3,
		historyCount: 100,
	}
)

func (self *Streaming) GetStream(streamType StreamType) *Stream {
	return &Stream{
		streamType: streamType,
	}
}

func (self Stream) GetChannelID() string {
	return fmt.Sprintf("%d:%d", self.runnerID, self.streamType.id)
}

type Message any

func NewStandardMessage(eventCode entity.EventCode, pageCode webEntity.PageCode) Message {
	return &webEntity.StandardMessage{
		EventCode: eventCode,
		PageCode:  pageCode,
	}
}

func NewStandardMessageWithProgress(eventCode entity.EventCode, progress int, pageCode webEntity.PageCode) Message {
	return &webEntity.StandardMessage{
		EventCode: eventCode,
		Progress:  progress,
		PageCode:  pageCode,
	}
}

func NewInfoMessage(infoCode entity.InfoCode, isError bool) Message {
	return &webEntity.InfoMessage{
		InfoCode: infoCode,
		IsError:  isError,
	}
}

func NewSysstatMessage(cpuUsage float64) Message {
	return &webEntity.SysstatMessage{
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
		if stream.streamType.historyCount > 0 {
			p.LPush(ctx, "status-history:"+channelID, data)
			p.LTrim(ctx, "status-history:"+channelID, 0, int64(stream.streamType.historyCount)-1)
		}
		p.Publish(ctx, "status:"+channelID, data)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (self *Streaming) SubscribeEvent(ctx context.Context, stream *Stream) (*redis.PubSub, [][]byte, error) {
	channelID := stream.GetChannelID()

	statusHistory, err := self.rdb.LRange(ctx, "status-history:"+channelID, 0, -1).Result()
	if err != nil {
		return nil, nil, err
	}

	historyBytes := make([][]byte, len(statusHistory))
	for i, entry := range statusHistory {
		historyBytes[len(historyBytes)-1-i] = []byte(entry)
	}

	subscription := self.rdb.Subscribe(ctx, "status:"+channelID)

	return subscription, historyBytes, nil
}
