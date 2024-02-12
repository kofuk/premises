#!/usr/bin/env bash

cd "$(dirname "${BASH_SOURCE:-0}")"

exec docker build -t premises.kofuk.org/dev-runner -f Dockerfile.runner \
     --label org.kofuk.premises.managed=true \
     --label org.kofuk.premises.id=$(uuidgen) \
     --label org.kofuk.premises.name=mc-premises .
