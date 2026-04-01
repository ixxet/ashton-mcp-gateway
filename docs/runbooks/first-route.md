# Gateway First Route Runbook

## Purpose

Use this runbook for the first gateway tracer once at least one service has a real read-only surface.

## Rules

- load one manifest
- route one read-only call
- log the path clearly
- do not expand to broad orchestration until the first route is stable

## Required Checks

- one manifest is discoverable
- one read-only call routes end to end
- audit data captures who called what
