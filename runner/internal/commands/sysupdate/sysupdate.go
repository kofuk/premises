package sysupdate

import (
	"context"

	"github.com/kofuk/premises/runner/internal/system"
)

func Run(ctx context.Context, args []string) int {
	system.AptGet("upgrade", "-y")
	system.AptGet("autoremove", "-y", "--purge")
	return 0
}
