#!/usr/bin/env bash

name="premises"
project_root="$(cd "$(dirname "${BASH_SOURCE:-0}")/.."; pwd)"

export DOCKER_API_VERSION="$(docker version --format '{{ .Server.APIVersion }}')"

make -C "${project_root}/runner" deploy-dev

exec tmux new-session -s "${name}" -n 'Middleware' "cd ${project_root}/dev; docker compose up; bash" \; \
     new-window -n 'Redis Console' "cd ${project_root}/dev; while true; do sleep 3; docker compose exec redis redis-cli; done" \; \
     new-window -n 'PostgreSQL Console' "cd ${project_root}/dev; while true; do sleep 3; docker compose exec postgres psql -Upremises; done" \; \
     new-window -n 'ConoHa Emulator' "cd ${project_root}/ostack-fake; go run .; bash" \; \
     new-window -n 'Frontend' "cd ${project_root}/controlpanel; npm start; bash" \; \
     new-window -n 'Backend' "cd ${project_root}/controlpanel; sleep 3; go run .; bash" \;
