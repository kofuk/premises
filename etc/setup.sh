#!/usr/bin/env bash

useradd -Us/bin/bash -u1000 premises

DEBIAN_FRONTEND=noninteractive \
    apt install -y \
    openjdk-17-jre-headless \
    xz-utils

mkdir -p "${PREMISES_BASEDIR}/servers.d"
