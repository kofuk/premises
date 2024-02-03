#!/usr/bin/env bash

name="premises"
project_root="$(cd "$(dirname "${BASH_SOURCE:-0}")/.."; pwd)"

export DOCKER_API_VERSION="$(docker version --format '{{ .Server.APIVersion }}')"

make -C "${project_root}/runner" deploy-dev

exec tmux new-session -s "${name}" -n 'Middleware' 'docker compose up' \; \
     new-window -n 'ConoHa Emulator' "cd ${project_root}/ostack-fake; go run ." \; \
     new-window -n 'Frontend' "cd ${project_root}/controlpanel; npm start" \; \
     new-window -n 'Backend' "cd ${project_root}/controlpanel; go run ." \;
