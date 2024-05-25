#!/usr/bin/env bash
set -euxo pipefail

# Create .env if not exists
[ -e controlpanel/.env ] || cp .devcontainer/env controlpanel/.env

# Install npm dependencies
( cd /workspace/premises/controlpanel && npm install ) &
p1=$!

./.devcontainer/fake-runner/build_base_image.sh &
p2=$!

(
    eval $(sed 's/^/export /' controlpanel/.env)

    ( cd controlpanel; PREMISES_MODE=web go run . migrate )
    ( cd pmctl; go run . user add -u user1 -p password1 --initialized )
) &
p3=$!

# Forward minio:9000 and minio:9001 in the background
lighttpd -f .devcontainer/lighttpd.conf

wait "${p1}"
wait "${p2}"
wait "${p3}"
