#!/bin/bash
PREMISES_BASEDIR=/opt/premises

# Keep system up-to-date
(
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get upgrade -y
    apt-get autoremove -y --purge
) &
_apt_pid=$!

# Self-update
if curl -L 'https://www.dropbox.com/s/mg1o23kp4gz4dgb/metadata?raw=1' >/tmp/metadata; then
    revision="$(cut /tmp/metadata -f1)"
    sha512sum="$(cut /tmp/metadata -f2)"

    installed_revision="$("${PREMISES_BASEDIR}/bin/premises-mcmanager" --version)"

    if [ "${revision}" != "${installed_revision}" ]; then
        echo 'Downloading latest Premises Minecraft Manager...' >&2

        if curl -Lo '/tmp/premises-mcmanager' 'https://www.dropbox.com/s/80haueadzasegxz/premises-mcmanager?raw=1'; then
            downloaded_sha512sum="$(sha512sum '/tmp/premises-mcmanager' | cut -d\  -f1)"
            if [ "${downloaded_sha512sum}" = "${sha512sum}" ]; then
                chmod 755 '/tmp/premises-mcmanager'
                mv '/tmp/premises-mcmanager' "${PREMISES_BASEDIR}/bin/premises-mcmanager"
                pid="$(pidof premises-mcmanager)"
                kill -KILL "${pid}"

                # Wait for the process to be actually killed.
                tail /dev/null -f --pid "${pid}"
            else
                echo 'Checksum not match' >&2
            fi
        else
            echo 'Failed to download latest Premises Minecraft Manager.' >&2
        fi
    else
        echo 'Latest Premises Minecraft Manager is already installed.' >&2
    fi
else
    echo 'Failed to download metadata' >&2
fi

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
