package keepsystemutd

import (
	"github.com/kofuk/premises/mcmanager/systemutil"
)

func KeepSystemUpToDate() {
	systemutil.AptGet("update", "-y")
	systemutil.AptGet("upgrade", "-y")
	systemutil.AptGet("autoremove", "-y", "--purge")
}
