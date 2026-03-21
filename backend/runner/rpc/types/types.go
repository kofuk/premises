package types

import "github.com/kofuk/premises/backend/common/entity/runner"

type SnapshotHelperInput struct {
	Slot int `json:"slot"`
}

type SnapshotHelperOutput struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type SnapshotInput struct {
	Slot  int `json:"slot"`
	Actor int `json:"actor"`
}

type StateSetInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type StateGetInput struct {
	Key string `json:"key"`
}

type StateRemoveInput struct {
	Key string `json:"key"`
}

type EventInput struct {
	Dispatch bool         `json:"dispatch"`
	Event    runner.Event `json:"event"`
}

type RegisterMeterTargetInput struct {
	Pid int `json:"pid"`
}

type UnregisterMeterTargetInput struct {
	Pid int `json:"pid"`
}
