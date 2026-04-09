# Growing Pains

Use this document to record real routing issues, protocol mistakes, approval-path
bugs, and the fixes that made the gateway more predictable and auditable.

## 2026-04-01

- Symptom: the gateway almost became a premature platform centerpiece before
  the services had real read surfaces.
  Cause: the control-plane architecture was clearer than the underlying service
  truth, which made it easy to design ahead of the platform.
  Fix: defer the repo until routed service surfaces were real and document
  Go-first routing as a later tracer.
  Rule: the gateway only earns executable scope after the services have
  something worth routing.

## 2026-04-03

- Symptom: the first route kept wanting to turn into a broad generic control
  plane before one manifest-backed read path was proven.
  Cause: the repo brief carried a lot of deferred future architecture, and it
  was easy to start designing for caller identity, audit persistence, and
  multi-service routing too early.
  Fix: lock Tracer 9 to one manifest, one ATHENA occupancy route, and one
  inspectable log line.
  Rule: discovery and routing must be proven before the gateway earns broader
  control-plane layers.

- Symptom: local `go build ./cmd/ashton-mcp-gateway` left an untracked repo-root
  binary behind after smoke runs.
  Cause: Go writes the compiled binary in the current working directory by
  default.
  Fix: ignore `/ashton-mcp-gateway` in `.gitignore`.
  Rule: first-route smoke should leave behind evidence, not confusing local
  build noise.

## 2026-04-08

- Symptom: the first Tracer 15 pass could have treated trusted caller headers
  as raw user input and turned caller attribution into spoofable theater.
  Cause: the gateway had no prior identity boundary, so the cheapest possible
  implementation path was to trust headers directly.
  Fix: split caller identity into two explicit narrow paths:
  `X-Gateway-Trusted-Caller-Token` plus caller headers for trusted internal
  boundaries, and `X-Gateway-API-Key` for configured automation callers.
  Rule: caller identity should be explicit, narrow, and testable; it should not
  quietly become an auth-platform rewrite.

- Symptom: the first audit closeout path returned a typed-nil `*ToolCallError`
  on successful routed calls, which made handler code treat success like a
  failure.
  Cause: the fail-closed audit helper returned the typed error value directly
  even when it was nil.
  Fix: return a real `nil` after successful audit persistence and keep audit
  failure as its own explicit error kind.
  Rule: fail-closed audit logic must keep success and failure shapes legible in
  both tests and handlers.
