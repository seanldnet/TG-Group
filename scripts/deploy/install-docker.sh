#!/usr/bin/env bash
set -Eeuo pipefail

REPO="seanldnet/TG-Group"
TG_GROUP_REF="${TG_GROUP_REF:-main}"
TG_GROUP_DIR="${TG_GROUP_DIR:-/opt/tg-group}"
RAW_BASE="https://raw.githubusercontent.com/${REPO}/${TG_GROUP_REF}"

log() { printf '[tg-group] %s\n' "$*"; }
fail() { printf '[tg-group] ERROR: %s\n' "$*" >&2; exit 1; }

[[ "$(id -u)" -eq 0 ]] || fail "请使用 sudo 运行，例如 curl ... | sudo bash"
[[ "$TG_GROUP_REF" != *[[:space:]]* ]] || fail "TG_GROUP_REF 不能包含空白字符"

for cmd in docker curl openssl; do
  command -v "$cmd" >/dev/null 2>&1 || fail "缺少命令：$cmd"
done
docker info >/dev/null 2>&1 || fail "Docker daemon 未运行或当前用户无权限访问"
docker compose version >/dev/null 2>&1 || fail "未安装 Docker Compose v2 插件"

install -d -m 0750 "$TG_GROUP_DIR"
cd "$TG_GROUP_DIR"

tmp_compose="$(mktemp)"
trap 'rm -f "$tmp_compose"' EXIT
log "下载 Compose 配置（ref: $TG_GROUP_REF）"
curl --fail --silent --show-error --location "$RAW_BASE/compose.yaml" -o "$tmp_compose"
grep -Eq '^[[:space:]]*image:' "$tmp_compose" || fail "下载的 compose.yaml 无效"
if grep -Eq '^[[:space:]]*build[[:space:]]*:' "$tmp_compose"; then
  fail "安全检查失败：Compose 不允许 build:，只允许预构建镜像"
fi
install -m 0644 "$tmp_compose" "$TG_GROUP_DIR/compose.yaml"

if [[ ! -e "$TG_GROUP_DIR/.env" ]]; then
  log "首次安装：生成独立随机数据库、Redis 和 Bot Token 加密密钥"
  umask 077
  cat > "$TG_GROUP_DIR/.env" <<ENV
APP_IMAGE=ghcr.io/seanldnet/tg-group:latest
APP_PORT=8080
POSTGRES_DB=tg_group
POSTGRES_USER=tg_group
POSTGRES_PASSWORD=$(openssl rand -hex 24)
REDIS_PASSWORD=$(openssl rand -hex 24)
REDIS_DB=0
INITIAL_ADMIN_USERNAME=admin
INITIAL_ADMIN_PASSWORD=123456
BOT_TOKEN_ENCRYPTION_KEY=$(openssl rand -hex 32)
ENV
  chmod 0600 "$TG_GROUP_DIR/.env"
else
  log "检测到已有 .env：保留全部凭据、镜像版本和管理员配置，不覆盖"
fi

log "拉取镜像并启动/升级服务"
docker compose --env-file "$TG_GROUP_DIR/.env" -f "$TG_GROUP_DIR/compose.yaml" pull
docker compose --env-file "$TG_GROUP_DIR/.env" -f "$TG_GROUP_DIR/compose.yaml" up -d
docker compose --env-file "$TG_GROUP_DIR/.env" -f "$TG_GROUP_DIR/compose.yaml" ps

log "完成。访问服务器端口（默认 8080）；首次管理员为 admin / 123456，请登录后立即修改密码。"
log "升级：再次执行本脚本；回滚：先将 .env 的 APP_IMAGE 固定为已验收的 ghcr.io/seanldnet/tg-group@sha256:<digest>，再执行本脚本。"
