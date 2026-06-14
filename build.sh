#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

# do we want to skip gamdl checks?
SKIP_GAMDL=0
for arg in "$@"; do
  case "$arg" in
    --skip-gamdl) SKIP_GAMDL=1 ;;
    --admin-url)
      shift
      [ "$#" -gt 0 ] && ADMIN_URL="$1" || true
      ;;
    *) echo "unknown flag: $arg" >&2; exit 2 ;;
  esac

done

# make uv avaliable 
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$PATH"

bold() { printf '\033[1m%s\033[0m\n' "$1"; }
info() { printf '\033[36m==>\033[0m %s\n' "$1"; }
warn() { printf '\033[33mwarning:\033[0m %s\n' "$1" >&2; }
die()  { printf '\033[31merror:\033[0m %s\n' "$1" >&2; exit 1; }

have() { command -v "$1" >/dev/null 2>&1; }

# check if we have everythin
info "Checking prerequisites"
have go   || die "Go is not installed, go to: https://go.dev/dl/"
have node || die "Node.js is not installed, go to: https://nodejs.org/ (or use nvm)"
have npm  || die "npm is not installed (comes with Node.js)"
echo "  go   $(go version | awk '{print $3}')"
echo "  node $(node --version)"
echo "  npm  v$(npm --version)"

# downloads UV & gamdl
if [ "$SKIP_GAMDL" -eq 0 ]; then
  info "Ensuring uv is installed"
  if ! have uv; then
    warn "uv not found, trying to install via the official installer"
    if have curl; then
      curl -LsSf https://astral.sh/uv/install.sh | sh
    elif have wget; then
      wget -qO- https://astral.sh/uv/install.sh | sh
    else
      die "need curl or wget to install uv (or install it manually: https://docs.astral.sh/uv/)"
    fi
    export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$PATH"
    have uv || die "uv still not on PATH after install, open a new shell and re-run, or add ~/.local/bin to PATH"
  fi
  echo "  uv $(uv --version | awk '{print $2}')"

  info "Ensuring gamdl is installed (uv tool)"
  if have gamdl; then
    echo "  gamdl already on PATH; upgrading"
    uv tool upgrade gamdl || warn "gamdl upgrade failed (keeping existing install)"
  else
    uv tool install gamdl || die "failed to install gamdl via uv"
  fi
  have gamdl && echo "  gamdl -> $(command -v gamdl)" \
             || warn "gamdl not on PATH yet; the app will fall back to 'uv tool run gamdl'"
else
  info "Skipping uv/gamdl (--skip-gamdl)"
fi

# builds web
info "Building the admin web panel"
pushd "web/admin" >/dev/null
if [ -f package-lock.json ]; then
  npm ci
else
  npm install
fi
npm run build -- ${ADMIN_URL:+--admin-url "$ADMIN_URL"}
popd >/dev/null

# check for configuration.yml, if none then clone from src/config/default-config.yml
if [ ! -f configuration.yml ]; then
  info "No configuration.yml found, copying default from src/config/default-config.yml"
  cp src/config/default-config.yml configuration.yml
fi

# build go bin
info "Building the aplsonic binary"
go build -o aplsonic .

bold "Done."
cat <<'EOF'

Now run the following commands:
  1. docker compose up -d        # start TiDB
  2. ./aplsonic serve            # prints a one-time admin password on first run
  3. open the admin panel at http://localhost:<web_port>/admin  (web_port in configuration.yml)

To create an account from the CLI instead: ./aplsonic create-account
EOF
