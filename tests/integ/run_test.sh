#!/usr/bin/env bash
set -euo pipefail

export TARGET_HOST="${TARGET_HOST:-http://localhost:8000}"
export USING_MCSERVER_FAKE="${USING_MCSERVER_FAKE:-no}"

cat <<EOF
Target Host:                 ${TARGET_HOST}
Using Fake Minecraft Server: ${USING_MCSERVER_FAKE}
EOF

dir="$(cd "$(dirname "${BASH_SOURCE:-0}")"; pwd)"

if ! deno run --check --allow-net --allow-env "${dir}/index.ts"; then
    container_id="$(docker container ls --filter label=org.kofuk.premises.managed --format '{{ .ID }}' | head -1)"
    echo '::group::Runner Log'
    if [ -n "${container_id}" ]; then
        echo "Container ID: ${container_id}"
        docker exec "${container_id}" cat /tmp/premises-startup.log
        docker exec "${container_id}" cat /exteriord.log
    else
        cat "$(ls /tmp/premises/exteriord-*.log | tail -1)"
    fi
    echo '::endgroup::'

    echo '::group::App Log'
    (
        cd "${dir}/.."
        if [ -n "$(docker compose ps -q)" ]; then
            docker compose logs web
        fi
    )
    echo '::endgroup::'

    exit 1
fi
