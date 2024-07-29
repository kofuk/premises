package fs

import "os"

func RemoveIfExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := os.RemoveAll(path); err != nil {
		return err
	}

	return nil
}
