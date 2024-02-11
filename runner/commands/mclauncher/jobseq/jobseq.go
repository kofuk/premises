package jobseq

import (
	"context"
	"sync"

	"github.com/kofuk/premises/common/entity"
	runnerEntity "github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/exterior"
	log "github.com/sirupsen/logrus"
)

type FailureLevel int

const (
	NeedsConfirmation FailureLevel = iota
	NeedsSettingsUpdate
)

type JobResult struct {
	Success          bool
	FailureLevel     FailureLevel
	FailureEventCode entity.EventCode
}

type Job interface {
	GetEventCode() entity.EventCode
	Execute(state *State, progress chan int) JobResult
}

type SimpleJob struct {
	eventCode entity.EventCode
	run       func(state *State, progress chan int) JobResult
}

func NewSimpleJob(eventCode entity.EventCode, run func(state *State, progress chan int) JobResult) Job {
	return &SimpleJob{
		eventCode: eventCode,
		run:       run,
	}
}

func (self *SimpleJob) GetEventCode() entity.EventCode {
	return self.eventCode
}

func (self *SimpleJob) Execute(state *State, progress chan int) JobResult {
	return self.run(state, progress)
}

type JobSequenceConfig struct {
	ConfirmContext func() context.Context
}

type JobSequence struct {
	config JobSequenceConfig
	jobs   []Job
}

func NewJobSequence(config JobSequenceConfig, jobs ...Job) *JobSequence {
	return &JobSequence{
		config: config,
		jobs:   jobs,
	}
}

func (self *JobSequence) runJob(state *State, job Job) bool {
	reportProgress := make(chan int)

	eventCode := job.GetEventCode()

	if eventCode != runnerEntity.NoEvent {
		if err := exterior.SendMessage("serverStatus", runnerEntity.Event{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: eventCode,
			},
		}); err != nil {
			log.WithError(err).Error("Unable to send message")
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		for progress := range reportProgress {
			if eventCode == runnerEntity.NoEvent {
				continue
			}

			if err := exterior.SendMessage("serverStatus", runnerEntity.Event{
				Type: runnerEntity.EventStatus,
				Status: &runnerEntity.StatusExtra{
					EventCode: eventCode,
					Progress:  progress,
				},
			}); err != nil {
				log.WithError(err).Error("Unable to send message")
			}
		}
	}()

	result := job.Execute(state, reportProgress)

	close(reportProgress)

	// We want to make sure all progress events are sent.
	wg.Wait()

	if result.Success {
		return true
	}

	if result.FailureEventCode != runnerEntity.NoEvent {
		if err := exterior.SendMessage("serverStatus", runnerEntity.Event{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: result.FailureEventCode,
			},
		}); err != nil {
			log.WithError(err).Error("Unable to send message")
		}
	}

	switch result.FailureLevel {
	case NeedsConfirmation:
		ctx := context.Background()
		if self.config.ConfirmContext != nil {
			ctx = self.config.ConfirmContext()
		}

		return state.WaitForConfirmation(ctx)

	case NeedsSettingsUpdate:
		state.WaitForUpdate()
		return false
	}

	return true
}

func (self *JobSequence) Run(state *State) {
out:
	for _, job := range self.jobs {
		if !self.runJob(state, job) {
			goto out
		}
	}
}
