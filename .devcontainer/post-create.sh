#!/usr/bin/env bash
(
    # Install npm dependencies
    cd /workspace/premises/controlpanel && npm install
)

# Create .env if not exists
[ -e controlpanel/.env ] || cp .devcontainer/env controlpanel/.env

./.devcontainer/fake-runner/build_base_image.sh

# TODO: Create a user
