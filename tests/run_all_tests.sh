#!/usr/bin/env bash
set -euo pipefail

specs=(
    'should_be_able_to_start_stop_server.ts'
)

export TARGET_HOST="${TARGET_HOST:-http://localhost:8000}"

cat <<EOF
Target Host: ${TARGET_HOST}
EOF

dir="$(cd "$(dirname "${BASH_SOURCE:-0}")"; pwd)"

for spec in "${specs[@]}"; do
    deno run --allow-net --allow-env "${dir}/specs/${spec}"
done
