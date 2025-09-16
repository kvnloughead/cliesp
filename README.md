# cliesp

A small Go CLI to append entries to an [Espanso](https://espanso.org) match file.

By default it writes to:

- `~/Library/Application Support/espanso/match/cliesp.yml`

## What it does

When you run the program it prompts you twice:
  - `triggers? (space separated list of strings)`
  - `replace with? (string)`

Then it appends a YAML match entry under the `matches:` root of the match file.

For a single trigger, it writes:
    ```yaml
    - trigger: ":example"
      replace: "Hello"
    ```
For multiple triggers, it writes:
    ```yaml
    - triggers: [":ex1", ":ex2"]
      replace: "Hello"
    ```

If the file doesn't exist, it creates it with the `matches:` header.

## Installation

```
git clone https://github.com/kvnloughead/cliesp
cd cliesp
go build -o cliesp
```

Once built, move the binary `cliesp` to a folder in your PATH.

## Run

```
./cliesp
```

Example session:

```
triggers? (space separated list of strings): :ex1 :ex2
replace with? (string): Hello world
Appended 2 trigger(s) to /Users/<you>/Library/Application Support/espanso/match/cliesp.yml
```

## Configuration

You can change the destination path and filename via:

1) Config file (recommended)

- Location: `~/.config/cliesp/settings.{yaml|yml|toml|json}`
- Keys:
  - `match_dir` (string) — directory path. `~` is supported.
  - `match_file` (string) — filename (e.g., `cliesp.yml`).

Example `~/.config/cliesp/settings.yaml`:

```yaml
match_dir: ~/Library/Application Support/espanso/match
match_file: cliesp.yml
```

Another example with a custom directory and filename:

```yaml
match_dir: ~/espanso/match
match_file: personal.yml
```

2) Environment variables and .env files

- Prefix: `CLIESP_`
- Variables:
  - `CLIESP_MATCH_DIR`
  - `CLIESP_MATCH_FILE`
- .env files (loaded in this order by default): `.env`, `.env.local`, `.env.production`

Example `.env`:

```
CLIESP_MATCH_DIR=/tmp/espanso/match
CLIESP_MATCH_FILE=my.yml
```

3) CLI flags (highest precedence)

- `-m` or `--matchFile` to set the match file path. You can provide either:
  - A directory path (the configured/default filename will be used)
  - A full file path (directory + filename)

Precedence: flag > env/.env > config file > defaults.

### Example espanso match file (YAML)

When entries are appended, they go under the `matches:` key. A small example of the target file:

```yaml
# espanso match file

# For a complete introduction, visit the official docs at: https://espanso.org/docs/

# You can use this file to define the base matches (aka snippets)
# that will be available in every application when using espanso.

# Matches are substitution rules: when you type the "trigger" string
# it gets replaced by the "replace" string.
matches:
  # Date example
  - trigger: ":date"
    replace: "{{mydate}}"
    vars:
      - name: mydate
        type: date
        params:
          format: "%m/%d/%Y"

  # Single trigger
  - trigger: ":hello"
    replace: "Hello world"

  # Multiple triggers
  - triggers: [":ex1", ":ex2"]
    replace: "Example replacement"
```

## Testing

Run the test suite:

```
go test ./...
```

## Notes

- YAML quoting: triggers and replace strings are quoted to be safe with spaces and special characters.
