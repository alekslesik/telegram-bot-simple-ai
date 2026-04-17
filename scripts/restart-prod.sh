#!/usr/bin/env bash
set -euo pipefail

# Manual production restart helper.
# Usage:
#   ./scripts/restart-prod.sh                # uses IMAGE_TAG from .env if present
#   ./scripts/restart-prod.sh v0.1.2         # explicit tag override
#   ./scripts/restart-prod.sh --no-pull      # restart without pulling
#   ./scripts/restart-prod.sh v0.1.2 --no-pull

APP_DIR="${APP_DIR:-/opt/bots/telegram-bot-simple-ai}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yaml}"
LOG_TAIL="${LOG_TAIL:-120}"
PULL_IMAGES="true"
IMAGE_TAG_OVERRIDE=""

usage() {
	echo "Usage: $0 [IMAGE_TAG] [--no-pull]"
	echo "Env overrides: APP_DIR, COMPOSE_FILE, LOG_TAIL"
}

for arg in "$@"; do
	case "$arg" in
	-h|--help)
		usage
		exit 0
		;;
	--no-pull)
		PULL_IMAGES="false"
		;;
	*)
		if [[ -n "$IMAGE_TAG_OVERRIDE" ]]; then
			echo "error: only one IMAGE_TAG argument is allowed"
			usage
			exit 1
		fi
		IMAGE_TAG_OVERRIDE="$arg"
		;;
	esac
done

if [[ ! -d "$APP_DIR" ]]; then
	echo "error: APP_DIR does not exist: $APP_DIR"
	exit 1
fi

if [[ ! -f "$APP_DIR/$COMPOSE_FILE" ]]; then
	echo "error: compose file not found: $APP_DIR/$COMPOSE_FILE"
	exit 1
fi

if [[ ! -f "$APP_DIR/.env" ]]; then
	echo "error: env file not found: $APP_DIR/.env"
	exit 1
fi

cd "$APP_DIR"

set -a
source ./.env
set +a

if [[ -n "$IMAGE_TAG_OVERRIDE" ]]; then
	export IMAGE_TAG="$IMAGE_TAG_OVERRIDE"
fi

if [[ -z "${IMAGE_TAG:-}" ]]; then
	echo "warn: IMAGE_TAG is empty; compose default may fallback to 'latest'"
else
	echo "info: using IMAGE_TAG=$IMAGE_TAG"
fi

compose_cmd=(docker compose -f "$COMPOSE_FILE")

if [[ "$PULL_IMAGES" == "true" ]]; then
	echo "info: pulling images..."
	"${compose_cmd[@]}" pull
fi

echo "info: restarting services..."
"${compose_cmd[@]}" up -d

echo "info: current status"
"${compose_cmd[@]}" ps

echo "info: postgres logs (tail=$LOG_TAIL)"
"${compose_cmd[@]}" logs --tail="$LOG_TAIL" postgres || true

echo "info: bot logs (tail=$LOG_TAIL)"
"${compose_cmd[@]}" logs --tail="$LOG_TAIL" bot || true

