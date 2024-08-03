#!/usr/bin/env bash

name="premises"

if tmux has-session -t "${name}" 2>/dev/null; then
  tmux attach-session -t "${name}"
  exit
fi

cd "$(dirname "${BASH_SOURCE:-0}")"

export DOCKER_API_VERSION="$(docker version --format '{{ .Server.APIVersion }}')"
export PREMISES_PROXY_BIND='127.0.0.1:25565'

make -C ./runner deploy-dev

exec tmux new-session -s "${name}" -n 'Redis Console' "redis-cli -h redis" \; \
     new-window -n 'PostgreSQL Console' "PGPASSWORD=password psql -h postgres -U premises premises" \; \
     new-window -n 'ConoHa Emulator' "cd ./cmd/ostack-fake; go run .; bash" \; \
     new-window -n 'Frontend' "cd ./controlpanel/frontend; npm start; bash" \; \
     new-window -n 'Backend' "export PREMISES_MODE=web; cd ./controlpanel/cmd/premises; sleep 3; go run .; bash" \; \
     new-window -n 'Proxy' "export PREMISES_MODE=proxy; cd ./controlpanel/cmd/premises; sleep 3; go run .; bash" \; \
     new-window -n 'Cron' "export PREMISES_MODE=cron; cd ./controlpanel/cmd/premises; sleep 3; go run .; bash" \;
