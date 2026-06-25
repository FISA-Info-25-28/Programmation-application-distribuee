#!/usr/bin/env sh
set -eu

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  exec docker compose "$@"
fi

if command -v podman >/dev/null 2>&1 && podman compose version >/dev/null 2>&1; then
  exec podman compose "$@"
fi

if command -v podman-compose >/dev/null 2>&1; then
  exec podman-compose "$@"
fi

echo "No Compose runtime found. Install Docker Compose or Podman Compose." >&2
exit 127
