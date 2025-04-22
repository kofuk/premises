package repository

import (
	"context"
	"encoding/json"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/rpc"
	"github.com/kofuk/premises/runner/internal/rpc/types"
)

type ExteriorStateRepository struct {
	rpcClient *rpc.Client
}

var _ core.StateRepository = (*ExteriorStateRepository)(nil)

func NewExteriorStateRepository(rpcClient *rpc.Client) *ExteriorStateRepository {
	return &ExteriorStateRepository{
		rpcClient: rpcClient,
	}
}

func (r *ExteriorStateRepository) SetState(ctx context.Context, key string, state string) error {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return r.rpcClient.Call(ctx, "state/save", types.StateSetInput{
		Key:   key,
		Value: string(stateJSON),
	}, nil)
}

func (r *ExteriorStateRepository) RemoveState(ctx context.Context, key string) error {
	return r.rpcClient.Call(ctx, "state/remove", types.StateRemoveInput{
		Key: key,
	}, nil)
}

func (r *ExteriorStateRepository) GetState(ctx context.Context, key string) (string, error) {
	var state string
	err := r.rpcClient.Call(ctx, "state/get", types.StateGetInput{
		Key: key,
	}, &state)
	if err != nil {
		return "", err
	}
	return state, nil
}
