package runner

import (
	"github.com/kofuk/premises/backend/common/entity"
)

type EventType string

const (
	EventHello   EventType = "hello"
	EventStatus  EventType = "status"
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
	Info     *InfoExtra    `json:"info,omitempty"`
	Started  *StartedExtra `json:"started,omitempty"`
}

type ActionType string

func (a ActionType) String() string {
	return string(a)
}

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
