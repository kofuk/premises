package runner

type Config struct {
	AuthKey string `json:"authKey"`
	Server  struct {
		PreferDetected   bool     `json:"preferDetected"`
		Version          string   `json:"name"`
		DownloadUrl      string   `json:"downloadUrl"`
		ManifestOverride string   `json:"manifestOverride"`
		CustomCommand    []string `json:"customCommand"`
	} `json:"server"`
	World struct {
		ShouldGenerate bool   `json:"shouldGenerate"`
		Name           string `json:"name"`
		GenerationId   string `json:"generationId"`
		Seed           string `json:"seed"`
		LevelType      string `json:"levelType"`
		Difficulty     string `json:"difficulty"`
	} `json:"world"`
	Motd      string   `json:"motd"`
	Operators []string `json:"operators"`
	Whitelist []string `json:"whitelist"`
	AWS       struct {
		AccessKey string
		SecretKey string
	} `json:"aws"`
	S3 struct {
		Endpoint string `json:"endpoint"`
		Bucket   string `json:"bucket"`
	} `json:"s3"`
	ControlPanel string `json:"controlPanel"`
}
