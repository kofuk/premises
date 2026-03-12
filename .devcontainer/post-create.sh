#!/usr/bin/env bash
set -euxo pipefail

mise trust
mise install
eval "$(mise activate bash)"

echo 'eval "$(mise activate bash)"' >~/.bashrc

# Create .env if not exists
[ -e backend/services/monolith/.env ] || cp .devcontainer/env backend/services/monolith/.env

# Install npm dependencies
(
    cd frontend && pnpm install
) &
p1=$!

./.devcontainer/fake-runner/build_base_image.sh &
p2=$!

(
    eval $(grep -F = backend/services/monolith/.env | sed 's/^/export /')

    ( cd backend/services/monolith; PREMISES_MODE=web go run . migrate )
    ( cd backend/services/pmctl; go run . user add -u admin -p password --initialized )
) &
p3=$!

wait "${p1}"
wait "${p2}"
wait "${p3}"
