// cliesp is a small CLI that appends new Espanso matches to a YAML file.
//
// Behavior:
//   - Prompts for triggers and a replacement text
//   - Appends a match entry to a target espanso match file
//
// Configuration (in order of precedence):
//  1. CLI flags: -m | --matchFile (directory or full path)
//  2. Environment variables / .env files (prefix: CLIESP_)
//     - CLIESP_MATCH_DIR, CLIESP_MATCH_FILE
//  3. Config file: ~/.config/cliesp/settings.{yaml|yml|toml|json}
//     - keys: match_dir, match_file
//  4. Defaults:
//     - dir:  ~/Library/Application Support/espanso/match
//     - file: cliesp.yml
//
// Single vs multiple triggers:
//   - Single:   - trigger: ":one"
//   - Multiple: - triggers: [":one", ":two"]
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	cfgpkg "github.com/kvnloughead/cliutils/config"
)

const (
	// Defaults if nothing is configured
	defaultEspansoMatchDir  = "~/Library/Application Support/espanso/match"
	defaultEspansoMatchFile = "cliesp.yml"
)

// AppConfig describes configurable fields for cliesp.
// Fields map to config file formats via tags and to env vars via `env` tags.
// Env prefix is derived from app name CLIESP_ by default by the loader.
type AppConfig struct {
	MatchDir  string `json:"match_dir" yaml:"match_dir" toml:"match_dir" env:"MATCH_DIR"`
	MatchFile string `json:"match_file" yaml:"match_file" toml:"match_file" env:"MATCH_FILE"`
}

func expandHome(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

// ensureFileWithHeader creates the file (and parent directories) if it does
// not exist. When creating, it writes a header that includes `matches:` as the
// root key required by espanso.
func ensureFileWithHeader(p string) error {
	// If file doesn't exist, create with header and root matches: key
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		f, err := os.Create(p)
		if err != nil {
			return err
		}
		defer f.Close()
		header := `# espanso match file (managed by cliesp)

# This file is generated and maintained by cliesp. For more information, see https://github.com/kvnloughead/cliesp.

# For information about espanso, visit the official docs at: https://espanso.org/docs/

matches:
`
		if _, err := f.WriteString(header); err != nil {
			return err
		}
	}
	return nil
}

// prompt writes a message to stdout and returns the user's input with trailing
// newline trimmed.
func prompt(s string) (string, error) {
	fmt.Print(s)
	r := bufio.NewReader(os.Stdin)
	text, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// buildYAMLSnippet returns a YAML fragment representing an espanso match
// entry. For a single trigger, the YAML uses `trigger:`; for multiple,
// it uses an inline list with `triggers:`.
func buildYAMLSnippet(triggers []string, replace string) string {
	var b strings.Builder
	b.WriteString("\n  - ")
	if len(triggers) == 1 {
		b.WriteString("trigger: ")
		// Quote if contains spaces or special chars; espanso examples show both quoted and unquoted.
		// We'll quote unless it's a simple :word pattern.
		b.WriteString(fmt.Sprintf("%q\n", triggers[0]))
	} else {
		b.WriteString("triggers: [")
		for i, t := range triggers {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%q", t))
		}
		b.WriteString("]\n")
	}
	b.WriteString("    replace: ")
	b.WriteString(fmt.Sprintf("%q\n", replace))
	return b.String()
}

// resolveMatchPath determines the final match file path using precedence:
// flagPath > env/config (via loader) > defaults. If only a directory is
// provided (no filename), default filename is used.
//
// When the flag path is a directory (ends with a separator or has no
// extension), the filename from the resolved configuration (or fallback
// defaults in this program) is appended. Tilde is expanded for both directory
// and file paths.
func resolveMatchPath(flagPath string, cfg AppConfig) (string, error) {
	// Determine base dir and file
	dir := cfg.MatchDir
	if dir == "" {
		dir = defaultEspansoMatchDir
	}
	file := cfg.MatchFile
	if file == "" {
		file = defaultEspansoMatchFile
	}
	// If flagPath is set, parse it; if it ends with a path separator or has no extension treat as dir
	if flagPath != "" {
		p := flagPath
		if strings.HasPrefix(p, "~") {
			expanded, err := expandHome(p)
			if err != nil {
				return "", err
			}
			p = expanded
		}
		// If p ends with a separator, assume directory
		if len(p) > 0 && os.IsPathSeparator(p[len(p)-1]) {
			return filepath.Join(p, file), nil
		}
		// If it looks like a file (has an extension), use it directly
		if filepath.Ext(p) != "" {
			return p, nil
		}
		// Otherwise, treat as directory
		return filepath.Join(p, file), nil
	}
	// No flag override — use cfg/defaults
	if strings.HasPrefix(dir, "~") {
		d, err := expandHome(dir)
		if err != nil {
			return "", err
		}
		dir = d
	}
	return filepath.Join(dir, file), nil
}

func main() {
	// Preload .env files from the current working directory to ensure env
	// variables are available via process environment even if file-based
	// loading is skipped. Missing files are ignored by godotenv.Load.
	_ = godotenv.Load(".env", ".env.local", ".env.production")

	// Flags
	var matchFlag string
	flag.StringVar(&matchFlag, "matchFile", "", "Path to the espanso match file (overrides config)")
	flag.StringVar(&matchFlag, "m", "", "Path to the espanso match file (shorthand)")
	// Allow intermixing flags and prompts
	flag.Parse()

	// Load config from files/env via cliutils/config
	cfg, err := cfgpkg.Load(cfgpkg.Options[AppConfig]{
		AppName: "cliesp",
		ConsumerConfig: AppConfig{
			MatchDir:  defaultEspansoMatchDir,
			MatchFile: defaultEspansoMatchFile,
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading config:", err)
		os.Exit(1)
	}

	// Resolve final match path using precedence: flag > env/config > defaults
	filePath, err := resolveMatchPath(matchFlag, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error resolving match file path:", err)
		os.Exit(1)
	}

	if err := ensureFileWithHeader(filePath); err != nil {
		fmt.Fprintln(os.Stderr, "error preparing file:", err)
		os.Exit(1)
	}

	triggersLine, err := prompt("triggers? (space separated list of strings): ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading triggers:", err)
		os.Exit(1)
	}
	var triggers []string
	for _, part := range strings.Fields(triggersLine) {
		p := strings.TrimSpace(part)
		if p != "" {
			triggers = append(triggers, p)
		}
	}
	if len(triggers) == 0 {
		fmt.Fprintln(os.Stderr, "no triggers provided, exiting")
		os.Exit(1)
	}

	replaceStr, err := prompt("replace with? (string): ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading replace string:", err)
		os.Exit(1)
	}

	entry := buildYAMLSnippet(triggers, replaceStr)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error opening file for append:", err)
		os.Exit(1)
	}
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil {
		fmt.Fprintln(os.Stderr, "error writing entry:", err)
		os.Exit(1)
	}
	fmt.Printf("Appended %d trigger(s) to %s\n", len(triggers), filePath)
}
