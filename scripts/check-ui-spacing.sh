#!/usr/bin/env bash
set -euo pipefail
for file in \
  src/components/PageContainer/index.vue \
  src/views/tg-saas/dashboard/index.vue \
  src/views/tg-saas/orders/index.vue \
  src/views/tg-saas/settings/index.vue \
  src/views/tg-saas/agents/index.vue \
  src/views/tg-saas/payments/index.vue
do
  test -f "$file" || { echo "Missing required page: $file"; exit 1; }
  if grep -En '(^|[[:space:]])(mt|mb)-[0-9]+|margin(-top|-bottom)?[[:space:]]*:' "$file"; then
    echo "Forbidden manual margin found in $file"
    exit 1
  fi
  grep -Eq 'gap-4' "$file" || { echo "gap-4 missing in $file"; exit 1; }
done
echo "UI spacing source check passed for all five required pages."
