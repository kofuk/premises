#!/usr/bin/env bash
# Helper script to build dedicated server on Ubuntu
set -eu

PREMISES_BASEDIR=/opt/premises
PACKAGES=(
    btrfs-progs
    curl
    openjdk-17-jre-headless
    ufw
    unzip
)

useradd -Us/bin/bash -u1000 premises

apt update
apt upgrade
DEBIAN_FRONTEND=noninteractive apt install -y "${PACKAGES[@]}"

systemctl enable --now ufw.service
ufw enable
ufw allow 25565/tcp
ufw allow 8521/tcp

mkdir -p "${PREMISES_BASEDIR}/servers.d/../gamedata/"

dd if='/dev/zero' of="${PREMISES_BASEDIR}/gamedata.img" bs=1G count=8
mkfs.btrfs "${PREMISES_BASEDIR}/gamedata.img"

mount "${PREMISES_BASEDIR}/gamedata.img" "/mnt"
chown -R premises:premises "/mnt"
umount /mnt

# Install following service
# * premises-mcmanager.service
# * premises-privileged.service

systemctl enable --now premises-mcmanager.service
systemctl enable --now premises-privileged.service
