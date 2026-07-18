#!/bin/sh
set -e

# Copy pregenerated data to /data only if it doesn't already exist in the mounted volume
# Using -n (no clobber) so it won't overwrite any existing files the user might have generated
if [ -d "/pregen-data/sourceVideo" ]; then
    cp -rn /pregen-data/sourceVideo/* /data/sourceVideo/ 2>/dev/null || true
fi

if [ -d "/pregen-data/video" ]; then
    cp -rn /pregen-data/video/* /data/video/ 2>/dev/null || true
fi

if [ -d "/pregen-data/stream" ]; then
    cp -rn /pregen-data/stream/* /data/stream/ 2>/dev/null || true
fi

# Execute the main application
exec "$@"
