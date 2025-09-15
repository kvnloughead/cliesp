# cliesp

A small Go CLI to append entries to an [Espanso](https://espanso.org) match file.

Currently hard-coded to write to:

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

## Notes

- The destination path is currently hard-coded in `main.go`. You can change `espansoMatchDir` and `espansoMatchFile` to suit your setup.
- YAML quoting: triggers and replace are quoted to be safe with spaces and special characters.
