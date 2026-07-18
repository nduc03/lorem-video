#!/bin/sh
set -e

# Ensure directories exist so cp doesn't fail or skip due to busybox limitations
mkdir -p /data/sourceVideo /data/video /data/stream

if [ -d "/pregen-data" ]; then
    # Copy files from each directory, not overwriting existing ones.
    # Busybox cp -n skips the whole directory if the dest directory exists, 
    # so we must copy the contents of the directories instead.
    cp -rn /pregen-data/sourceVideo/* /data/sourceVideo/ 2>/dev/null || true
    cp -rn /pregen-data/video/* /data/video/ 2>/dev/null || true
    cp -rn /pregen-data/stream/* /data/stream/ 2>/dev/null || true
fi

# Execute the main application
exec "$@"
