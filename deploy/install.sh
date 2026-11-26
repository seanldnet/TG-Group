#!/usr/bin/env bash
set -Eeuo pipefail

APP_DIR="${TG_GROUP_INSTALL_DIR:-/opt/tg-group}"
REPO_RAW="https://raw.githubusercontent.com/seanldnet/Tg-group/main"
COMPOSE_URL="$REPO_RAW/deploy/docker-compose.yml"

if [[ "$EUID" -ne 0 ]]; then
  echo "请使用 root 执行：sudo bash install.sh"
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "未检测到 Docker，正在安装 Docker Engine..."
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update
    apt-get install -y ca-certificates curl
    curl -fsSL https://get.docker.com | sh
    systemctl enable --now docker
  elif command -v dnf >/dev/null 2>&1; then
    dnf -y install dnf-plugins-core
    dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
    dnf -y install docker-ce docker-ce-cli containerd.io docker-compose-plugin
    systemctl enable --now docker
  else
    echo "暂不支持自动安装 Docker。请先安装 Docker Engine 和 Compose Plugin 后重试。"
    exit 1
  fi
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "缺少 Docker Compose Plugin，请安装 docker-compose-plugin 后重试。"
  exit 1
fi
if ! command -v curl >/dev/null 2>&1; then
  if command -v apt-get >/dev/null 2>&1; then apt-get update && apt-get install -y curl
  elif command -v dnf >/dev/null 2>&1; then dnf -y install curl
  else echo "请先安装 curl"; exit 1; fi
fi

mkdir -p "$APP_DIR"
chmod 700 "$APP_DIR"
cd "$APP_DIR"

if [[ ! -f .env ]]; then
  if command -v openssl >/dev/null 2>&1; then
    DB_PASSWORD="$(openssl rand -hex 32)"
    BOT_KEY="$(openssl rand -hex 32)"
  else
    DB_PASSWORD="$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')"
    BOT_KEY="$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')"
  fi
  umask 077
  printf 'TG_GROUP_DB_PASSWORD=%s\nTG_GROUP_BOT_ENCRYPTION_KEY=%s\n' "$DB_PASSWORD" "$BOT_KEY" > .env
fi
chmod 600 .env

TMP_COMPOSE="$(mktemp)"
trap 'rm -f "$TMP_COMPOSE"' EXIT
curl -fsSL "$COMPOSE_URL" -o "$TMP_COMPOSE"
mv "$TMP_COMPOSE" docker-compose.yml
chmod 600 docker-compose.yml

echo "正在拉取预构建镜像（服务器不会编译项目）..."
docker compose pull
docker compose up -d --remove-orphans

echo
echo "=================================================="
echo " TG-Group 已启动"
echo " 访问地址: http://服务器IP:18680"
echo " 管理员账号: admin"
echo " 管理员密码: 123456"
echo " 安装目录: $APP_DIR"
echo " 查看日志: cd $APP_DIR && docker compose logs -f"
echo " 停止服务: cd $APP_DIR && docker compose down"
echo "=================================================="
echo "首次启动会自动迁移数据库并创建演示管理员。"
echo "请先在云服务器安全组/防火墙放行 TCP 18680。"
