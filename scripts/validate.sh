#!/usr/bin/env sh
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"
set -a
if [ -f .env ]; then . ./.env; else . ./.env.example; fi
set +a
(command -v jq >/dev/null 2>&1) || { echo "jq is required for API validation" >&2; exit 1; }
(cd backend && go test ./... && go build ./...)
(cd frontend && npm install --no-audit --no-fund && npm run build)
docker compose config --quiet
docker compose up -d --build
cleanup() { docker compose down -v --remove-orphans; }
if [ "${KEEP_RUNNING:-0}" = "1" ]; then
  trap cleanup INT TERM
else
  trap cleanup EXIT INT TERM
fi
i=0
until curl -fsS "http://127.0.0.1:${BACKEND_PORT:-19512}/healthz" >/dev/null; do
  i=$((i+1)); [ "$i" -lt 60 ] || { docker compose logs; exit 1; }; sleep 2
done
curl -fsS "http://127.0.0.1:${FRONTEND_PORT:-18512}/" >/dev/null
token=$(curl -fsS -X POST "http://127.0.0.1:${BACKEND_PORT:-19512}/api/auth/login" -H 'Content-Type: application/json' -d '{"username":"admin","password":"Admin123!"}' | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
[ -n "$token" ]
viewer_token=$(curl -fsS -X POST "http://127.0.0.1:${BACKEND_PORT:-19512}/api/auth/login" -H 'Content-Type: application/json' -d '{"username":"viewer","password":"Admin123!"}' | jq -er '.data.token')
operator_token=$(curl -fsS -X POST "http://127.0.0.1:${BACKEND_PORT:-19512}/api/auth/login" -H 'Content-Type: application/json' -d '{"username":"operator","password":"Admin123!"}' | jq -er '.data.token')
reviewer_token=$(curl -fsS -X POST "http://127.0.0.1:${BACKEND_PORT:-19512}/api/auth/login" -H 'Content-Type: application/json' -d '{"username":"reviewer","password":"Admin123!"}' | jq -er '.data.token')
curl -fsS "http://127.0.0.1:${BACKEND_PORT:-19512}/api/overview" -H "Authorization: Bearer $token" >/dev/null
curl -fsS "http://127.0.0.1:${BACKEND_PORT}/api/session" -H "Authorization: Bearer $token" | jq -e '.data.role == "admin" and (.data.requestId | length > 0)' >/dev/null
curl -fsS "http://127.0.0.1:${BACKEND_PORT}/api/runtime" -H "Authorization: Bearer $token" | jq -e '.data.appName and .data.databaseDriver and (.data.requestLimit > 0)' >/dev/null
paths=$(sed -n "s/.*path: '\\([^']*\\)'.*/\\1/p" frontend/src/types/status.ts)
for path in $paths; do
  curl -fsS "http://127.0.0.1:${BACKEND_PORT}/api/$path?page=1&pageSize=20" -H "Authorization: Bearer $token" | jq -e '.data | type == "array"' >/dev/null
done
entity_config=$(sed -n "s/.*path: '\\([^']*\\)'.*statuses: \\['\\([^']*\\)', '\\([^']*\\)'.*/\\1|\\2|\\3/p" frontend/src/types/status.ts | head -n 1)
resource=$(printf '%s' "$entity_config" | cut -d '|' -f 1)
initial_status=$(printf '%s' "$entity_config" | cut -d '|' -f 2)
next_status=$(printf '%s' "$entity_config" | cut -d '|' -f 3)
now=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
code="SMOKE-$(date +%s)"
payload=$(printf '{"code":"%s","name":"Runtime smoke record","description":"Automated Compose workflow validation","facility":"Validation Lab","owner":"admin","category":"smoke","riskLevel":"low","metricValue":1,"metricUnit":"unit","effectiveAt":"%s","evidence":"scripts/validate.sh","relatedCode":"SMOKE"}' "$code" "$now")
created=$(curl -fsS -X POST "http://127.0.0.1:${BACKEND_PORT}/api/$resource" -H "Authorization: Bearer $token" -H 'Content-Type: application/json' -d "$payload")
id=$(printf '%s' "$created" | jq -er '.data.id')
version=$(printf '%s' "$created" | jq -er '.data.version')
printf '%s' "$created" | jq -e --arg status "$initial_status" '.data.status == $status' >/dev/null
transition=$(printf '{"status":"%s","expectedVersion":%s,"reason":"automated runtime validation"}' "$next_status" "$version")
curl -fsS -X POST "http://127.0.0.1:${BACKEND_PORT}/api/$resource/$id/transition" -H "Authorization: Bearer $token" -H 'Content-Type: application/json' -d "$transition" | jq -e --arg status "$next_status" '.data.status == $status' >/dev/null

viewer_status=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "http://127.0.0.1:${BACKEND_PORT}/api/artifacts" -H "Authorization: Bearer $viewer_token" -H 'Content-Type: application/json' -d "$payload")
[ "$viewer_status" = "403" ]

