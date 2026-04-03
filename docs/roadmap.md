# ashton-mcp-gateway Roadmap

## Objective

Create a single narrow control surface for routed tool discovery and invocation
only after the underlying service surfaces are worth routing.

## Current Line

Current active line: `v0.1.0`

- one executable Go gateway runtime is real
- one manifest loader is real
- one routed read-only ATHENA occupancy call is real
- approval, persisted audit, caller identity, and multi-service routing are
  still deferred

## Planned Release Lines

| Planned tag | Intended purpose | Restrictions | What it should not do yet |
| --- | --- | --- | --- |
| `v0.2.0` | caller identity, persisted audit, and second routed read for Tracer 15 | keep the gateway read-only while adding stronger auditability | do not add write approvals yet |
| `v0.3.0` | first write approval and HITL line | add explicit human approval for write calls only after the read path is trusted | do not widen into rate limiting or full multi-service orchestration in the same line |
| `v0.4.0` | rate limiting and broader registry line | expand only after the gateway already has real read and write proof | do not justify a Rust rewrite without a measured Go bottleneck |

## Boundaries

- Tracer 9 is not broad orchestration
- the first gateway slice is one manifest, one routed read-only call, and
  inspectable logs
- approval, rate limiting, and Rust remain later lines
- the gateway only becomes useful after service surfaces are already real

## Tracer / Workstream Ownership

- `Tracer 9`: first Go gateway bootstrap, first manifest, first routed read-only
  call, first inspectable logs
- `Tracer 15`: caller identity, persisted audit, second routed read
- later lines: first write approval, then rate limiting and broader registry
