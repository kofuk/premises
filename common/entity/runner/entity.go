package runner

import (
	"github.com/kofuk/premises/common/entity"
)

type Message struct {
	Type     string `json:"type"`
	UserData string `json:"user_data"`
}

type EventType string

const (
	EventHello   EventType = "hello"
	EventStatus            = "status"
	EventSysstat           = "sysstat"
	EventInfo              = "info"
	EventStarted           = "started"
)

const (
	EventShutdown      entity.EventCode = iota + 1 // 1
	EventSysInit                                   // 2
	EventGameDownload                              // 3
	EventWorldDownload                             // 4
	EventWorldPrepare                              // 5
	EventWorldUpload                               // 6
	EventLoading                                   // 7
	EventRunning                                   // 8
	EventStopping                                  // 9
	EventCrashed                                   // 10
	EventClean                                     // 11
)

// Event codes that should be provided UI to retry.
const (
	EventGameErr   entity.EventCode = iota + 50 // 50
	EventWorldErr                               // 51
	EventLaunchErr                              // 52
)

const (
	InfoSnapshotDone  entity.InfoCode = iota + 1 // 1
	InfoSnapshotError                            // 2
	InfoNoSnapshot                               // 3
)

type StatusExtra struct {
	EventCode entity.EventCode `json:"eventCode"`
	Progress  int              `json:"progress"`
}

type SysstatExtra struct {
	CPUUsage float64 `json:"cpuUsage"`
	Time     int64   `json:"time"`
}

type InfoExtra struct {
	InfoCode entity.InfoCode `json:"infoCode"`
	IsError  bool            `json:"isError"`
}

type HelloExtra struct {
	Version string `json:"version"`
	Host    string `json:"host"`
	Addr    struct {
		IPv4 []string  `json:"ipv4"`
		IPv6 []string `json:"ipv6,omitempty"`
	} `json:"addr"`
}

type StartedExtra struct {
	ServerVersion string `json:"serverVersion"`
	World         struct {
		Name string `json:"name"`
		Seed string `json:"seed"`
	} `json:"world"`
}

type Event struct {
	Type    EventType     `json:"type"`
	Hello   *HelloExtra   `json:"hello,omitempty"`
	Status  *StatusExtra  `json:"status,omitempty"`
	Sysstat *SysstatExtra `json:"sysstat,omitempty"`
	Info    *InfoExtra    `json:"info,omitempty"`
	Started *StartedExtra `json:"started,omitempty"`
}

type ActionType string

const (
	ActionStop        ActionType = "stop"
	ActionSnapshot               = "snapshot"
	ActionUndo                   = "undo"
	ActionReconfigure            = "reconfigure"
)

type Action struct {
	Type   ActionType `json:"type"`
	Config *Config    `json:"config,omitempty"`
}
