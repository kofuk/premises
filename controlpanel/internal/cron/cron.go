package cron

import (
	"context"
	"math/rand"
	"time"

	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/robfig/cron/v3"
)

type CronService struct{}

func (cr *CronService) runSaveStorageJob() {
}

func (cr *CronService) runCreateStorageJob() {
}

func withDelay(fn func()) func() {
	return func() {
		time.Sleep(time.Duration(rand.Intn(10)) * time.Minute)
		fn()
	}
}

func (cr *CronService) Run(ctx context.Context, config *config.Config) error {
	c := cron.New(cron.WithLocation(time.UTC))

	// 23:30 on Sunday (JST)
	// Save existing storage data to image service.
	c.AddFunc("30 14 * * 0", withDelay(cr.runSaveStorageJob))

	// 09:00 on Monday (JST)
	// Create a new storage using saved image.
	c.AddFunc("0 0 * * 1", withDelay(cr.runCreateStorageJob))

	c.Start()

	<-ctx.Done()
	<-c.Stop().Done()
	return nil
}
