#!/usr/bin/env bash
set -eu

new_version="$1"

cd "$(git rev-parse --show-toplevel)"

(
    cd internal

    sed -i "s@^\\(const Version = \"\\)[0-9]\\+\\.[0-9]\\+\\.[0-9]\\+\\(\"\\)\$@\\1${new_version}\\2@" version.go
)

(
    sed -i "s@^\\( \\+image: ghcr.io/kofuk/premises:\\)[0-9]\\+\\.[0-9]\\+\\.[0-9]\\+\$@\\1${new_version}@" compose.yaml
)

(
    cd charts/premises

    sed -i "s/^\\(version: \\|appVersion: \"\\)[0-9]\\+\\.[0-9]\\+\\.[0-9]\\+\\(\"\\?\\)\$/\\1${new_version}\\2/;" Chart.yaml
    sed -i "s@^\\(image: ghcr.io/kofuk/premises:\\)[0-9]\\+\\.[0-9]\\+\\.[0-9]\\+\$@\\1${new_version}@" values.yaml
)

(
    cd controlpanel/frontend

    sed -i "s/\"version\": \"[0-9]\\+\\.[0-9]\\+\\.[0-9]\\+\",\$/\"version\": \"${new_version}\",/" package.json
    pnpm install
)
