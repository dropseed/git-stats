# git-stats

CLI tool that stores per-commit metrics as git notes. Binary name is `git-stats` (so `git stats` works as a git subcommand).

## Build

```sh
go build -o git-stats .
```

Or run directly without building: `go run . check`

## Test

```sh
scripts/test
```

System test that builds the binary and exercises all commands in an isolated temp git repo.

## Lint and format

```sh
scripts/lint      # gofmt check + go vet + golangci-lint
scripts/format    # gofmt -w
```

## Project structure

```
main.go                     # entry point, calls cmd.Execute()
internal/
  cmd/                      # cobra commands (one file per command)
    root.go                 # root command + version
    check.go save.go show.go log.go regen.go
    fetch.go push.go delete.go clear.go ci.go
  config/config.go          # YAML config parsing (preserves key order)
  git/git.go                # git subprocess helpers (Exec, Output, RunShell)
  notes/notes.go            # git notes operations
  stats/stats.go            # stat types, parsing, sparklines
  github/github.go          # GitHub commit status API for --goal
```

## Dependencies

Only two external: `github.com/spf13/cobra` and `gopkg.in/yaml.v3`.

## Notes

- Git log args are passed after `--` (e.g., `git stats log -- -n 5`)
- Config file is `git-stats.yml`
- Version is set via ldflags at release time (`internal/cmd.version`)
- Repo will be renamed to `dropseed/git-stats` (module path already updated)
