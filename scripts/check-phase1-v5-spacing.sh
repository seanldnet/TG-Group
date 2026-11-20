#!/usr/bin/env bash
set -euo pipefail

files=(
  apps/web-antd/src/views/tg-saas/dashboard/index.vue
  apps/web-antd/src/views/tg-saas/orders/index.vue
  apps/web-antd/src/views/tg-saas/settings/index.vue
  apps/web-antd/src/views/tg-saas/agents/index.vue
  apps/web-antd/src/views/tg-saas/payments/index.vue
)

for file in "${files[@]}"; do
  test -f "$file" || { echo "Missing required V5 page: $file"; exit 1; }
  if grep -En '(^|[[:space:]])(mt|mb)-[0-9]+|margin(-top|-bottom)?[[:space:]]*:' "$file"; then
    echo "Forbidden manual margin found in $file"
    exit 1
  fi
  grep -Eq 'gap-4' "$file" || { echo "gap-4 missing in $file"; exit 1; }
done

if grep -REn 'BasicTable|BasicForm|BasicModal|BasicDrawer|PageWrapper|@/components/(Table|Form|Modal|Drawer)' apps/web-antd/src/views/tg-saas; then
  echo "V2-only component reference found in V5 pages"
  exit 1
fi

echo "V5 page spacing and no-V2 compatibility checks passed."