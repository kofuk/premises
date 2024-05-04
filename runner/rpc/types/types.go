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
