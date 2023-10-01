#!/usr/bin/env bash

if [ ! -e /userdata ]; then
    exit 1
fi

cat /userdata | base64 -d >/userdata_decoded.sh
chmod +x /userdata_decoded.sh

/userdata_decoded.sh

sleep inf
