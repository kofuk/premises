#!/usr/bin/env bash

cd "$(dirname "${BASH_SOURCE:-0}")"

docker build --tag premises.kofuk.org/dev-runner-raw .

name="premises-runner-${RANDOM}"

host_nameserver=$(grep '^nameserver' /etc/resolv.conf | head -1 | cut --delimiter ' ' --fields 2)

(
    cd ../../runner
    make deploy-dev
)

extra_flags=()
if [ "${1:-}" = 'manage' ]; then
    extra_flags=(
       --label org.kofuk.premises.managed=true \
       --label org.kofuk.premises.id="$(uuidgen)" \
       --label org.kofuk.premises.name="runner-${RANDOM}"
    )
fi

docker container run \
       --detach --privileged \
       --volume /dev/null:/userdata:ro \
       --volume /tmp/premises:/premises-dev \
       --name "${name}" \
       --cap-add MKNOD \
       --network host \
       --dns "${host_nameserver}" \
       "${extra_flags[@]}" \
       premises.kofuk.org/dev-runner-raw

docker exec --interactive --tty "${name}" bash

