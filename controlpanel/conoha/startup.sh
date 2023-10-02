#!/bin/bash

atomic_copy() {
    from="$1"
    to="$2"

    cp "${from}" "${to}.new"
    chmod +x "${to}.new"
    mv "${to}"{.new,}
}

do_remote_update() {
    cd '/tmp'

    remote_version="$(curl 'https://storage.googleapis.com/premises-artifacts/version.txt')"

    current_version="$("${PREMISES_BASEDIR}/bin/premises-mcmanager" --version || true)"

    [ "${current_version}" = "${remote_version}" ] && exit 0

    curl 'https://storage.googleapis.com/premises-artifacts/premises-mcmanager.tar.gz' | tar -xz
    atomic_copy 'premises-mcmanager' "${PREMISES_BASEDIR}/bin/premises-mcmanager"

    curl 'https://storage.googleapis.com/premises-artifacts/exteriord.tar.gz' | tar -xz
    atomic_copy 'exteriord' "/opt/premises/bin/exteriord"

    # Make sure new version launched
    pid="$(pidof -s premises-mcmanager)"
    kill -KILL "${pid}"
    tail -f /dev/null --pid "${pid}"
}

do_local_update() {
    cd /premises-dev
    [ -e exteriord ] && atomic_copy exteriord /opt/premises/bin/exteriord
    [ -e premises-mcmanager ] && atomic_copy premises-mcmanager /opt/premises/bin/premises-mcmanager
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

    mkdir -p "${PREMISES_BASEDIR}/bin"

    if ! command -v curl &>/dev/null; then
        # Unfortunately, first initialization can't be done without installing curl.
        (
            export DEBIAN_FRONTEND=noninteractive
            apt-get update -y
            apt-get install -y curl
        )
    fi

    (
        set -euo pipefail

        if [ -d /premises-dev ]; then
            do_local_update
        else
            do_remote_update
        fi
    )

    cat <<'EOF' >"${PREMISES_BASEDIR}/server.crt"
#__SERVER_CRT__
EOF

    cat <<'EOF' >"${PREMISES_BASEDIR}/server.key"
#__SERVER_KEY__
EOF

    cat <<'EOF' >"${PREMISES_BASEDIR}/config.json"
#__CONFIG_FILE__
EOF

    nohup /opt/premises/bin/exteriord &>/exteriord.log &

    exit
} && __run |& tee /tmp/premises-startup.log
