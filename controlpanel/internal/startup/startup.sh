#!/bin/bash
set -x

export PROTOCOL_VERSION='%s'

atomic_copy() {
    from="$1"
    to="$2"

    cp "${from}" "${to}.new"
    chmod +x "${to}.new"
    mv "${to}"{.new,}
}

http_get() {
    if command -v curl &>/dev/null; then
        curl "$@"
        return
    fi
    wget -O- "$@"
}

do_remote_update() {
    cd '/tmp'

    remote_version="$(http_get "https://premises.kofuk.org/artifacts/runner/version@${PROTOCOL_VERSION}.txt")"

    current_version="$("${PREMISES_BASEDIR}/bin/premises-runner" --version || true)"

    [ "${current_version}" = "${remote_version}" ] && exit 0

    http_get "https://premises.kofuk.org/artifacts/runner/premises-runner@${PROTOCOL_VERSION}.tar.gz" | tar -xz
    atomic_copy 'premises-runner' "${PREMISES_BASEDIR}/bin/premises-runner"
}

do_local_update() {
    cd /premises-dev
    [ -e premises-runner ] && atomic_copy premises-runner /opt/premises/bin/premises-runner
}

xaptget() {
    if apt-get "$@"; then
        return
    else
        dpkg --configure -a
        apt-get "$@"
    fi
}

__run() {
    PREMISES_BASEDIR=/opt/premises

    [ -f "${PREMISES_BASEDIR}/runner_env.sh" ] && source "${PREMISES_BASEDIR}/runner_env.sh" || true

    mkdir -p "${PREMISES_BASEDIR}/bin"

    if ! command -v curl &>/dev/null && ! command -v wget &>/dev/null; then
        # Unfortunately, first initialization can't be done without installing curl.
        DEBIAN_FRONTEND=noninteractive xaptget update -y
        DEBIAN_FRONTEND=noninteractive xaptget install -y curl
    fi

    (
        set -euo pipefail

        if [ -e /premises-dev/premises-runner ]; then
            do_local_update
        else
            do_remote_update
        fi
    )

    cat <<'EOF' >"${PREMISES_BASEDIR}/config.json"
%s
EOF

    [ -e /premises-dev/env ] && . /premises-dev/env
    /opt/premises/bin/premises-runner --exteriord &>/exteriord.log &

    exit
} && __run |& tee /tmp/premises-startup.log
