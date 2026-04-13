#!/usr/bin/env bash
set -euo pipefail

for path in $(find -name go.work -or -name go.work.sum -or -name go.mod -or -name go.sum -printf "%P\n" | sort); do
    echo "--mount=type=bind,source=${path},target=${path}"
done

