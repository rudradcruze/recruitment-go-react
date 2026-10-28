#!/usr/bin/env bash
# This script runs the browser tests (Playwright) against the full stack.
# It starts the stack in a Compose project of its own (compose.e2e.yaml), with an empty database.
# It removes the project and its volumes when the tests end, also when a test fails or when a person presses Ctrl-C.
# Only the web service has a host port (E2E_WEB_PORT, 5655 by default), so the development stack can run at the same time.
# A kill with SIGKILL cannot run the cleanup. The next run removes what the earlier run left.
# Do not run 2 copies of this script on one Docker host at the same time: they use the same project and the same port.
set -euo pipefail

cd "$(dirname "$0")/.."

export E2E_WEB_PORT="${E2E_WEB_PORT:-5655}"
export WEB_PORT="$E2E_WEB_PORT"
# The password is new at each run, and it exists only in this process.
POSTGRES_PASSWORD="$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')"
export POSTGRES_PASSWORD

# The settings of the stack are the defaults of compose.yaml. --env-file /dev/null skips the file .env,
# and unset removes the settings that the shell of the developer can have.
unset POSTGRES_USER POSTGRES_DB POSTGRES_SSLMODE REDIS_PASSWORD REDIS_CACHE_DB REDIS_QUEUE_DB \
  ADMIN_EMAIL ADMIN_PASSWORD VITE_DEMO_ADMIN_EMAIL VITE_DEMO_ADMIN_PASSWORD API_PORT API_DOCS_ENABLED API_DOCS_AUTH_ENABLED API_DOCS_USERNAME API_DOCS_PASSWORD LOG_LEVEL WORKER_CONCURRENCY
compose=(docker compose --env-file /dev/null -f compose.yaml -f compose.e2e.yaml)

remove_project() {
  "${compose[@]}" down --volumes --remove-orphans
}

# on_exit runs when the script ends. After a failure, it shows the end of the logs of the services before it removes them.
on_exit() {
  local status=$?
  if [ "$status" -ne 0 ]; then
    echo "The tests failed or the stack did not start. The end of the logs of the services:" >&2
    "${compose[@]}" logs --no-color --tail 100 >&2 || true
  fi
  remove_project || true
  exit "$status"
}
trap on_exit EXIT

# Remove the data of an earlier run that did not end, so the database is empty.
remove_project

# The first run downloads the browser. Later runs find it.
pnpm --filter frontend exec playwright install chromium

"${compose[@]}" up -d --build --wait --wait-timeout 300

pnpm --filter frontend exec playwright test "$@"
