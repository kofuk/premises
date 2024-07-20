package web

type GetLatestWorldIDResponse struct {
	WorldID string `json:"worldId"`
}

type CreateWorldDownloadURLRequest struct {
	WorldID string `json:"worldId"`
}

type CreateWorldDownloadURLResponse struct {
	URL string `json:"url"`
}

type CreateWorldUploadURLRequest struct {
	WorldName string `json:"worldName"`
}

type CreateWorldUploadURLResponse struct {
	URL string `json:"url"`
}
