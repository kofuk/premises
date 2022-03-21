#!/usr/bin/env bash

tmpname="/tmp/$(uuidgen)"
go build -o "${tmpname}"

(
    sleep 1
    rm -f "${tmpname}"
) &

exec sudo -E PREMISES_RUNNER_DEBUG=true "${tmpname}" --privileged-helper
