package keepsystemutd

import (
	"github.com/kofuk/premises/runner/systemutil"
)

func KeepSystemUpToDate(args []string) {
	systemutil.AptGet("update", "-y")
	systemutil.AptGet("upgrade", "-y")
	systemutil.AptGet("autoremove", "-y", "--purge")
}
