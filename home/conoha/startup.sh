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
=bLfa
-----END PGP PUBLIC KEY BLOCK-----
EOF

    # Self-update
    (
        cd '/tmp'

        curl -O 'https://prmssupd.000webhostapp.com/files/meta'
        curl -O 'https://prmssupd.000webhostapp.com/files/meta.asc'
        if gpg --verify 'meta.asc'; then
            remote_version="$(cut -d\  -f1 'meta')"
            meta_hash="$(cut -d\  -f2 'meta')"

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
