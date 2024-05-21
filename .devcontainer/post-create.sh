#!/usr/bin/env bash
(
    # Create cache for go modules to speed up builds
    cd /workspace/premises/controlpanel && go build -o /dev/null .
    cd /workspace/premises/runner && go build -o /dev/null .

    # Install npm dependencies
    cd /workspace/premises/controlpanel && npm install
)

./dev/build_base_image.sh
./dev/launch_all.sh

# TODO: Create a user
