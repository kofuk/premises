package caching

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
)

type CacheImpl interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Del(ctx context.Context, key ...string) error
}

type Cacher struct {
	c CacheImpl
}

func New(c CacheImpl) Cacher {
	return Cacher{
		c: c,
	}
}

func (self Cacher) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	ser, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return self.c.Set(ctx, key, ser, ttl)
}

func (self Cacher) Get(ctx context.Context, key string, result any) error {
	data, err := self.c.Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, result)
}

func (self Cacher) Del(ctx context.Context, key ...string) error {
	return self.c.Del(ctx, key...)
}

type RedisCacheImpl struct {
	rdb *redis.Client
}

func NewRedis(rdb *redis.Client) RedisCacheImpl {
	return RedisCacheImpl{
		rdb: rdb,
	}
}

func (self RedisCacheImpl) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if _, err := self.rdb.Set(ctx, key, value, ttl).Result(); err != nil {
		return err
	}
	return nil
}

func (self RedisCacheImpl) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := self.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func (self RedisCacheImpl) Del(ctx context.Context, keys ...string) error {
	if _, err := self.rdb.Del(ctx, keys...).Result(); err != nil {
		return err
	}
	return nil
}
