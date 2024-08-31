package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/go-redis/redis/v8"
	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/web"
)

type StreamingService struct {
	redis *redis.Client
}

func New(redis *redis.Client) *StreamingService {
	return &StreamingService{
		redis: redis,
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

func (s *StreamingService) GetStream(streamType StreamType) *Stream {
	return &Stream{
		streamType: streamType,
	}
}

func (s Stream) GetChannelID() string {
	return fmt.Sprintf("%d:%d", s.runnerID, s.streamType.id)
}

type MessageType string

const (
	EventMessage   MessageType = "event"
	SysstatMessage MessageType = "sysstat"
	NotifyMessage  MessageType = "notify"
)

type Message2 struct {
	Type MessageType
	Body any
}

type Message any

func NewStandardMessage(eventCode entity.EventCode, pageCode web.PageCode) Message2 {
	return Message2{
		Type: EventMessage,
		Body: web.StandardMessage{
			EventCode: eventCode,
			PageCode:  pageCode,
		},
	}
}

func NewStandardMessageWithProgress(eventCode entity.EventCode, progress int, pageCode web.PageCode) Message2 {
	msg := web.StandardMessage{
		EventCode: eventCode,
		PageCode:  pageCode,
	}
	msg.Extra.Progress = progress
	return Message2{
		Type: EventMessage,
		Body: msg,
	}
}

func NewStandardMessageWithTextData(eventCode entity.EventCode, textData string, pageCode web.PageCode) Message2 {
	msg := web.StandardMessage{
		EventCode: eventCode,
		PageCode:  pageCode,
	}
	msg.Extra.TextData = textData
	return Message2{
		Type: EventMessage,
		Body: msg,
	}
}

func NewInfoMessage(infoCode entity.InfoCode, isError bool) Message2 {
	return Message2{
		Type: NotifyMessage,
		Body: web.InfoMessage{
			InfoCode: infoCode,
			IsError:  isError,
		},
	}
}

func NewSysstatMessage(cpuUsage float64, time int64) Message2 {
	return Message2{
		Type: SysstatMessage,
		Body: web.SysstatMessage{
			CPUUsage: cpuUsage,
			Time:     time,
		},
	}
}

func (s *StreamingService) publishEvent2(ctx context.Context, stream *Stream, message Message2) error {
	switch message.Type {
	case EventMessage:
		body, err := json.Marshal(message.Body)
		if err != nil {
			return err
		}
		if _, err := s.redis.Set(ctx, "current-state", body, 0).Result(); err != nil {
			return err
		}

	case SysstatMessage:
		if _, err := s.redis.Pipelined(ctx, func(p redis.Pipeliner) error {
			data, err := json.Marshal(message.Body)
			if err != nil {
				return err
			}

			if stream.streamType.historyCount > 0 {
				p.LPush(ctx, "sysstat-history", data)
				p.LTrim(ctx, "sysstat-history", 0, int64(stream.streamType.historyCount)-1)
			}
			return nil
		}); err != nil {
			return err
		}
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if _, err := s.redis.Publish(ctx, "events", data).Result(); err != nil {
		return err
	}

	return nil
}

func (s *StreamingService) PublishEvent2(ctx context.Context, stream *Stream, message Message2) {
	if err := s.publishEvent2(ctx, stream, message); err != nil {
		slog.Error("Failed to publish event: %v", slog.Any("error", err))
	}
}

func (s *StreamingService) PublishEvent(ctx context.Context, stream *Stream, message Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	if _, err := s.redis.Pipelined(ctx, func(p redis.Pipeliner) error {
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

func (s *StreamingService) SubscribeEvent(ctx context.Context, stream *Stream) (*redis.PubSub, [][]byte, error) {
	channelID := stream.GetChannelID()

	statusHistory, err := s.redis.LRange(ctx, "status-history:"+channelID, 0, -1).Result()
	if err != nil {
		return nil, nil, err
	}

	historyBytes := make([][]byte, len(statusHistory))
	for i, entry := range statusHistory {
		historyBytes[len(historyBytes)-1-i] = []byte(entry)
	}

	subscription := s.redis.Subscribe(ctx, "status:"+channelID)

	return subscription, historyBytes, nil
}

type Subscription struct {
	subscription   *redis.PubSub
	CurrentState   []byte
	SysstatHistory [][]byte
}

func (s *Subscription) Close() error {
	return s.subscription.Close()
}

func (s *Subscription) Channel() chan Message2 {
	outChannel := make(chan Message2)

	go func() {
		defer close(outChannel)

		channel := s.subscription.Channel()

		for msg := range channel {
			var outMsg Message2
			if err := json.Unmarshal([]byte(msg.Payload), &outMsg); err != nil {
				continue
			}

			outChannel <- outMsg
		}
	}()

	return outChannel
}

func (s *StreamingService) SubscribeEvent2(ctx context.Context) (*Subscription, error) {
	currentState, err := s.redis.Get(ctx, "current-state").Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	sysstatHistory, err := s.redis.LRange(ctx, "sysstat-history", 0, -1).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	historyData := make([][]byte, len(sysstatHistory))
	for i, entry := range sysstatHistory {
		historyData[len(historyData)-1-i] = []byte(entry)
	}

	subscription := s.redis.Subscribe(ctx, "events")

	return &Subscription{
		subscription:   subscription,
		CurrentState:   []byte(currentState),
		SysstatHistory: historyData,
	}, nil
}

func (s *StreamingService) ClearHistory(ctx context.Context, stream *Stream) error {
	channelID := stream.GetChannelID()
	if _, err := s.redis.Del(ctx, "status-history:"+channelID).Result(); err != nil {
		return err
	}
	return nil
}

func (s *StreamingService) ClearSysstat2(ctx context.Context) error {
	if _, err := s.redis.Del(ctx, "sysstat-history").Result(); err != nil {
		return err
	}
	return nil
}
