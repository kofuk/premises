#!/usr/bin/env bash

cd "$(dirname "${BASH_SOURCE:-0}")"

docker build -t premises.kofuk.org/dev-runner-raw .

name="premises-runner-${RANDOM}"

docker container run \
       -d -p 127.0.0.2:25565:25565 --privileged \
       -v /dev/null:/userdata:ro \
       --name "${name}" \
       --cap-add MKNOD \
       --add-host host.docker.internal:host-gateway \
       --label org.kofuk.premises.managed=true \
       --label org.kofuk.premises.id="$(uuidgen)" \
       --label org.kofuk.premises.name="runner-${RANDOM}" \
       premises.kofuk.org/dev-runner-raw

docker exec -it "${name}" bash

