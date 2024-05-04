package types

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
