#!/bin/bash
PREMISES_BASEDIR=/opt/premises

# Keep system up-to-date
(
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get upgrade -y
) &
_apt_pid=$!

cat <<'EOF' >"${PREMISES_BASEDIR}/config.json"
#__CONFIG_FILE__
EOF

cat <<'EOF' >"${PREMISES_BASEDIR}/server.crt"
#__SERVER_CRT__
EOF

cat <<'EOF' >"${PREMISES_BASEDIR}/server.key"
#__SERVER_KEY__
EOF

chown -R premises:premises "${PREMISES_BASEDIR}"

wait "${_apt_pid}"
