#!/usr/bin/env bash

if [ ! -e /userdata ]; then
    exit 1
fi

### Temporary (we should do this in exteriord in the future)
mkdir -p /opt/premises/servers.d/../gamedata/../bin
fallocate --posix --length 256M /opt/premises/gamedata.img
mkfs.btrfs /opt/premises/gamedata.img
mount "/opt/premises/gamedata.img" "/mnt"
chown -R 1000:1000 "/mnt"
umount /mnt
###

cat /userdata | base64 -d >/userdata_decoded.sh
chmod +x /userdata_decoded.sh

/userdata_decoded.sh

### Temporary
/opt/premises/bin/exteriord &>/exteriord.log
###

sleep inf
