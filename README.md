# ashton-mcp-gateway

The unified tool gateway for ASHTON.

This repo will eventually register tool manifests from the service repos, route tool calls, enforce approval on write operations, and centralize audit visibility. It is intentionally later in the build order because it only becomes valuable after the first service surfaces are real.

This repo is docs-first for now. The detailed brief lives in [ashton-platform/planning/repo-briefs/ashton-mcp-gateway.md](https://github.com/ixxet/ashton-platform/blob/main/planning/repo-briefs/ashton-mcp-gateway.md).

## Role In The Platform

- shared tool gateway over the service repos
- depends on the contract repo plus real service surfaces
- centralizes auth, approval, and audit behavior later

## First Execution Goal

Only start this repo once the first service interfaces exist. The first useful slice is:

- discover one tool manifest
- route one read-only tool call
- defer write approval flows until read routing is stable

## Current State

Docs-first stub only. No Go or Rust implementation scaffold has been created yet.

See:

- `docs/roadmap.md`
- `docs/runbooks/first-route.md`
- `docs/growing-pains.md`
