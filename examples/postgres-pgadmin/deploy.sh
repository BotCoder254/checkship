#!/usr/bin/env bash
# Deploys Postgres + pgAdmin to a local checkship through its API.
#
#   CHECKSHIP_EMAIL=admin@example.com CHECKSHIP_PASSWORD=... ./deploy.sh
#
# It signs in like the dashboard does: creating targets and apps is not open to API keys.
# It reuses (or creates) the local Docker target and the "postgres" app, keeps the
# generated passwords in .secrets.env next to this script (git ignored), uploads
# this folder as an archive and waits for the production deployment to finish.
set -euo pipefail

cd "$(dirname "$0")"

CHECKSHIP_URL="${CHECKSHIP_URL:-http://localhost:8080}"
TARGET_URL="${TARGET_URL:-http://docker.localhost}"
APP_NAME="${APP_NAME:-postgres}"
: "${CHECKSHIP_EMAIL:?Set CHECKSHIP_EMAIL to your checkship login}"
if [[ -z "${CHECKSHIP_PASSWORD:-}" ]]; then
	read -rsp "checkship password for $CHECKSHIP_EMAIL: " CHECKSHIP_PASSWORD
	echo
fi

cookies="$(mktemp)"
archive="$(mktemp --suffix=.tar.gz)"
trap 'rm -f "$cookies" "$archive"' EXIT

api() {
	local method="$1" path="$2"
	shift 2
	curl -fsS -b "$cookies" -c "$cookies" -X "$method" "$@" "$CHECKSHIP_URL/api/v1$path"
}

api POST /sessions -H 'Content-Type: application/json' \
	-d "$(jq -n --arg email "$CHECKSHIP_EMAIL" --arg password "$CHECKSHIP_PASSWORD" '{email: $email, password: $password}')" >/dev/null

# 1. Secrets, generated once and kept so redeploys don't change them.
if [[ ! -f .secrets.env ]]; then
	umask 077
	cat > .secrets.env <<EOF
POSTGRES_PASSWORD=$(openssl rand -hex 16)
PGADMIN_DEFAULT_EMAIL=admin@example.com
PGADMIN_DEFAULT_PASSWORD=$(openssl rand -hex 12)
EOF
	echo "Generated .secrets.env"
fi
# shellcheck disable=SC1091
source .secrets.env

# 2. Local Docker target with an automatic proxy.
target_id="$(api GET /targets | jq -r --arg url "$TARGET_URL" '[.[] | select(.url == $url and (.provider.data.host // "") == "")][0].id // empty')"

if [[ -z "$target_id" ]]; then
	echo "Creating the local target $TARGET_URL"
	target_id="$(api POST /targets -H 'Content-Type: application/json' \
		-d "$(jq -n --arg url "$TARGET_URL" '{name: "local docker", url: $url, docker: {}}')" | jq -r .id)"
fi

echo -n "Waiting for target $target_id to be ready"
for _ in $(seq 1 90); do
	status="$(api GET "/targets/$target_id" | jq -r .state.status)"
	[[ "$status" == 2 ]] && break
	if [[ "$status" == 1 ]]; then
		echo
		api GET "/targets/$target_id" | jq .state >&2
		echo "The target failed to configure, check the Jobs page." >&2
		exit 1
	fi
	echo -n .
	sleep 2
done
[[ "$status" == 2 ]] || { echo; echo "The target is still configuring, try again later." >&2; exit 1; }
echo " ready"

# 3. The app, with the passwords as per-service variables.
vars="$(jq -n \
	--arg pg "$POSTGRES_PASSWORD" --arg email "$PGADMIN_DEFAULT_EMAIL" --arg pga "$PGADMIN_DEFAULT_PASSWORD" \
	'{db: {POSTGRES_PASSWORD: $pg}, pgadmin: {PGADMIN_DEFAULT_EMAIL: $email, PGADMIN_DEFAULT_PASSWORD: $pga}}')"

app_id="$(api GET /apps | jq -r --arg name "$APP_NAME" '[.[] | select(.name == $name)][0].id // empty')"

if [[ -z "$app_id" ]]; then
	echo "Creating the app $APP_NAME"
	app_id="$(api POST /apps -H 'Content-Type: application/json' -d "$(jq -n \
		--arg name "$APP_NAME" --arg target "$target_id" --argjson vars "$vars" \
		'{name: $name, production: {target: $target, vars: $vars}, staging: {target: $target, vars: $vars}}')" | jq -r .id)"
fi

# 4. Upload this folder and wait for the deployment.
tar -czf "$archive" --exclude=.secrets.env --exclude=deploy.sh --exclude=README.md .

number="$(api POST "/apps/$app_id/deployments" -F environment=production -F "archive=@$archive;type=application/gzip" | jq -r .deployment_number)"
echo -n "Deployment #$number queued"

# Up to 30 minutes: the first deployment pulls and builds every image
for _ in $(seq 1 900); do
	deployment="$(api GET "/apps/$app_id/deployments/$number")"
	status="$(jq -r .state.status <<<"$deployment")"
	[[ "$status" == 2 || "$status" == 3 ]] && break
	echo -n .
	sleep 2
done
echo

if [[ "$status" == 0 || "$status" == 1 ]]; then
	echo "Deployment #$number is still running, follow it on $CHECKSHIP_URL/apps/$app_id/deployments/$number" >&2
	exit 1
fi

if [[ "$status" != 3 ]]; then
	api GET "/apps/$app_id/deployments/$number/logs" | tail -40 >&2
	echo "Deployment #$number did not succeed (status $status)." >&2
	exit 1
fi

echo "Deployment #$number succeeded"
jq -r '.state.services[] | .name as $s | .entrypoints[] | "  \($s): \(.url // "")\(if .published_port then " (host port \(.published_port))" else "" end)"' <<<"$deployment"
