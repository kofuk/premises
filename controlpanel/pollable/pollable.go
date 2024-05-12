package pollable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
)

var ErrCancelled = errors.New("Cancelled")

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

func (pa *PollableActionService) Push(ctx context.Context, runnerId string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if _, err := pa.redis.RPush(ctx, fmt.Sprintf("%s:%s", pa.key, runnerId), string(jsonData)).Result(); err != nil {
		return err
	}
	if _, err := pa.redis.Publish(ctx, fmt.Sprintf("%s:notify:%s", pa.key, runnerId), "").Result(); err != nil {
		return err
	}

	return nil
}

func (pa *PollableActionService) getAction(ctx context.Context, runnerId string) (string, error) {
	act, err := pa.redis.LPop(ctx, fmt.Sprintf("%s:%s", pa.key, runnerId)).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}
	return act, nil
}

func (pa *PollableActionService) Wait(ctx context.Context, runnerId string) (string, error) {
	subscription := pa.redis.Subscribe(ctx, fmt.Sprintf("%s:notify:%s", pa.key, runnerId))
	defer subscription.Close()

	act, err := pa.getAction(ctx, runnerId)
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
			act, err := pa.getAction(ctx, runnerId)
			if err != nil {
				return act, nil
			}
			return act, nil
		}
	}

	return "", ErrCancelled
}
