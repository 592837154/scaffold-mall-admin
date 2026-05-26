#!/bin/sh
set -e

export GIN_MODE=release
export MYSQL_HOST="${MYSQL_HOST:-gateway01.ap-northeast-1.prod.aws.tidbcloud.com}"
export MYSQL_PORT="${MYSQL_PORT:-4000}"
export MYSQL_USER="${MYSQL_USER:-3d1HaJiKd8td5iE.root}"
export MYSQL_DATABASE="${MYSQL_DATABASE:-mall_admin}"
export MYSQL_TLS="${MYSQL_TLS:-skip-verify}"

/app/server &

nginx -g "daemon off;"
