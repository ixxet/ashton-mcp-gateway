# Gateway First Route Runbook

## Purpose

Use this runbook to prove the first routed gateway slice:
`gateway -> manifest -> ATHENA occupancy read -> structured result`.

## Rules

- load one manifest
- route one read-only call
- log the path clearly
- do not expand to broad orchestration until the first route is stable

## Required Env

Gateway:

- `GATEWAY_HTTP_ADDR`
- `GATEWAY_MANIFEST_DIR`
- `ATHENA_BASE_URL`
- optional `GATEWAY_HTTP_TIMEOUT`

ATHENA:

- `ATHENA_HTTP_ADDR`

## Exact Commands

Assumption: start from the `ASHTON/` parent directory that contains
`athena/`, `ashton-proto/`, and `ashton-mcp-gateway/`.

Start ATHENA:

```bash
cd athena
ATHENA_HTTP_ADDR='127.0.0.1:18090' go run ./cmd/athena serve
```

Start the gateway:

```bash
cd ../ashton-mcp-gateway
GATEWAY_HTTP_ADDR='127.0.0.1:18091' \
GATEWAY_MANIFEST_DIR='../ashton-proto/mcp' \
ATHENA_BASE_URL='http://127.0.0.1:18090' \
go run ./cmd/ashton-mcp-gateway
```

Verify health and discovery:

```bash
curl -sS http://127.0.0.1:18091/health
curl -sS -X POST http://127.0.0.1:18091/mcp/v1/tools/list \
  -H 'Content-Type: application/json' \
  --data '{}'
```

Run the routed occupancy call:

```bash
curl -sS -X POST http://127.0.0.1:18091/mcp/v1/tools/call \
  -H 'Content-Type: application/json' \
  --data '{"tool_name":"athena.get_current_occupancy","arguments":{"facility_id":"ashtonbee"}}'
```

Expected success shape:

- `facility_id`
- `current_count`
- `observed_at`
- `source_service=athena`
- `latency_ms`

Expected gateway success log:

```text
INFO gateway tool call tool_name=athena.get_current_occupancy source_service=athena facility_id=ashtonbee latency_ms=0 outcome=success
```

## Degraded Check

Stop ATHENA or point the gateway at a dead ATHENA port, then rerun the same
`tools/call` request.

Expected HTTP behavior:

- `502 Bad Gateway`
- clear connection or upstream failure text

Expected gateway failure log:

```text
INFO gateway tool call tool_name=athena.get_current_occupancy source_service=athena facility_id=ashtonbee latency_ms=0 outcome=upstream_error
```

## What This Runbook Does Not Prove

- caller identity
- persisted audit storage
- write approvals
- multi-service routing
- live cluster deployment
