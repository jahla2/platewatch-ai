#!/usr/bin/env bash
set -euo pipefail

container_id="${1:-}"
if [[ -z "${container_id}" ]]; then
  echo "usage: $0 <postgres-container-id>" >&2
  exit 2
fi

db="${POSTGRES_DB:-platewatch_test}"
user="${POSTGRES_USER:-platewatch}"
restore_db="${db}_restore_verify"
backup_path="/tmp/platewatch-backup.dump"

cleanup() {
  docker exec "${container_id}" dropdb -U "${user}" --if-exists "${restore_db}" >/dev/null 2>&1 || true
  docker exec "${container_id}" rm -f "${backup_path}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

source_count="$(
  docker exec "${container_id}" psql -U "${user}" -d "${db}" -Atc \
    "SELECT COUNT(*) FROM detection_events;"
)"

docker exec "${container_id}" pg_dump \
  -U "${user}" \
  -d "${db}" \
  --format=custom \
  --file="${backup_path}"

docker exec "${container_id}" dropdb -U "${user}" --if-exists "${restore_db}"
docker exec "${container_id}" createdb -U "${user}" "${restore_db}"
docker exec "${container_id}" pg_restore \
  -U "${user}" \
  -d "${restore_db}" \
  --clean \
  --if-exists \
  "${backup_path}"

restored_count="$(
  docker exec "${container_id}" psql -U "${user}" -d "${restore_db}" -Atc \
    "SELECT COUNT(*) FROM detection_events;"
)"

if [[ "${source_count}" != "${restored_count}" ]]; then
  echo "backup restore verification failed: source=${source_count}, restored=${restored_count}" >&2
  exit 1
fi

schema_present="$(
  docker exec "${container_id}" psql -U "${user}" -d "${restore_db}" -Atc \
    "SELECT to_regclass('public.watchlist_entries') IS NOT NULL;"
)"

if [[ "${schema_present}" != "t" ]]; then
  echo "backup restore verification failed: watchlist_entries table missing" >&2
  exit 1
fi

echo "backup restore verified: detection_events=${restored_count}"
