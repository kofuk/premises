package logging

import (
	"fmt"
	"math/rand"
	"time"
)

func Log(topic, level, message string) {
	time.Sleep((time.Millisecond * time.Duration(rand.Intn(256))) << 1)

	fmt.Printf("[%s] [%s/%s]: %s\n", time.Now().Format(time.TimeOnly), topic, level, message)
}
