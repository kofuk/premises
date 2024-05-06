#!/usr/bin/env bash

name="premises"
cd "$(dirname "${BASH_SOURCE:-0}")/.."

export DOCKER_API_VERSION="$(docker version --format '{{ .Server.APIVersion }}')"
export PREMISES_PROXY_BIND='127.0.0.1:25565'

mkdir /tmp/premises-world

make -C ./runner deploy-dev

exec tmux new-session -s "${name}" -n 'Middleware' "cd ./dev; docker compose -f compose.yaml -f tmpfs-minio.yaml up; bash" \; \
     new-window -n 'Redis Console' "cd ./dev; while true; do sleep 3; docker compose exec redis redis-cli; done" \; \
     new-window -n 'PostgreSQL Console' "cd ./dev; while true; do sleep 3; docker compose exec postgres psql -Upremises; done" \; \
     new-window -n 'ConoHa Emulator' "cd ./ostack-fake; go run .; bash" \; \
     new-window -n 'Frontend' "cd ./controlpanel; npm start; bash" \; \
     new-window -n 'Backend' "export PREMISES_MODE=web; cd ./controlpanel; sleep 3; go run .; bash" \; \
     new-window -n 'Proxy' "export PREMISES_MODE=proxy; cd ./controlpanel; sleep 3; go run .; bash" \;
