# Release Notes

## v0.1.0-alpha (squids bootstrap)

### Highlights

- Introduced `sq` CLI with beads-compatible task workflow foundation.
- Added SQLite storage layer with schema initialization and migrations.
- Enabled concurrent access-friendly SQLite settings:
  - WAL mode
  - busy timeout
- Implemented core task commands:
  - `init`, `ready`, `create`, `show`, `list`, `update`, `close`
- Added parity shell automation suite and dual-target runner (`bd` vs `sq`).

### Compatibility

`sq` intentionally does **not** implement dolt/server mechanics.
Task command behavior parity is validated through shell automation.

### Known limits (alpha)

- Coverage currently focuses on core command/flag behaviors in parity suite.
- Additional advanced beads flags/flows may be added incrementally with test-first expansion.
