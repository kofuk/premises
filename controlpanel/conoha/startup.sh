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

    curl -O 'https://storage.googleapis.com/premises-artifacts/metadata.txt'
    remote_version="$(cut -d\  -f1 metadata.txt)"

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

__run() {
    PREMISES_BASEDIR=/opt/premises

    # Keep system up-to-date
    (
        export DEBIAN_FRONTEND=noninteractive
        apt-get update &>>/tmp/premises-apt-upgrade.log
        apt-get upgrade -y &>>/tmp/premises-apt-upgrade.log
        apt-get autoremove -y --purge &>>/tmp/premises-apt-upgrade.log
    ) &
    _apt_pid=$!

    (
        set -euo pipefail

        if [ -d /premises-dev ]; then
            do_local_update
        else
            do_remote_update
        fi
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
} && __run |& tee /tmp/premises-startup.log
