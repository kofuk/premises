package keepsystemutd

import (
	"github.com/kofuk/premises/runner/systemutil"
)

func KeepSystemUpToDate() {
	systemutil.AptGet("update", "-y")
	systemutil.AptGet("upgrade", "-y")
	systemutil.AptGet("autoremove", "-y", "--purge")
}
