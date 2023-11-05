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
	EventShutdown      EventCode = iota + 1 // 1
	EventSysInit                            // 2
	EventGameDownload                       // 3
	EventWorldDownload                      // 4
	EventWorldPrepare                       // 5
	EventWorldUpload                        // 6
	EventLoading                            // 7
	EventRunning                            // 8
	EventStopping                           // 9
	EventCrashed                            // 10
	EventClean                              // 11

	// Event codes below should be provided UI to retry.
	EventGameErr   // 12
	EventWorldErr  // 13
	EventLaunchErr // 14
)

type InfoCode int

const (
	InfoSnapshotDone  InfoCode = iota + 1 // 1
	InfoSnapshotError                     // 2
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
