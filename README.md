# cliesp

A small Go CLI to append entries to an [Espanso](https://espanso.org) match file.

## Installation

Download the [most recent release](https://github.com/kvnloughead/cliesp/releases/new) and move the downloaded binary to somewhere in your `PATH`.

## Basic Usage

Simply run `cliesp` to enter interactive mode. You'll be prompted for

- a space-separated list of triggers
- what to replace them with

A new match entry will be added to the configured file (which will be created if it doesn't exist). By default, the location is

- `~/Library/Application Support/espanso/match/cliesp.yml`

The match will use a `triggers` array if multiple values are provided. Otherwise it uses `trigger`.

## Configuration

The app can be configured via a config file `~/.config/cliesp/settings.{yaml|yml|toml|json}`. Configurable settings:

```yaml
match_dir: ~/Library/Application Support/espanso/match
match_file: cliesp.yml
file_opener: open # On windows 'explorer', on Linux 'xdg-open'
dir_opener: vim # EDITOR environmental variable or vim
```

You can also configure via environmental variables with the `CLIESP_` prefix. Environmental variables take precedence over values from the configuration file. For development, you can use a `.env` file inside the local repo. The following env files are loaded (in this order): `.env`, `.env.local`, `.env.production`

## CLI Flags

Flags take precedence over all configured settings.

- `-m` or `--matchFile` to set the match file path. You can provide either:
  - A directory path (the configured/default filename will be used)
  - A full file path (directory + filename)
- `-o` or `--open` to open the resolved match file and exit (no prompting)
- `-d` or `--openDir` to open the resolved match directory and exit (no prompting)
  - `--open` and `--openDir` are mutually exclusive
  - On macOS this uses `open`, on Linux `xdg-open`, on Windows `explorer`

## Installation from source

```
git clone https://github.com/kvnloughead/cliesp
cd cliesp
go build -o cliesp
```

