# Caller-Aware Audited Routed Reads Runbook

## Purpose

Use this runbook to prove the Tracer 15 gateway slice:

`caller identity -> manifest -> routed ATHENA read -> persisted audit row`

This runbook proves:

- `POST /mcp/v1/tools/list` stays open and narrow
- `POST /mcp/v1/tools/call` requires explicit caller identity
- successful routed calls persist sanitized audit rows
- routed failures that cross the tool boundary persist sanitized audit rows

This runbook does not prove:

- write approvals
- write routes
- live deployment
- broad multi-service orchestration

## Required Env

Gateway:

- `GATEWAY_HTTP_ADDR`
- `GATEWAY_MANIFEST_DIR`
- `ATHENA_BASE_URL`
- `GATEWAY_AUDIT_DATABASE_URL`
- one or both of:
  - `GATEWAY_TRUSTED_CALLER_TOKEN`
  - `GATEWAY_API_KEYS_JSON`
- optional `GATEWAY_HTTP_TIMEOUT`

ATHENA:

- `ATHENA_HTTP_ADDR`

Postgres:

- one local Postgres instance reachable from `GATEWAY_AUDIT_DATABASE_URL`

## Exact Local Commands

Start Postgres:

```bash
docker run --rm -d \
  --name tracer15-gateway-pg \
  -e POSTGRES_USER=gateway \
  -e POSTGRES_PASSWORD=gateway \
  -e POSTGRES_DB=gateway \
  -p 15432:5432 \
  postgres:16-alpine
```

Start ATHENA:

```bash
cd /Users/zizo/Personal-Projects/ASHTON/athena
ATHENA_HTTP_ADDR='127.0.0.1:18090' go run ./cmd/athena serve
```

Start the gateway:

```bash
cd /Users/zizo/Personal-Projects/ASHTON/ashton-mcp-gateway
GATEWAY_HTTP_ADDR='127.0.0.1:18091' \
GATEWAY_MANIFEST_DIR='../ashton-proto/mcp' \
ATHENA_BASE_URL='http://127.0.0.1:18090' \
GATEWAY_AUDIT_DATABASE_URL='postgres://gateway:gateway@127.0.0.1:15432/gateway?sslmode=disable' \
GATEWAY_TRUSTED_CALLER_TOKEN='trusted-token' \
GATEWAY_API_KEYS_JSON='[{"id":"ci-bot","display":"CI Bot","key":"automation-secret"}]' \
go run ./cmd/ashton-mcp-gateway
```

## Discovery And Success Checks

Health and discovery:

```bash
curl -sS http://127.0.0.1:18091/health
curl -sS -X POST http://127.0.0.1:18091/mcp/v1/tools/list \
  -H 'Content-Type: application/json' \
  --data '{}'
```

Automated caller via API key:

```bash
curl -sS -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-API-Key: automation-secret' \
  --data '{"tool_name":"athena.get_current_occupancy","arguments":{"facility_id":"ashtonbee"}}'
```

Trusted internal caller via explicit caller headers:

```bash
curl -sS -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-Trusted-Caller-Token: trusted-token' \
  -H 'X-Gateway-Caller-Type: interactive' \
  -H 'X-Gateway-Caller-Id: operator-001' \
  -H 'X-Gateway-Caller-Display: Operator One' \
  --data '{"tool_name":"athena.get_current_zone_occupancy","arguments":{"facility_id":"ashtonbee","zone_id":"gym-floor"}}'
```

Expected success truth:

- `/health` returns `manifests_loaded: 2`
- `/mcp/v1/tools/list` returns both ATHENA tools
- the occupancy route returns `facility_id`, `current_count`, `observed_at`,
  `source_service`, and `latency_ms`
- the zone route returns the same shape plus `zone_id`
- with the default ATHENA mock adapter, a zone-filtered read may legitimately
  return `current_count: 0`; that still proves the real routed zone filter and
  result shape

## Negative Checks

Missing caller identity:

```bash
curl -sS -i -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  --data '{"tool_name":"athena.get_current_occupancy","arguments":{"facility_id":"ashtonbee"}}'
```

Unknown API key:

```bash
curl -sS -i -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-API-Key: missing' \
  --data '{"tool_name":"athena.get_current_occupancy","arguments":{"facility_id":"ashtonbee"}}'
```

Malformed trusted caller identity:

```bash
curl -sS -i -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-Trusted-Caller-Token: trusted-token' \
  -H 'X-Gateway-Caller-Type: interactive' \
  --data '{"tool_name":"athena.get_current_occupancy","arguments":{"facility_id":"ashtonbee"}}'
```

Unknown tool:

```bash
curl -sS -i -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-API-Key: automation-secret' \
  --data '{"tool_name":"missing.tool","arguments":{"facility_id":"ashtonbee"}}'
```

Invalid arguments:

```bash
curl -sS -i -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-API-Key: automation-secret' \
  --data '{"tool_name":"athena.get_current_zone_occupancy","arguments":{"facility_id":"ashtonbee"}}'
```

Expected negative truth:

- missing identity returns `401`
- unknown API key returns `401`
- malformed trusted caller headers return `400`
- unknown tool returns `404`
- invalid arguments return `400`

Only routed failures that cross the tool boundary should create audit rows.
Identity failures that never resolve a caller should not create routed audit
rows.

## Audit Check

Read back the persisted rows:

```bash
docker exec tracer15-gateway-pg psql -U gateway -d gateway -c \
  "select caller_type, caller_id, tool_name, outcome, http_status from gateway_audit_log order by occurred_at asc;"
```

Expected persisted truth:

- successful occupancy call row exists
- successful zone occupancy call row exists
- unknown-tool row exists
- invalid-arguments row exists
- raw API keys, caller tokens, cookies, or auth headers do not appear in the
  table

## Cleanup

Stop the gateway and ATHENA processes, then remove the Postgres container:

```bash
docker rm -f tracer15-gateway-pg
```
