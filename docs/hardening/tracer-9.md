# Tracer 9 Hardening

## Scope

Tracer 9 closed only the first routed read-only gateway line:

- one Go HTTP runtime
- one manifest directory, currently `../ashton-proto/mcp`
- one listed tool: `athena.get_current_occupancy`
- one routed read-only ATHENA occupancy call
- bounded success and upstream-failure logs

This hardening note does not claim caller identity, persisted audit storage,
write approvals, or broader multi-service routing.

## Verified Checks

```bash
cd /Users/zizo/Personal-Projects/ASHTON/ashton-mcp-gateway
go test ./...
go test -count=5 ./internal/athena ./internal/gateway ./internal/server ./internal/config ./internal/manifest ./cmd/ashton-mcp-gateway
go vet ./...
go build ./cmd/ashton-mcp-gateway
```

## Local Smoke

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
go run ./cmd/ashton-mcp-gateway
```

Verify the first route:

```bash
curl -sS http://127.0.0.1:18091/health
curl -sS -X POST http://127.0.0.1:18091/mcp/v1/tools/list \
  -H 'Content-Type: application/json' \
  --data '{}'
curl -sS -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  --data '{"tool_name":"athena.get_current_occupancy","arguments":{"facility_id":"ashtonbee"}}'
```

Observed success shape:

- `health` returned `{"service":"ashton-mcp-gateway","status":"ok","manifests_loaded":1}`
- `tools/list` returned one tool: `athena.get_current_occupancy`
- `tools/call` returned:
  - `facility_id`
  - `current_count`
  - `observed_at`
  - `source_service`
  - `latency_ms`

Observed success log shape:

```text
INFO gateway tool call tool_name=athena.get_current_occupancy source_service=athena facility_id=ashtonbee latency_ms=<non-negative integer> outcome=success
```

## Degraded Check

With ATHENA unavailable, the same routed call returned:

- `502 Bad Gateway`
- clear upstream failure text

Observed degraded log shape:

```text
INFO gateway tool call tool_name=athena.get_current_occupancy source_service=athena facility_id=ashtonbee latency_ms=<non-negative integer> outcome=upstream_error
```

## Carry-Forward Gaps

- caller identity is not real yet
- persisted audit storage is not real yet
- the runtime remains intentionally single-tool and single-route
- write approvals are still deferred
- release gating is still manual rather than CI-enforced
