# ashton-mcp-gateway Roadmap

## Objective

Create a single safe entry point for tool discovery and invocation after the service repos expose real surfaces worth routing to.

## First Implementation Slice

- load one tool manifest from `ashton-proto`
- route one read-only call to a real service
- log the invocation path clearly
- keep write approval behavior designed but deferred

## Boundaries

- do not start with a broad multi-service control plane
- do not build the Rust rewrite before the Go shape is justified by real usage
- do not pretend the gateway is useful before the services exist

## Exit Criteria

- one tool can be discovered
- one read-only call can be routed end to end
- the gateway remains narrow, inspectable, and worth expanding later

## Tracer Ownership

- later tracer: first routed read-only tool call after service repos expose stable surfaces
