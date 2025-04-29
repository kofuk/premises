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
			Timeout: time.Second * 3,
		},
	}
}

func (l *LivenessWatchdog) Name() string {
	return "LivenessWatchdog"
}

func (l *LivenessWatchdog) Check(c core.LauncherContext, watchID int, status *Status) error {
	if l.prevOnline {
		// If the previous check was successful, we perform a check every 10 seconds.
		if watchID%10 != 0 {
			status.Online = l.prevOnline
			return nil
		}
	} else {
		// If the previous check was failed, we perform a check every 3 second.
		if watchID%3 != 0 {
			status.Online = l.prevOnline
			return nil
		}
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
