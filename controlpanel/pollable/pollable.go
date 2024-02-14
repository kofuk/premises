package pollable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
)

var Cancelled = errors.New("Cancelled")

type PollableActionService struct {
	redis *redis.Client
	key   string
}

func New(redis *redis.Client, key string) *PollableActionService {
	return &PollableActionService{
		redis: redis,
		key:   key,
	}
}

func (self *PollableActionService) Push(ctx context.Context, runnerId string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if _, err := self.redis.RPush(ctx, fmt.Sprintf("%s:%s", self.key, runnerId), string(jsonData)).Result(); err != nil {
		return err
	}
	if _, err := self.redis.Publish(ctx, fmt.Sprintf("%s:notify:%s", self.key, runnerId), "").Result(); err != nil {
		return err
	}

	return nil
}

func (self *PollableActionService) getAction(ctx context.Context, runnerId string) (string, error) {
	act, err := self.redis.LPop(ctx, fmt.Sprintf("%s:%s", self.key, runnerId)).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}
	return act, nil
}

func (self *PollableActionService) Wait(ctx context.Context, runnerId string) (string, error) {
	subscription := self.redis.Subscribe(ctx, fmt.Sprintf("%s:notify:%s", self.key, runnerId))
	defer subscription.Close()

	act, err := self.getAction(ctx, runnerId)
	if err != nil {
		return act, nil
	}

	c := subscription.Channel()
out:
	for {
		select {
		case <-ctx.Done():
			break out

		case <-c:
			act, err := self.getAction(ctx, runnerId)
			if err != nil {
				return act, nil
			}
			return act, nil
		}
	}

	return "", Cancelled
}
