# ADR 001: Go First, Rust Later

## Status

Accepted.

## Context

The gateway may eventually benefit from Rust, but the first implementation wave
is intentionally narrow: discover one tool, route one read-only call, and prove
the gateway pattern at all.

## Decision

Implement the first gateway in Go. Consider a Rust rewrite only after:

- the gateway has real usage
- routing or concurrency bottlenecks are measured
- the rewrite has a specific, defensible payoff

## Why

- Go matches the rest of the first-wave platform
- Go keeps iteration speed high
- there is no proven hot path yet
- rewriting a measured bottleneck is stronger engineering than speculative polyglot complexity

## Consequences

- the first gateway ships faster
- Rust remains available as a deliberate optimization path
- the platform avoids premature language fragmentation
