#!/usr/bin/env bash

SUDO=""
SYNOLOGY_KERNEL=$(uname -a | grep 'synology_')
if [[ -n "${SYNOLOGY_KERNEL}" ]]; then
  SUDO="sudo --user=postgres"
fi

cd "$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)" || {
  echo "Error: Failed to change directory to scripts directory" >&2
  exit 1
}

echo "SUDO: '${SUDO}'"

echo "0. Drop database for godlna"
$SUDO dropdb godlna

echo "1. Create database for godlna"
$SUDO createdb godlna -E utf8 -T template0

echo "2. Install database schema"
$SUDO psql godlna < ./schema.psql.sql
