package jobseq

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	runnerEntity "github.com/kofuk/premises/common/entity/runner"
	"github.com/stretchr/testify/assert"
)

func registerPushStatusResponder(t *testing.T, sentEvents *[]runnerEntity.Event) {
	httpmock.RegisterResponder(http.MethodPost, "http://127.0.0.1:2000/pushstatus",
		func(r *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)

			var msg runnerEntity.Message
			assert.NoError(t, json.Unmarshal(body, &msg))

			assert.Equal(t, "serverStatus", msg.Type)

			var event runnerEntity.Event
			assert.NoError(t, json.Unmarshal([]byte(msg.UserData), &event))

			*sentEvents = append(*sentEvents, event)

			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		},
	)
}

func Test_JobSequence_allJobsSuccess(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var sentEvents []runnerEntity.Event
	registerPushStatusResponder(t, &sentEvents)

	jobseq := NewJobSequence(
		JobSequenceConfig{},
		NewSimpleJob(runnerEntity.EventWorldDownload, func(state *State, progress chan int) JobResult {
			progress <- 10
			progress <- 70

			return JobResult{
				Success: true,
			}
		}),
		NewSimpleJob(runnerEntity.EventLoading, func(state *State, progress chan int) JobResult {
			return JobResult{
				Success: true,
			}
		}),
	)

	state := NewState()
	jobseq.Run(state)

	expectedSentEvents := []runnerEntity.Event{
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventWorldDownload,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventWorldDownload,
				Progress:  10,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventWorldDownload,
				Progress:  70,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventLoading,
			},
		},
	}

	assert.Equal(t, expectedSentEvents, sentEvents)
}

func Test_JobSequence_updated(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var sentEvents []runnerEntity.Event
	registerPushStatusResponder(t, &sentEvents)

	attempt := 0

	jobseq := NewJobSequence(
		JobSequenceConfig{},
		NewSimpleJob(runnerEntity.EventGameDownload, func(state *State, progress chan int) JobResult {
			attempt++

			return JobResult{
				Success: true,
			}
		}),
		NewSimpleJob(runnerEntity.EventWorldDownload, func(state *State, progress chan int) JobResult {
			if attempt >= 2 {
				return JobResult{
					Success: true,
				}
			}

			return JobResult{
				Success:          false,
				FailureLevel:     NeedsSettingsUpdate,
				FailureEventCode: runnerEntity.EventWorldErr,
			}
		}),
		NewSimpleJob(runnerEntity.EventLoading, func(state *State, progress chan int) JobResult {
			return JobResult{
				Success: true,
			}
		}),
	)

	state := NewState(WaitForUpdateFunction(func(state *State) {}))
	jobseq.Run(state)

	expectedSentEvents := []runnerEntity.Event{
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventGameDownload,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventWorldDownload,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventWorldErr,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventGameDownload,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventWorldDownload,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventLoading,
			},
		},
	}

	assert.Equal(t, expectedSentEvents, sentEvents)
}

func Test_JobSequence_confirmed(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var sentEvents []runnerEntity.Event
	registerPushStatusResponder(t, &sentEvents)

	jobseq := NewJobSequence(
		JobSequenceConfig{},
		NewSimpleJob(runnerEntity.EventGameDownload, func(state *State, progress chan int) JobResult {
			return JobResult{
				Success: true,
			}
		}),
		NewSimpleJob(runnerEntity.EventWorldDownload, func(state *State, progress chan int) JobResult {
			return JobResult{
				Success:          false,
				FailureLevel:     NeedsConfirmation,
				FailureEventCode: runnerEntity.EventWorldErr,
			}
		}),
		NewSimpleJob(runnerEntity.EventLoading, func(state *State, progress chan int) JobResult {
			return JobResult{
				Success: true,
			}
		}),
	)

	confirmCount := 0
	state := NewState(WaitForConfirmationFunction(func(ctx context.Context, state *State) bool {
		confirmCount++
		// Update (instead of confirm) on first confirmation
		return !(confirmCount == 1)
	}))
	jobseq.Run(state)

	expectedSentEvents := []runnerEntity.Event{
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventGameDownload,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventWorldDownload,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventWorldErr,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventGameDownload,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventWorldDownload,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventWorldErr,
			},
		},
		{
			Type: runnerEntity.EventStatus,
			Status: &runnerEntity.StatusExtra{
				EventCode: runnerEntity.EventLoading,
			},
		},
	}

	assert.Equal(t, expectedSentEvents, sentEvents)
}
