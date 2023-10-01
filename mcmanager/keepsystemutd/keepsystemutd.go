package keepsystemutd

import (
	"github.com/kofuk/premises/mcmanager/systemutil"
)

func KeepSystemUpToDate() {
	systemutil.Cmd("apt-get", []string{"update", "-y"}, []string{"DEBIAN_FRONTEND=noninteractive"})
	systemutil.Cmd("apt-get", []string{"upgrade", "-y"}, []string{"DEBIAN_FRONTEND=noninteractive"})
	systemutil.Cmd("apt-get", []string{"autoremove", "-y", "--purge"}, []string{"DEBIAN_FRONTEND=noninteractive"})
}
