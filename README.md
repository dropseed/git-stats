# git-stats

A lightweight tool to store metrics and stats directly in your git repo as [git notes](https://git-scm.com/docs/git-notes).

## Install

```console
$ curl -sSL https://raw.githubusercontent.com/dropseed/git-stats/master/install.sh | sh
```

Or install with Go:

```console
$ go install github.com/dropseed/git-stats@latest
```

The binary is named `git-stats`, so it works as a git subcommand:

```console
$ git stats --help
```

## Quick start

Create a `git-stats.yml` in your repo:

```yaml
# git-stats.yml
stats:
  todos:
    run: |
      grep "TODO" -r app -c | awk -F: '{sum+=$NF} END {print sum}'
  coverage:
    type: "%"
    default: 0%
    goal: increase
    run: |
      coverage report | tail -n 1 | awk '{print $4}'
  loc:
    run: |
      find . -name "*.go" | xargs wc -l | tail -1 | awk '{print $1}'
```

Each stat has a `run` command that outputs a single number (or percentage).

Optional fields:
- `type` — `"number"` (default) or `"%"` (strips trailing `%` for comparisons)
- `default` — value to use when a commit has no stat recorded (default: `0`)
- `goal` — `"increase"` or `"decrease"` — reports a GitHub commit status pass/fail based on whether the value moved in the right direction

Check your stats on the working directory:

```console
$ git stats check
Generating value for todos: 42
Generating value for coverage: 87%
Generating value for loc: 1203
```

Save stats for the current commit:

```console
$ git stats save
```

## CI usage

The `ci` command is an all-in-one that fetches existing stats, saves stats for the current commit, regenerates any missing stats for the last 10 commits, and pushes:

```yaml
name: CI

on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 10
      # ... run your tests ...
      - name: Install git-stats
        run: curl -sSL https://raw.githubusercontent.com/dropseed/git-stats/master/install.sh | sh
      - run: git stats ci
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

The `fetch-depth: 10` is needed so `regen` can access recent commits.

Setting `GITHUB_TOKEN` enables commit status reporting for stats with a `goal` configured.

You can also add sparklines to the GitHub Actions job summary:

```yaml
      - run: |
          git stats ci

          echo "## Commit Stats" >> "$GITHUB_STEP_SUMMARY"
          echo '```' >> "$GITHUB_STEP_SUMMARY"
          git stats log --format sparklines -- --reverse >> "$GITHUB_STEP_SUMMARY"
          echo '```' >> "$GITHUB_STEP_SUMMARY"
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Commands

| Command | Description |
|---------|-------------|
| `check` | Run stat commands and print values (dry run) |
| `save` | Run stat commands and save as git notes on HEAD |
| `show [commit]` | Display saved stats for a commit |
| `log [-- git log args]` | Show stats across commits (pretty, table, tsv, csv, json, sparklines) |
| `regen [-- git log args]` | Check out old commits and regenerate stats |
| `fetch` | Fetch stats from remote |
| `push` | Push stats to remote |
| `delete` | Remove a stat key from a commit |
| `clear` | Delete all stats |
| `ci` | All-in-one: fetch, save, regen, push |

## Viewing stats

The default `pretty` format shows an aligned table with short hashes, plus sparklines when there are 3+ commits:

```console
$ git stats log -- -n 10
commit  todos  coverage    loc
abc1234    42        87   1203
def5678    41        86   1180
...

todos (min 0, max 43, avg 25.4)
▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▇▇▇███████████████████████████████
```

Other formats:

```console
$ git stats log --format table -- -n 10      # aligned table only
$ git stats log --format sparklines -- -n 50  # sparklines only
$ git stats log --format csv -- -n 20         # machine-readable CSV
$ git stats log --format tsv -- -n 20         # machine-readable TSV
$ git stats log --format json -- -n 10        # JSON array
```

Pass any `git log` arguments after `--`:

```console
$ git stats log -- -n 20 --reverse --author="dave"
```

## Global flags

| Flag | Description |
|------|-------------|
| `--config <path>` | Path to config file (default: `git-stats.yml` in repo). Available on `check`, `save`, `regen`, and `ci`. |
| `-y`, `--yes` | Skip confirmation prompts. Useful for scripts and CI. |

## Retroactive stats

Generate stats for existing commits:

```console
$ git stats regen -- -n 50
```

Only fill in commits that are missing stats:

```console
$ git stats regen --missing-only -- -n 50
```

Your working tree must be clean (no uncommitted changes to tracked files) to use regen.
Untracked files are preserved through checkouts.

### First-time setup

When adding git-stats to an existing repo, your config file and any scripts it
references won't exist in older commits. Use `--config` to point at a config
file outside the checkout, and `--keep` to preserve files through checkouts:

```console
# Copy config somewhere safe and regen with it
$ cp git-stats.yml /tmp/git-stats.yml
$ git stats regen --config /tmp/git-stats.yml -- -n 50

# Or keep scripts and config in place during checkouts
$ git stats regen --keep scripts/bench,git-stats.yml -- -n 50
```
