# Milestone 2.0 Hardening

Milestone 2.0 does not widen the gateway into new routed services or writes. It
hardens the existing caller-aware read boundary.

## Scope

- constant-time comparison for trusted caller secrets
- undeclared and wrong-type optional argument rejection
- symlink/path hardening for the manifest directory
- bounded JSON decoding for `POST /mcp/v1/tools/call`
- no change to audit policy: persisted-audit failure remains fail-closed

## Proof

Run from `/Users/zizo/Personal-Projects/ASHTON/ashton-mcp-gateway`:

```sh
go test ./...
go test -count=5 ./internal/...
go test -race ./internal/...
go vet ./...
go build ./...
git diff --check
```

Focused destructive coverage now includes:

- bad trusted caller token fails cleanly
- bad API key fails cleanly
- undeclared arguments fail cleanly
- wrong-type optional arguments fail cleanly
- oversized and unknown-field tool-call request bodies fail cleanly
- manifest symlink directory/file escapes fail cleanly
- audit persistence failure still returns `audit_failure`

## Negative Proof

Milestone 2.0 does not claim:

- routed writes
- HITL approval
- APOLLO or HERMES routing
- gateway deployment proof

## Truth Split

- local/runtime truth: the current routed read boundary is stricter and harder
  to misuse on the `v0.2.1` patch line
- deployed truth: unchanged and still deferred
- deferred truth: routed writes, approvals, rate limiting, and live deployment
