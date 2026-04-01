# Growing Pains

Use this document to record real routing issues, protocol mistakes, approval-path
bugs, and the fixes that made the gateway more predictable and auditable.

## 2026-04-01

- The gateway almost became a premature platform centerpiece before the services
  had real read surfaces. The fix was to defer it and document Go-first routing
  as a later tracer after service interfaces exist.
