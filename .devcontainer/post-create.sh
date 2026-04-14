#!/usr/bin/env bash
set -euxo pipefail

mise trust
mise install

# Create .env if not exists
[ -e backend/ctrlplane/monolith/.env ] || cp .devcontainer/env backend/ctrlplane/monolith/.env

# Install npm dependencies
(
    cd frontend && pnpm install
) &
p1=$!

./etc/fake-runner/build_base_image.sh &
p2=$!

(
    eval $(grep -F = backend/ctrlplane/monolith/.env | sed 's/^/export /')

    ( cd backend/ctrlplane/monolith; PREMISES_MODE=web go run . migrate )
    ( cd backend/ctrlplane/pmctl; go run . user add -u admin -p password --initialized )
) &
p3=$!

wait "${p1}"
wait "${p2}"
wait "${p3}"
