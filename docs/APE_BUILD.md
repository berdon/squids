# APE / Cosmopolitan Build Notes

`sq` supports a best-effort APE artifact build for release pipelines.

## Local build

```bash
make ape-build
```

This runs `scripts/build-ape.sh`.

## Behavior

- If both `cosmocc` and `ape` are available on PATH:
  - builds `dist/sq-ape` (Cosmopolitan/APE-wrapped artifact)
  - writes `dist/sq-ape.README.txt`
- If tooling is unavailable:
  - builds `dist/sq-ape-fallback-linux-amd64`
  - writes `dist/sq-ape.UNAVAILABLE.txt` explaining fallback

## CI / release integration

- CI runs `scripts/build-ape.sh` to validate the path.
- Release workflow publishes `dist/sq-ape*` alongside standard platform binaries.

## Fallback strategy

If host/kernel/tooling cannot run or produce APE binaries:
- use native release artifacts (`sq-linux-amd64`, `sq-darwin-arm64`, etc.)
- or run `scripts/build-ape.sh` in an environment where `cosmocc` + `ape` are installed.
