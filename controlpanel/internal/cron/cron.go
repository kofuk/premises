package cron

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"time"

	"github.com/kofuk/premises/controlpanel/internal/config"
	conohaClient "github.com/kofuk/premises/controlpanel/internal/launcher/server/conoha/client"
	"github.com/kofuk/premises/controlpanel/internal/world"
	"github.com/robfig/cron/v3"
)

type CronService struct {
	conoha       *conohaClient.Client
	nameTag      string
	worldService *world.WorldService
}

func NewCronService(config *config.Config, worldService *world.WorldService) *CronService {
	identity := conohaClient.Identity{
		User:     config.ConohaUser,
		Password: config.ConohaPassword,
		TenantID: config.ConohaTenantID,
	}
	endpoints := conohaClient.Endpoints{
		Identity: config.ConohaIdentityService,
		Compute:  config.ConohaComputeService,
		Image:    config.ConohaImageService,
		Volume:   config.ConohaVolumeService,
	}
	conoha := conohaClient.NewClient(identity, endpoints, nil)

	return &CronService{
		conoha:       conoha,
		nameTag:      config.ConohaNameTag,
		worldService: worldService,
	}
}

func (cr *CronService) runSaveStorageJob() error {
	images, err := cr.conoha.ListImages(context.Background())
	if err != nil {
		return err
	}

	for _, image := range images.Images {
		if image.Name == cr.nameTag {
			// If the image already exists, delete it first.
			err := cr.conoha.DeleteImage(context.Background(), conohaClient.DeleteImageInput{
				ImageID: image.ID,
			})
			if err != nil {
				return err
			}
			break
		}
	}

	volumes, err := cr.conoha.ListVolumes(context.Background())
	if err != nil {
		return err
	}

	var volume *conohaClient.Volume
	for _, v := range volumes.Volumes {
		if v.Name == cr.nameTag {
			volume = &v
			break
		}
	}

	if volume == nil {
		return errors.New("volume not found")
	}

	err = cr.conoha.SaveVolumeImage(context.Background(), conohaClient.SaveVolumeImageInput{
		VolumeID:  volume.ID,
		ImageName: cr.nameTag,
	})
	if err != nil {
		return err
	}

	return nil
}

func (cr *CronService) runCreateStorageJob() error {
	volumes, err := cr.conoha.ListVolumes(context.Background())
	if err != nil {
		return err
	}
	for _, v := range volumes.Volumes {
		if v.Name == cr.nameTag {
			slog.Info("Volume already exists. Skip creating a new volume.")
			return nil
		}
	}

	images, err := cr.conoha.ListImages(context.Background())
	if err != nil {
		return err
	}

	var image *conohaClient.Image
	for _, i := range images.Images {
		if i.Name == cr.nameTag {
			image = &i
			break
		}
	}

	if image == nil {
		return errors.New("image not found")
	} else if image.Status != "active" {
		return errors.New("image is not active")
	}

	_, err = cr.conoha.CreateBootVolume(context.Background(), conohaClient.CreateBootVolumeInput{
		ImageID: image.ID,
		Name:    cr.nameTag,
	})
	if err != nil {
		return err
	}

	return nil
}

func (cr *CronService) runPruneWorldsJob() error {
	return cr.worldService.Prune(context.Background(), 3)
}

func withDelay(fn func() error) func() {
	return func() {
		time.Sleep(time.Duration(rand.Intn(10)) * time.Minute)
		if err := fn(); err != nil {
			slog.Error("cron job failed", slog.Any("error", err))
		}
	}
}

func (cr *CronService) Run(ctx context.Context) error {
	c := cron.New(cron.WithLocation(time.UTC))

	// 23:30 on Sunday (JST)
	// Save existing storage data to image service.
	c.AddFunc("30 14 * * 0", withDelay(cr.runSaveStorageJob))

	// Every 1 hour.
	// Create a new storage using saved image.
	c.AddFunc("45 * * * *", withDelay(cr.runCreateStorageJob))

	// Every day at 04:00 (JST)
	// Prune old worlds
	c.AddFunc("0 19 * * *", withDelay(cr.runPruneWorldsJob))

	c.Start()

	<-ctx.Done()
	<-c.Stop().Done()
	return nil
}
