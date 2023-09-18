package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
)

const (
	SessionStateLoggedIn = "LoggedIn"

	SessionV2Prefix = "sessionv2"
)

type SessionV2 struct {
	State  string `json:"state"`
	UserID uint   `json:"user_id"`
}

func SaveSessionV2(redi *redis.Client, sessId string, state SessionV2) {
	data, _ := json.Marshal(state)

	if err := redi.Set(context.Background(), fmt.Sprintf("%s:%s", SessionV2Prefix, sessId), data, 30*24*time.Hour).Err(); err != nil {
		log.WithError(err).Error("Failed to store session v2")
		return
	}
}

func DiscardSessionV2(redi *redis.Client, sessId string) {
	if err := redi.Del(context.Background(), fmt.Sprintf("%s:%s", SessionV2Prefix, sessId)).Err(); err != nil {
		log.WithError(err).Error("Failed to store session v2")
		return
	}
}