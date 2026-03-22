package sysupdate

import (
	"context"

	"github.com/kofuk/premises/backend/common/entity/runner"
	"github.com/kofuk/premises/backend/runner/system"
)

func Run(ctx context.Context, config *runner.Config, args []string) int {
	system.AptGet(ctx, "upgrade", "-y")
	system.AptGet(ctx, "autoremove", "-y", "--purge")
	return 0
}
