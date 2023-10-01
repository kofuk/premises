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
	for _, env := range envs {
		cmd.Env = append(cmd.Env, env)
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
