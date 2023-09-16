#!/bin/bash
__run() {
    PREMISES_BASEDIR=/opt/premises

    # Keep system up-to-date
    (
        export DEBIAN_FRONTEND=noninteractive
        apt-get update
        apt-get upgrade -y
        apt-get autoremove -y --purge
    ) &
    _apt_pid=$!

    (
        set -euo pipefail

        cd '/tmp'

        curl -O 'https://storage.googleapis.com/premises-artifacts/metadata.txt'
        remote_version="$(cut -d\  -f1 metadata.txt)"
        meta_hash="$(cut -d\  -f2 metadata.txt)"

        current_version="$("${PREMISES_BASEDIR}/bin/premises-mcmanager" --version || true)"

        [ "${current_version}" = "${remote_version}" ] && exit 0

        curl 'https://storage.googleapis.com/premises-artifacts/premises-mcmanager.tar.xz' | tar -xJ
        mv 'premises-mcmanager' "${PREMISES_BASEDIR}/bin/premises-mcmanager"

        # Make sure new version launched
        pid="$(pidof -s premises-mcmanager)"
        kill -KILL "${pid}"
        tail -f /dev/null --pid "${pid}"
    )

    mount "${PREMISES_BASEDIR}/gamedata.img" "${PREMISES_BASEDIR}/gamedata"

    cat <<'EOF' >"${PREMISES_BASEDIR}/server.crt"
#__SERVER_CRT__
EOF

    cat <<'EOF' >"${PREMISES_BASEDIR}/server.key"
#__SERVER_KEY__
EOF

    cat <<'EOF' >"${PREMISES_BASEDIR}/config.json"
#__CONFIG_FILE__
EOF

    chown -R premises:premises "${PREMISES_BASEDIR}"

    wait "${_apt_pid}"

    exit
} && __run
