#!/usr/bin/env sh
set -eu

: "${PGHOST:?PGHOST is required}"
: "${PGPORT:?PGPORT is required}"
: "${PGUSER:?PGUSER is required}"
: "${PGDATABASE:?PGDATABASE is required}"
: "${SKILL_ARENA_RESTORE_DATABASE:?SKILL_ARENA_RESTORE_DATABASE is required}"

case "$SKILL_ARENA_RESTORE_DATABASE" in
  skillarena_restore_*) ;;
  *)
    echo "restore database must start with skillarena_restore_" >&2
    exit 1
    ;;
esac

artifact_dir="${SKILL_ARENA_BACKUP_ARTIFACT_DIR:-./artifacts/backup-restore}"
mkdir -p "$artifact_dir"
backup="$artifact_dir/skillarena.dump"
source_integrity="$artifact_dir/source-integrity.txt"
restore_integrity="$artifact_dir/restore-integrity.txt"

pg_dump --format=custom --no-owner --no-acl --file="$backup"

psql --dbname=postgres --set=ON_ERROR_STOP=1 \
  --command="DROP DATABASE IF EXISTS \"$SKILL_ARENA_RESTORE_DATABASE\" WITH (FORCE)"
createdb "$SKILL_ARENA_RESTORE_DATABASE"
pg_restore --exit-on-error --no-owner --no-acl \
  --dbname="$SKILL_ARENA_RESTORE_DATABASE" "$backup"

tables=$(psql --tuples-only --no-align --command="
  SELECT quote_ident(schemaname)||'.'||quote_ident(tablename)
  FROM pg_tables
  WHERE schemaname='public'
  ORDER BY tablename")

: >"$source_integrity"
: >"$restore_integrity"
for table in $tables; do
  psql --tuples-only --no-align \
    --command="SELECT '$table', count(*), md5(COALESCE(string_agg(row_data, E'\n' ORDER BY row_data), '')) FROM (SELECT row_to_json(source_row)::text AS row_data FROM $table AS source_row) AS rows" \
    >>"$source_integrity"
  PGDATABASE="$SKILL_ARENA_RESTORE_DATABASE" \
    psql --tuples-only --no-align \
      --command="SELECT '$table', count(*), md5(COALESCE(string_agg(row_data, E'\n' ORDER BY row_data), '')) FROM (SELECT row_to_json(restore_row)::text AS row_data FROM $table AS restore_row) AS rows" \
      >>"$restore_integrity"
done

diff -u "$source_integrity" "$restore_integrity"
sha256sum "$backup" >"$backup.sha256"
echo "PostgreSQL backup and restore content-integrity verification passed"
