package types

type SnapshotInput struct {
	Slot int `json:"slot"`
}

type SnapshotOutput struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}
