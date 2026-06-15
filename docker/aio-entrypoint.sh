#!/usr/bin/env bash
# setups aio package

set -euo pipefail

DATA="${APLSONIC_DATA:-/data}"
DB_DIR="$DATA/mysql"
STORAGE_DIR="$DATA/storage"

mkdir -p "$DB_DIR" "$STORAGE_DIR" /run/mysqld
chown -R mysql:mysql "$DB_DIR" /run/mysqld

log() { printf 'aplsonic-aio: %s\n' "$1"; }

# db
if [ ! -d "$DB_DIR/mysql" ]; then
  log "initializing mariadb data dir (first run)"
  mariadb-install-db --user=mysql --datadir="$DB_DIR" \
    --auth-root-authentication-method=normal --skip-test-db >/dev/null
fi

log "starting mariadb"
mariadbd --user=mysql --datadir="$DB_DIR" \
  --bind-address=127.0.0.1 --port=3306 --socket=/run/mysqld/mysqld.sock &
DB_PID=$!

log "waiting for mariadb"
for _ in $(seq 1 60); do
  if mariadb-admin --socket=/run/mysqld/mysqld.sock ping >/dev/null 2>&1; then break; fi
  sleep 1
done

# create db
mariadb --socket=/run/mysqld/mysqld.sock <<'SQL'
CREATE DATABASE IF NOT EXISTS aplsonic;
CREATE USER IF NOT EXISTS 'root'@'127.0.0.1' IDENTIFIED VIA mysql_native_password USING '';
GRANT ALL PRIVILEGES ON *.* TO 'root'@'127.0.0.1' WITH GRANT OPTION;
FLUSH PRIVILEGES;
SQL

# shutdown
shutdown() {
  log "shutting down"
  kill -TERM "${APP_PID:-}" "$DB_PID" 2>/dev/null || true
  wait 2>/dev/null || true
  exit 0
}
trap shutdown TERM INT

# actual process
log "starting aplsonic"
/app/aplsonic "${@:-serve}" &
APP_PID=$!

# exit when aplsonic exits
wait "$APP_PID"
shutdown
