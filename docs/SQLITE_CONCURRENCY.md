# SQLite Concurrency Strategy (sq-011)

## Goals

Support safe multi-process access to a single SQLite task database file without external server components.

## Strategy

1. **WAL mode enabled**
   - Configure `PRAGMA journal_mode=WAL` at open time.
   - Allows readers during writer activity and improves concurrent workflow behavior.

2. **Busy timeout configured**
   - Configure `PRAGMA busy_timeout=5000` and DSN busy timeout.
   - Reduces immediate lock failures under brief contention.

3. **Short write transactions**
   - Keep create/update/close writes as short, bounded operations.
   - Avoid long-lived transactions around command-level orchestration.

4. **Deterministic command boundaries**
   - Each CLI command owns a concise open/init/read/write/close cycle.
   - No background daemon/server lock ownership.

5. **Test for concurrent process behavior**
   - Add shell automation that runs many `sq create` in parallel against the same DB.
   - Verify all commands complete and resulting list count is at least expected.

## Future hardening

- Add retry-with-backoff wrappers for known lock errors in write paths.
- Add concurrent update-on-same-record tests.
- Add stress tests with mixed create/update/close operations and assertion of DB consistency invariants.
