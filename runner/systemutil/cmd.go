package systemutil

import (
	"log"
	"os"
	"os/exec"
)

func Cmd(cmdPath string, args []string, envs []string) error {
	log.Println("Command:", cmdPath, args)

	cmd := exec.Command(cmdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(cmd.Environ(), envs...)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func AptGet(args ...string) error {
	if err := Cmd("apt-get", args, []string{"DEBIAN_FRONTEND=noninteractive"}); err == nil {
		return nil
	}
	Cmd("dpkg", []string{"--configure", "-a"}, []string{"DEBIAN_FRONTEND=noninteractive"})
	return Cmd("apt-get", args, []string{"DEBIAN_FRONTEND=noninteractive"})
}