approval_code="APPROVAL-SMOKE-$(date +%s)"
approval_payload=$(printf '{"code":"%s","name":"Immutable opinion validation","description":"Stage approval workflow","facility":"Conservation Lab","owner":"operator","category":"treatment","riskLevel":"high","metricValue":9,"metricUnit":"score","effectiveAt":"%s","evidence":"Material test and image evidence attached","relatedCode":"TP-SMOKE"}' "$approval_code" "$now")
approval=$(curl -fsS -X POST "http://127.0.0.1:${BACKEND_PORT}/api/approvals" -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' -d "$approval_payload")
approval_id=$(printf '%s' "$approval" | jq -er '.data.id')
approval_version=$(printf '%s' "$approval" | jq -er '.data.version')
review_payload=$(printf '{"status":"review","expectedVersion":%s,"reason":"材料检测通过，提交阶段复核"}' "$approval_version")
reviewed=$(curl -fsS -X POST "http://127.0.0.1:${BACKEND_PORT}/api/approvals/$approval_id/transition" -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' -d "$review_payload")
printf '%s' "$reviewed" | jq -e '.data.status == "review" and (.data.opinions | length) == 1 and .data.opinions[0].version == 2 and .data.opinions[0].actor == "operator" and (.data.opinions[0].requestId | length > 0)' >/dev/null
review_version=$(printf '%s' "$reviewed" | jq -er '.data.version')
approve_payload=$(printf '{"status":"approved","expectedVersion":%s,"reason":"证据完整，同意进入下一保护阶段"}' "$review_version")
operator_approve_status=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "http://127.0.0.1:${BACKEND_PORT}/api/approvals/$approval_id/transition" -H "Authorization: Bearer $operator_token" -H 'Content-Type: application/json' -d "$approve_payload")
[ "$operator_approve_status" = "422" ]
approved=$(curl -fsS -X POST "http://127.0.0.1:${BACKEND_PORT}/api/approvals/$approval_id/transition" -H "Authorization: Bearer $reviewer_token" -H 'Content-Type: application/json' -d "$approve_payload")
printf '%s' "$approved" | jq -e '.data.status == "approved" and (.data.opinions | length) == 2 and .data.opinions[0].version == 2 and .data.opinions[1].version == 3 and .data.opinions[1].actor == "reviewer" and (.data.opinions[1].requestId | length > 0)' >/dev/null

curl -fsS "http://127.0.0.1:${BACKEND_PORT}/api/audits?page=1&pageSize=100" -H "Authorization: Bearer $token" | jq -e '.meta.total >= 2' >/dev/null
curl -fsS "http://127.0.0.1:${BACKEND_PORT}/api/audit-summary?windowHours=24" -H "Authorization: Bearer $token" | jq -e '.data.total >= 2 and .data.transitions >= 1' >/dev/null
docker compose ps
[ "${KEEP_RUNNING:-0}" = "1" ] && echo "KEEP_RUNNING=1: containers left running for browser validation"
