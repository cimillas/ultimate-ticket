#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"

if [ -s "$HOME/.nvm/nvm.sh" ]; then
  # shellcheck source=/dev/null
  source "$HOME/.nvm/nvm.sh"
  nvm use >/dev/null
  cd "$FRONTEND_DIR"
  exec npm "$@"
fi

if command -v npm >/dev/null 2>&1; then
  cd "$FRONTEND_DIR"
  exec npm "$@"
fi

echo "npm not found; install Node.js or nvm to run frontend commands." >&2
exit 1
