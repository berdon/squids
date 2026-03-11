# squids

SQLite-backed, beads-compatible task CLI (planned).

## Parity Shell Automation (sq-001)

This repo starts with a black-box parity suite that can run against an existing CLI target
(currently `bd`, later `squids`).

### Run against beads

```bash
./scripts/parity/run-parity.sh
```

### Run against another binary

```bash
TARGET_BIN=./bin/squids ./scripts/parity/run-parity.sh
```

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
