# Tracer 15 Hardening

## Scope

Tracer 15 closes the next narrow control-plane line:

- caller identity for routed reads
- persisted audit for routed calls
- one second routed read
- no writes, approvals, rate limiting, or deployment proof

This hardening note does not claim live gateway deployment, write governance,
APOLLO routing, HERMES routing, or broad orchestration.

## Verified Checks

```bash
cd /Users/zizo/Personal-Projects/ASHTON/ashton-mcp-gateway
go test ./...
go test -count=5 ./internal/...
go vet ./...
go build ./cmd/ashton-mcp-gateway

cd /Users/zizo/Personal-Projects/ASHTON/ashton-proto
go test ./...
```

## Local Smoke

Manual local smoke was run against:

- local Postgres `postgres:16-alpine` on `127.0.0.1:15432`
- local ATHENA mock runtime on `127.0.0.1:18090`
- local gateway runtime on `127.0.0.1:18091`

Manual HTTP checks:

```bash
curl -sS http://127.0.0.1:18091/health
curl -sS -X POST http://127.0.0.1:18091/mcp/v1/tools/list \
  -H 'Content-Type: application/json' \
  --data '{}'
curl -sS -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-API-Key: automation-secret' \
  --data '{"tool_name":"athena.get_current_occupancy","arguments":{"facility_id":"ashtonbee"}}'
curl -sS -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-Trusted-Caller-Token: trusted-token' \
  -H 'X-Gateway-Caller-Type: interactive' \
  -H 'X-Gateway-Caller-Id: operator-001' \
  -H 'X-Gateway-Caller-Display: Operator One' \
  --data '{"tool_name":"athena.get_current_zone_occupancy","arguments":{"facility_id":"ashtonbee","zone_id":"gym-floor"}}'
curl -sS -i -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  --data '{"tool_name":"athena.get_current_occupancy","arguments":{"facility_id":"ashtonbee"}}'
curl -sS -i -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-API-Key: missing' \
  --data '{"tool_name":"athena.get_current_occupancy","arguments":{"facility_id":"ashtonbee"}}'
curl -sS -i -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-Trusted-Caller-Token: trusted-token' \
  -H 'X-Gateway-Caller-Type: interactive' \
  --data '{"tool_name":"athena.get_current_occupancy","arguments":{"facility_id":"ashtonbee"}}'
curl -sS -i -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-API-Key: automation-secret' \
  --data '{"tool_name":"missing.tool","arguments":{"facility_id":"ashtonbee"}}'
curl -sS -i -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  -H 'X-Gateway-API-Key: automation-secret' \
  --data '{"tool_name":"athena.get_current_zone_occupancy","arguments":{"facility_id":"ashtonbee"}}'
docker exec tracer15-gateway-pg psql -U gateway -d gateway -c \
  "select caller_type, caller_id, tool_name, outcome, http_status from gateway_audit_log order by occurred_at asc;"
```

Observed success truth:

- `health` returned `{"service":"ashton-mcp-gateway","status":"ok","manifests_loaded":2}`
- `tools/list` returned both manifest-backed tools
- API-key-backed occupancy read returned a `200` with `facility_id`,
  `current_count`, `observed_at`, `source_service`, and `latency_ms`
- trusted-header-backed zone occupancy read returned a `200` with the same
  fields plus `zone_id`
- the mock ATHENA runtime returned `current_count: 0` for the zone-filtered
  path, which is still an honest routed zone result over the live public
  endpoint

Observed persisted audit truth:

- success row for `athena.get_current_occupancy`
- success row for `athena.get_current_zone_occupancy`
- `unknown_tool` row for `missing.tool`
- `invalid_arguments` row for missing `zone_id`
- no audit row for missing or malformed caller identity before the route
  boundary

Observed failure truth:

- missing identity returned `401 Unauthorized`
- unknown API key returned `401 Unauthorized`
- malformed trusted caller headers returned `400 Bad Request`
- unknown tool returned `404 Not Found`
- missing `zone_id` returned `400 Bad Request`

## Carry-Forward Gaps

- APOLLO and HERMES remain out of scope for routed reads
- write approvals are still deferred to `v0.3.0`
- deployment proof remains unchanged in this tracer
- rate limiting and broader registry work remain later lines
