#!/bin/bash
function __run() {
    PREMISES_BASEDIR=/opt/premises

    # Keep system up-to-date
    (
        export DEBIAN_FRONTEND=noninteractive
        apt-get update
        apt-get upgrade -y
        apt-get autoremove -y --purge
    ) &
    _apt_pid=$!

    gpg --import <<'EOF'
-----BEGIN PGP PUBLIC KEY BLOCK-----

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
=LpmY
-----END PGP PUBLIC KEY BLOCK-----
EOF

    # Self-update
    (
        cd '/tmp'

        curl -O 'https://prmssupd.000webhostapp.com/files/meta'
        curl -O 'https://prmssupd.000webhostapp.com/files/meta.asc'
        if gpg --verify 'meta.asc'; then
            remote_version="$(cat -d\  -f1 'meta')"
            meta_hash="$(cat -d\ -f2 'meta')"

            current_version="$("${PREMISES_BASEDIR}/bin/premises-mcmanager" --version)"

            if [ "${current_version}" != "${remote_version}" ]; then
                curl -O 'https://prmssupd.000webhostapp.com/files/premises-mcmanager.tar.xz'
                curl -O 'https://prmssupd.000webhostapp.com/files/premises-mcmanager.tar.xz.asc'

                if gpg --verify 'premises-mcmanager.tar.xz.asc'; then
                    tar -xf 'premises-mcmanager.tar.xz'
                    mv 'premises-mcmanager' "${PREMISES_BASEDIR}/bin/premises-mcmanager"

                    # Make sure new version launched
                    pid="$(pidof -s premises-mcmanager)"
                    kill -KILL "${pid}"
                    tail -f /dev/null --pid "${pid}"
                fi
            fi
        fi

        rm -f meta{,.asc} premises-mcmanager.tar.xz{,.asc}
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

    chown -R premises:premises "${PREMISES_BASEDIR}"

    wait "${_apt_pid}"

    exit
} && __run
