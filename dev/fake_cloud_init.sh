#!/usr/bin/env bash

if [ ! -e /userdata ]; then
    exit 1
fi

{
    # XXX  Create loop[0-3] device node so that mounting image don't fail.
    loop_devmajor=$(grep loop /proc/devices | cut -c3)
    for i in 0 1 2 3; do
        mknod "/dev/loop${i}" b "${loop_devmajor}" "${i}"
    done
}

cat /userdata | base64 -d >/userdata_decoded.sh
chmod +x /userdata_decoded.sh

/userdata_decoded.sh

sleep inf
