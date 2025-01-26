package runner

import (
	"github.com/kofuk/premises/internal/entity"
)

type EventType string

const (
	EventHello   EventType = "hello"
	EventStatus  EventType = "status"
	EventSysstat EventType = "sysstat"
	EventInfo    EventType = "info"
	EventStarted EventType = "started"
)

func (ev EventType) String() string {
	return string(ev)
}

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
	Actor    int             `json:"actor"`
	IsError  bool            `json:"isError"`
}

type HelloExtra struct {
	Version string `json:"version"`
	Host    string `json:"host"`
	Addr    struct {
		IPv4 []string `json:"ipv4"`
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

type RequestMeta struct {
	Traceparent string `json:"traceparent"`
}

type Event struct {
	Type     EventType     `json:"type"`
	Metadata RequestMeta   `json:"metadata"`
	Hello    *HelloExtra   `json:"hello,omitempty"`
	Status   *StatusExtra  `json:"status,omitempty"`
	Sysstat  *SysstatExtra `json:"sysstat,omitempty"`
	Info     *InfoExtra    `json:"info,omitempty"`
	Started  *StartedExtra `json:"started,omitempty"`
}

type ActionType string

const (
	ActionStop        ActionType = "stop"
	ActionSnapshot    ActionType = "snapshot"
	ActionUndo        ActionType = "undo"
	ActionReconfigure ActionType = "reconfigure"
	ActionConnReq     ActionType = "connectionRequest"
)

type SnapshotConfig struct {
	Slot int `json:"slot"`
}

type ConnReqInfo struct {
	ConnectionID string `json:"connectionId"`
	Endpoint     string `json:"endpoint"`
	ServerCert   string `json:"serverCert"`
}

type Action struct {
	Type     ActionType      `json:"type"`
	Actor    int             `json:"actor"`
	Metadata RequestMeta     `json:"metadata"`
	Config   *GameConfig     `json:"config,omitempty"`
	Snapshot *SnapshotConfig `json:"snapshot,omitempty"`
	ConnReq  *ConnReqInfo    `json:"connectionRequestInfo,omitempty"`
}
