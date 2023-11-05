package runner

type Message struct {
	Type     string `json:"type"`
	UserData string `json:"user_data"`
}

type EventType string

const (
	EventStatus  EventType = "status"
	EventSysstat EventType = "sysstat"
	EventInfo    EventType = "info"
)

type EventCode int

const (
	EventShutdown EventCode = iota + 1 // 1
	EventSysInit
	EventGameDownload
	EventGameErr
	EventWorldDownload
	EventWorldPrepare
	EventWorldUpload
	EventWorldErr
	EventLaunchErr
	EventLoading
	EventRunning
	EventStopping
	EventCrashed
	EventClean
)

type InfoCode int

const (
	InfoSnapshotDone InfoCode = iota + 1
	InfoSnapshotError
)

type StatusExtra struct {
	EventCode EventCode `json:"eventCode"`
	Progress  int       `json:"progress"`
	LegacyMsg string    `json:"message"`
}

type SysstatExtra struct {
	CPUUsage float64 `json:"cpuUsage"`
}

type InfoExtra struct {
	InfoCode  InfoCode `json:"infoCode"`
	LegacyMsg string   `json:"message"`
}

type Event struct {
	Type    EventType     `json:"type"`
	Status  *StatusExtra  `json:"status,omitempty"`
	Sysstat *SysstatExtra `json:"sysstat,omitempty"`
	Info    *InfoExtra    `json:"info,omitempty"`
}
