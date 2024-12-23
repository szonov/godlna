#!/usr/bin/env bash

SUDO=""
SYNOLOGY_KERNEL=$(uname -a | grep 'synology_')
if [[ -n "${SYNOLOGY_KERNEL}" ]]; then
  SUDO="sudo --user=postgres"
fi

echo "SUDO: '${SUDO}'"

echo "0. Drop database for godlna"
$SUDO dropdb godlna

echo "1. Create database for godlna"
$SUDO createdb godlna -E utf8 -T template0

echo "2. Install database schema"
$SUDO psql godlna -c "
  CREATE TABLE IF NOT EXISTS objects (
    id BIGSERIAL PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    typ SMALLINT NOT NULL,
    format TEXT NOT NULL DEFAULT '',
    file_size BIGINT NOT NULL DEFAULT 0,
    video_codec TEXT NOT NULL DEFAULT '',
    audio_codec TEXT NOT NULL DEFAULT '',
    width INT NOT NULL DEFAULT 0,
    height INT NOT NULL DEFAULT 0,
    channels INT NOT NULL DEFAULT 0,
    bitrate INT NOT NULL DEFAULT 0,
    frequency INT NOT NULL DEFAULT 0,
    duration BIGINT NOT NULL DEFAULT 0,
    bookmark BIGINT,
    date BIGINT NOT NULL,
    online BOOLEAN NOT NULL DEFAULT true
  );

  CREATE UNIQUE INDEX ON objects (path);

  CREATE TABLE IF NOT EXISTS queue (
      id BIGSERIAL PRIMARY KEY,
      path TEXT NOT NULL UNIQUE
  );

  CREATE UNIQUE INDEX ON queue (path);
"
