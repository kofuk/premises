package kvs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
)

type Store interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	GetSet(ctx context.Context, key string, value []byte, ttl time.Duration) ([]byte, error)
	Del(ctx context.Context, key ...string) error
}

type KeyValueStore struct {
	c Store
}

func New(c Store) KeyValueStore {
	return KeyValueStore{
		c: c,
	}
}

func (self KeyValueStore) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	ser, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return self.c.Set(ctx, key, ser, ttl)
}

func (self KeyValueStore) Get(ctx context.Context, key string, result any) error {
	data, err := self.c.Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, result)
}

func (self KeyValueStore) GetSet(ctx context.Context, key string, value any, ttl time.Duration, oldValue any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	oldData, err := self.c.GetSet(ctx, key, data, ttl)
	if err != nil {
		return err
	}
	return json.Unmarshal(oldData, oldValue)
}

func (self KeyValueStore) Del(ctx context.Context, key ...string) error {
	return self.c.Del(ctx, key...)
}

type RedisStore struct {
	redis *redis.Client
}

func NewRedis(redis *redis.Client) RedisStore {
	return RedisStore{
		redis: redis,
	}
}

func (self RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if _, err := self.redis.Set(ctx, key, value, ttl).Result(); err != nil {
		return err
	}
	return nil
}

func (self RedisStore) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := self.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func (self RedisStore) GetSet(ctx context.Context, key string, value []byte, ttl time.Duration) ([]byte, error) {
	val, err := self.redis.SetArgs(ctx, key, value, redis.SetArgs{Get: true, TTL: ttl}).Result()
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func (self RedisStore) Del(ctx context.Context, keys ...string) error {
	if _, err := self.redis.Del(ctx, keys...).Result(); err != nil {
		return err
	}
	return nil
}
