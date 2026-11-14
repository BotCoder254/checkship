# compose (patched)

Go sources of `github.com/YuukanOO/compose/v2 v2.27.2-0.20240821071151-110203bb5f25`, the Docker Compose fork checkship already used (it skips network aliases on external networks, see `go.mod`). Tests, docs and packaging were left out. Licensed under Apache-2.0, see `LICENSE` and `NOTICE`.

`go.mod` at the repository root replaces `github.com/docker/compose/v2` with this directory.

## checkship patch

- `pkg/compose/build.go`, `pkg/compose/build_output.go`: BuildKit build progress is printed to the docker CLI output instead of always to `os.Stdout`, so checkship can record image builds in deployment logs.

Search for `checkship patch` to find every change. When moving to another compose version, re-apply them or drop this directory if upstream prints builds to the CLI output.
