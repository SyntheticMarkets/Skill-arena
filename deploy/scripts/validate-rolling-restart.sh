#!/usr/bin/env sh
set -eu

: "${SKILL_ARENA_HEALTH_URL:?SKILL_ARENA_HEALTH_URL is required}"
: "${SKILL_ARENA_COMPOSE_FILE:?SKILL_ARENA_COMPOSE_FILE is required}"

service="${SKILL_ARENA_BACKEND_SERVICE:-backend}"
project="${SKILL_ARENA_COMPOSE_PROJECT:-skill-arena}"

containers=$(docker compose -p "$project" -f "$SKILL_ARENA_COMPOSE_FILE" ps -q "$service")
count=$(printf '%s\n' "$containers" | sed '/^$/d' | wc -l | tr -d ' ')
if [ "$count" -lt 2 ]; then
  echo "rolling restart requires at least two backend instances" >&2
  exit 1
fi

assert_ready() {
  attempts=0
  while [ "$attempts" -lt 60 ]; do
    if curl --fail --silent --show-error "$SKILL_ARENA_HEALTH_URL" >/dev/null; then
      return 0
    fi
    attempts=$((attempts + 1))
    sleep 1
  done
  return 1
}

assert_ready
for container in $containers; do
  docker restart "$container" >/dev/null
  assert_ready
  state=$(docker inspect --format '{{.State.Status}}' "$container")
  test "$state" = "running"
done
assert_ready

echo "rolling restart validation passed for $count backend instances"
