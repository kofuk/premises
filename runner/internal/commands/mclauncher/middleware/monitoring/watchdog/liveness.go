package watchdog

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"
)

type LivenessWatchdog struct {
	addr       string
	dialer     net.Dialer
	prevOnline bool
}

var _ Watchdog = (*LivenessWatchdog)(nil)

func NewLivenessWatchdog(optionalAddr ...string) *LivenessWatchdog {
	addr := "127.0.0.1.32109"
	if len(optionalAddr) > 0 {
		addr = optionalAddr[0]
	}

	return &LivenessWatchdog{
		addr: addr,
		dialer: net.Dialer{
			Timeout: time.Second * 5,
		},
	}
}

func (l *LivenessWatchdog) Name() string {
	return "LivenessWatchdog"
}

func (l *LivenessWatchdog) Check(ctx context.Context, watchID int, status *Status) error {
	if l.prevOnline && watchID%30 != 0 {
		return nil
	}

	conn, err := l.dialer.DialContext(ctx, "tcp", l.addr)
	if err != nil {
		slog.Debug(fmt.Sprintf("Server is not healthy: %v", err))
	} else {
		conn.Close()
	}

	online := err == nil
	status.Online = online
	l.prevOnline = online

	return nil
}
