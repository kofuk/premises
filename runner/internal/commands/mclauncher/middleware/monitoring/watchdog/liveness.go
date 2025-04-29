package watchdog

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
)

type LivenessWatchdog struct {
	addr       string
	dialer     net.Dialer
	prevOnline bool
}

var _ Watchdog = (*LivenessWatchdog)(nil)

func NewLivenessWatchdog(optionalAddr ...string) *LivenessWatchdog {
	addr := "127.0.0.2:32109"
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

func (l *LivenessWatchdog) Check(c core.LauncherContext, watchID int, status *Status) error {
	if l.prevOnline && watchID%30 != 0 {
		// Assume that the server's liveness is not changing
		status.Online = l.prevOnline
		return nil
	}

	conn, err := l.dialer.DialContext(c.Context(), "tcp", l.addr)
	if err != nil {
		slog.Debug(fmt.Sprintf("Server is not healthy: %v", err))
	} else {
		conn.Close()
		slog.Debug("Server is healthy")
	}

	online := err == nil
	status.Online = online
	l.prevOnline = online

	return nil
}
