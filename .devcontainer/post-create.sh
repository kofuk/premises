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
    ( cd controlpanel/pmctl; go run . user add -u user1 -p password1 --initialized )
) &
p3=$!

wait "${p1}"
wait "${p2}"
wait "${p3}"
