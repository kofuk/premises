package systemutil

import (
	"errors"
	"math"
	"os/user"
	"strconv"
)

func GetAppUserID() (int, int, error) {
	user, err := user.Lookup("premises")
	if err != nil {
		return 0, 0, err
	}
	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		return 0, 0, err
	}
	gid, err := strconv.Atoi(user.Gid)
	if err != nil {
		return 0, 0, err
	}

	if (uid < 0 || math.MaxInt32 < uid) || (gid < 0 || math.MaxInt32 < gid) {
		// Check invalid uid and gid
		return 0, 0, errors.New("uid and gid must be in [0..MaxInt32]")
	}

	return uid, gid, nil
}
