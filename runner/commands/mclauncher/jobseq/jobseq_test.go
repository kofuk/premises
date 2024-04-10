package jobseq

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/commands/exteriord/msgrouter"
	"github.com/stretchr/testify/assert"
)

func registerPushStatusResponder(t *testing.T, sentEvents *[]runner.Event) {
	httpmock.RegisterResponder(http.MethodPost, "http://127.0.0.1:2000/pushstatus",
		func(r *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)

			var msg msgrouter.Message
			assert.NoError(t, json.Unmarshal(body, &msg))

			assert.Equal(t, "serverStatus", msg.Type)

			var event runner.Event
			assert.NoError(t, json.Unmarshal([]byte(msg.UserData), &event))

			*sentEvents = append(*sentEvents, event)

			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		},
	)
}

func Test_JobSequence_allJobsSuccess(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var sentEvents []runner.Event
	registerPushStatusResponder(t, &sentEvents)

	jobseq := NewJobSequence(
		JobSequenceConfig{},
		NewSimpleJob(entity.EventWorldDownload, func(state *State, progress chan int) JobResult {
			progress <- 10
			progress <- 70

			return JobResult{
				Success: true,
			}
		}),
		NewSimpleJob(entity.EventLoading, func(state *State, progress chan int) JobResult {
			return JobResult{
				Success: true,
			}
		}),
	)

	state := NewState()
	jobseq.Run(state)

	expectedSentEvents := []runner.Event{
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldDownload,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldDownload,
				Progress:  10,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldDownload,
				Progress:  70,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventLoading,
			},
		},
	}

	assert.Equal(t, expectedSentEvents, sentEvents)
}

func Test_JobSequence_updated(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var sentEvents []runner.Event
	registerPushStatusResponder(t, &sentEvents)

	attempt := 0

	jobseq := NewJobSequence(
		JobSequenceConfig{},
		NewSimpleJob(entity.EventGameDownload, func(state *State, progress chan int) JobResult {
			attempt++

			return JobResult{
				Success: true,
			}
		}),
		NewSimpleJob(entity.EventWorldDownload, func(state *State, progress chan int) JobResult {
			if attempt >= 2 {
				return JobResult{
					Success: true,
				}
			}

			return JobResult{
				Success:          false,
				FailureLevel:     NeedsSettingsUpdate,
				FailureEventCode: entity.EventWorldErr,
			}
		}),
		NewSimpleJob(entity.EventLoading, func(state *State, progress chan int) JobResult {
			return JobResult{
				Success: true,
			}
		}),
	)

	state := NewState(WaitForUpdateFunction(func(state *State) {}))
	jobseq.Run(state)

	expectedSentEvents := []runner.Event{
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventGameDownload,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldDownload,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventGameDownload,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldDownload,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventLoading,
			},
		},
	}

	assert.Equal(t, expectedSentEvents, sentEvents)
}

func Test_JobSequence_confirmed(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	var sentEvents []runner.Event
	registerPushStatusResponder(t, &sentEvents)

	jobseq := NewJobSequence(
		JobSequenceConfig{},
		NewSimpleJob(entity.EventGameDownload, func(state *State, progress chan int) JobResult {
			return JobResult{
				Success: true,
			}
		}),
		NewSimpleJob(entity.EventWorldDownload, func(state *State, progress chan int) JobResult {
			return JobResult{
				Success:          false,
				FailureLevel:     NeedsConfirmation,
				FailureEventCode: entity.EventWorldErr,
			}
		}),
		NewSimpleJob(entity.EventLoading, func(state *State, progress chan int) JobResult {
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

	expectedSentEvents := []runner.Event{
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventGameDownload,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldDownload,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventGameDownload,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldDownload,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		},
		{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventLoading,
			},
		},
	}

	assert.Equal(t, expectedSentEvents, sentEvents)
}
