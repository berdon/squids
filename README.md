# squids

SQLite-backed, beads-compatible task CLI (planned).

Design constraints:
- Binary name is `sq`
- No dolt/server mechanics
- Concurrency supported via single-file SQLite backend (multi-process safe)

## Build

```bash
make build
./bin/sq --help
```

## Parity Shell Automation (sq-001)

This repo starts with a black-box parity suite that can run against an existing CLI target
(currently `bd`, later `squids`).

### Run against beads

```bash
./scripts/parity/run-parity.sh
```

### Run against squids binary (`sq`)

```bash
TARGET_BIN=./bin/sq ./scripts/parity/run-parity.sh
```

### Dual-target compatibility run (beads vs sq)

```bash
./scripts/parity/compat-runner.sh
```

Outputs logs and deltas under `.parity-results/`.

### SQLite concurrency smoke test

```bash
./scripts/parity/concurrency-smoke.sh
```

## Docs

- Compatibility contract: `docs/COMPATIBILITY_CONTRACT.md`
- Migration guide: `docs/MIGRATION_FROM_BEADS.md`
- Release notes: `docs/RELEASE_NOTES.md`

### Optional target args

If your target needs global args before the command:

```bash
TARGET_ARGS="--json" ./scripts/parity/run-parity.sh
```

The harness spins up an isolated temp workspace per run and validates:
- init + ready
- create/show/list/update/close lifecycle
- json contracts
- label/dependency metadata behavior
- selected error semantics
