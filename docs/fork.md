# Maintained Sponge fork

The maintained branch is `thruster_generate` in `Eric-Guo/sponge`. This fork builds on [go-dev-frame/sponge](https://github.com/go-dev-frame/sponge); upstream authorship and the MIT license are preserved.

## Install and update

Use Go 1.27.1 or newer. Install protoc and add your Go binary directory to PATH.

```sh
go install github.com/Eric-Guo/sponge/cmd/sponge@thruster_generate
sponge init
sponge --version
# sponge version v1.15.1 https://github.com/Eric-Guo/sponge fork
```

`sponge init` and `sponge upgrade` resolve `thruster_generate` and install the binary, templates, and built-in Protobuf plugins from the same Go module revision. `sponge upgrade --version=<revision>` can pin a specific fork revision. Run `sponge init` when migrating an existing upstream installation so `~/.sponge` contains fork templates.

The CLI display version is `v1.15.1`. Dependency versions are separate: Go resolves the branch to a canonical pseudo-version. Do not use an upstream release tag to select this fork. `--version` works without initialized templates and does not depend on the cached template version.

## Migrate a service

1. Replace imports of `github.com/go-dev-frame/sponge` with `github.com/Eric-Guo/sponge`, including tests and Protobuf `go_package` options.
2. Remove the old Sponge requirement and any local Sponge `replace` directive from `go.mod`.
3. Resolve the maintained branch and refresh checksums:

   ```sh
   go get github.com/Eric-Guo/sponge@thruster_generate
   go mod tidy
   go test -count=1 -short ./...
   go build ./...
   ```

Commit the resulting canonical version in `go.mod` and checksums in `go.sum`. Builds then fetch the fork directly and need no local checkout. Regenerate Protobuf files when their `go_package` option changes.

Generated services import this fork. Installed templates record the resolved module version in `~/.sponge/.github/version`; a source checkout without that file uses `thruster_generate`, resolved by `go mod tidy`.

## Local validation

```sh
go build -o /tmp/sponge ./cmd/sponge
go test -count=1 -short ./...
SPONGE_TEST_GENERATED_BUILD=1 go test -count=1 ./cmd/sponge/commands/generate -run TestHTTPGenerationUsesRepositoryTemplates
```

The last command generates SQLite HTTP services with both model styles and builds them against this checkout in temporary directories.
