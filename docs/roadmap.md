# ashton-mcp-gateway Roadmap

## Objective

Create a single narrow control surface for routed tool discovery and invocation
only after the underlying service surfaces are worth routing.

## Current Line

Current shipped line: `v0.2.0`

Current released line: `v0.2.0`

- one executable Go gateway runtime is real
- shared manifest loading is real
- caller identity for routed calls is real
- persisted audit for routed calls is real
- two routed read-only ATHENA occupancy calls are real
- write approvals, rate limiting, and broader routing are still deferred

## Versioning Discipline

The gateway now follows formal pre-`1.0.0` semantic versioning.

- `PATCH` releases cover hardening, docs sync, deployment closeout,
  observability, and bounded non-widening fixes
- `MINOR` releases cover new routed capabilities, new trust boundaries, or
  intentional contract changes
- pre-`1.0.0` breaking changes still require a `MINOR`, never a `PATCH`

## Planned Release Lines

| Planned tag | Intended purpose | Restrictions | What it should not do yet |
| --- | --- | --- | --- |
| `v0.3.0` | first write approval and HITL line | add explicit human approval for write calls only after the read path is trusted | do not widen into rate limiting or full multi-service orchestration in the same line |
| `v0.4.0` | rate limiting and broader registry line | expand only after the gateway already has real read and write proof | do not justify a Rust rewrite without a measured Go bottleneck |

## Current Ladder Role

| Line | Role | Why it matters |
| --- | --- | --- |
| `Tracer 15` | caller identity, persisted audit, and one second routed read | turns the gateway from a first routed proof into a trusted narrow control-plane layer |
| `v0.3.0` | first write approval and HITL line | adds explicit write governance only after the read path is trusted |
| `v0.4.0` | broader registry and rate limiting | widens the control plane only after read and write proof already exist |

## Boundaries

- keep `tools/list` open and narrow
- keep `tools/call` as the explicit identity and audit boundary
- keep the routed slice read-only
- keep APOLLO, HERMES, approvals, rate limiting, and Rust for later lines
- the gateway only becomes useful after service surfaces are already real

## Tracer / Workstream Ownership

- `Tracer 9`: first Go gateway bootstrap, first manifest, first routed read-only
  call, first inspectable logs
- `Tracer 15`: caller identity, persisted audit, second routed read
- later lines: first write approval, then rate limiting and broader registry
