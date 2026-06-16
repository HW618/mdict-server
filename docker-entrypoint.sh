#!/bin/sh
set -e

# Fix ownership of mounted volumes (Docker bind mounts retain host ownership)
chown -R mdict:mdict /dicts /data 2>/dev/null || true

exec su-exec mdict ./mdict-server "$@"
