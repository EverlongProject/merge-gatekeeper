# merge-gatekeeper

League fork of Merge Gatekeeper. It is a Go CLI that runs as a Docker GitHub
Action (`action.yml`). It fails until the other checks on a PR ref pass.

## Build and test

CI (`.github/workflows/build-ci.yaml`) uses Go 1.26.1 and runs these commands:

- Build all packages: `go build ./...`
- Run all tests: `go test ./...` (`make test` runs the same command)

- CI has no lint step.
- Build the binary: `CGO_ENABLED=0 go build .` (from `Dockerfile`). It writes
  `merge-gatekeeper` at the repo root, and `.gitignore` ignores that file.
- Do not use `make go-build`. It builds `./cmd/merge-gatekeeper`, which does
  not exist, so it fails with "directory not found". `make go-run` depends on
  it and fails too.

## Files

- `main.go` is the entry point. It embeds `version.txt` as the version string.
- `internal/cli` holds the cobra command and its flags (`validate.go`).
- `internal/validators/status` holds the check logic.
- `internal/github` wraps the go-github client behind the `Client` interface.
- Tests are `*_test.go` files next to the code they test.
- `internal/github/mock/mock.go` and `internal/validators/mock/mock.go` are
  hand-written. No `go:generate` line makes them. When you change the `Client`
  interface in `internal/github/github.go`, update the mock in the same change.

## Conventions

### Generated docs

- The parts of `README.md` between the `imptr` begin and end comments come
  from `export` blocks in `docs/details.md`, `docs/action-usage.md`, and
  `example/definitions.yaml`. Edit the `export` block in the source file. Do
  not edit those parts of `README.md`.
- `.github/workflows/documentation-ci.yaml` runs on PRs that change `*.md`
  files. It installs the tool with
  `go install github.com/upsidr/importer/cmd/importer@v0.1.4`, then runs
  `find . -name '*.md' -exec importer update {} \;`. The job fails if the tree
  changes. Run the same two commands after you change a source block, and
  commit the result.
- `example/merge-gatekeeper.yml` is the `standard-setup` block of
  `example/definitions.yaml` without its comments. No tool keeps the two files
  the same. Change both.

### Action inputs

An action input must agree in these places:

1. `action.yml`: the `inputs` entry and the `runs.args` flag.
2. `internal/cli/validate.go`: the flag definition.
3. `docs/action-usage.md`: the inputs table in the `inputs` export block. Then
   run importer to update `README.md`.

## CI behavior

- `build-ci.yaml` runs on a PR only when `main.go`, `version.txt`,
  `internal/**`, `go.mod`, or `go.sum` change.
- `.github/workflows/merge-gatekeeper-latest.yaml` runs the action from the PR
  checkout (`uses: ./`) on each PR to `main`. Docker builds the PR's own
  `Dockerfile`. A broken `Dockerfile` or `action.yml` fails the Merge
  Gatekeeper check on the same PR.

## Sandbox and network

- The first build downloads modules from `proxy.golang.org` and checks them
  with `sum.golang.org` (dependencies in `go.mod`, default `GOPROXY`). The
  importer install uses the same hosts.
- Go writes its build cache and module cache outside the repo
  (`go env GOCACHE`, `go env GOMODCACHE`). If the sandbox blocks these paths,
  or blocks TLS to these hosts, ask the engineer to run the Go command or to
  approve it outside the sandbox.
- `make docker-build` and `make docker-run` need the Docker daemon. Ask the
  engineer to run them.
- `make go-run` and `make docker-run` call the live GitHub API with
  `GITHUB_TOKEN`. The `Makefile` defaults point them at
  `EverlongProject/merge-gatekeeper` at `main`. Ask the engineer before you
  run them.

## Safe git

- PRs target `main`. `build-ci.yaml` also accepts PRs to `release/v*`.
- A push to `main` or a `v*` tag runs `build-ci.yaml`. No workflow in this
  repo deploys or publishes.
- Consumer repos use `EverlongProject/merge-gatekeeper@v1`. The `v1` tag moves
  to each new v1.x.y release (`README.md`). A new or moved tag changes the
  merge gate in each consumer repo. Do not create, move, or push tags.
