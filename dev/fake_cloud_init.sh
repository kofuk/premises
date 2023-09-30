#!/usr/bin/env bash

if [ ! -e /userdata ]; then
    exit 1
fi

### Temporary (we should do this in exteriord in the future)
mount "${PREMISES_BASEDIR}/gamedata.img" "/mnt"
chown -R premises:premises "/mnt"
umount /mnt
###

cat /userdata | base64 -d >/userdata_decoded.sh
chmod +x /userdata_decoded.sh

/userdata_decoded.sh

### Temporary
/exteriord
###

sleep inf
