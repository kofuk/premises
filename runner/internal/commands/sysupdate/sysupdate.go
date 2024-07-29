package sysupdate

import (
	"github.com/kofuk/premises/runner/internal/system"
)

func Run(args []string) int {
	system.AptGet("upgrade", "-y")
	system.AptGet("autoremove", "-y", "--purge")
	return 0
}
