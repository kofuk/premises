#!/usr/bin/env bash
(
    # Create cache for go modules to speed up builds
    cd /workspace/premises/controlpanel && go build -o /dev/null .
    cd /workspace/premises/runner && go build -o /dev/null .

    # Install npm dependencies
    cd /workspace/premises/controlpanel && npm install
)

# Create .env if not exists
[ -e controlpanel/.env ] || cp .devcontainer/env controlpanel/.env

./.devcontainer/fake-runner/build_base_image.sh
./.devcontainer/launch_all.sh

# TODO: Create a user
