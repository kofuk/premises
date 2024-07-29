package env

import "os"

func IsDevEnv() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}
