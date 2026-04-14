package kvs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
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

func (kvs KeyValueStore) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	ser, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return kvs.c.Set(ctx, key, ser, ttl)
}

func (kvs KeyValueStore) Get(ctx context.Context, key string, result any) error {
	data, err := kvs.c.Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, result)
}

func (kvs KeyValueStore) GetSet(ctx context.Context, key string, value any, ttl time.Duration, oldValue any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	oldData, err := kvs.c.GetSet(ctx, key, data, ttl)
	if err != nil {
		return err
	}
	return json.Unmarshal(oldData, oldValue)
}

func (kvs KeyValueStore) Del(ctx context.Context, key ...string) error {
	return kvs.c.Del(ctx, key...)
}

type RedisStore struct {
	redis *redis.Client
}

func NewRedis(redis *redis.Client) RedisStore {
	return RedisStore{
		redis: redis,
	}
}

func (r RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if _, err := r.redis.Set(ctx, key, value, ttl).Result(); err != nil {
		return err
	}
	return nil
}

func (r RedisStore) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func (r RedisStore) GetSet(ctx context.Context, key string, value []byte, ttl time.Duration) ([]byte, error) {
	val, err := r.redis.SetArgs(ctx, key, value, redis.SetArgs{Get: true, TTL: ttl}).Result()
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func (r RedisStore) Del(ctx context.Context, keys ...string) error {
	if _, err := r.redis.Del(ctx, keys...).Result(); err != nil {
		return err
	}
	return nil
}
