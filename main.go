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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
	cfgpkg "github.com/kvnloughead/cliutils/config"
)

const (
	// Defaults if nothing is configured
	defaultEspansoMatchDir   = "~/Library/Application Support/espanso/match"
	defaultEspansoMatchFile  = "cliesp.yml"
	defaultMultilineMode     = "messaging"

	// Multiline input modes
	multilineModeMessaging = "messaging" // Shift+Enter for newline, Enter submits
	multilineModeEOF       = "eof"       // EOF/Ctrl+D to submit
)

// AppConfig describes configurable fields for cliesp.
// Fields map to config file formats via tags and to env vars via `env` tags.
// Env prefix is derived from app name CLIESP_ by default by the loader.
type AppConfig struct {
	MatchDir  string `json:"match_dir" yaml:"match_dir" toml:"match_dir" env:"MATCH_DIR"`
	MatchFile string `json:"match_file" yaml:"match_file" toml:"match_file" env:"MATCH_FILE"`
	// Optional commands to open files/dirs. If not set:
	// - FileOpener: $EDITOR or "vim"
	// - DirOpener: platform default (open | xdg-open | explorer)
	FileOpener string `json:"file_opener" yaml:"file_opener" toml:"file_opener" env:"FILE_OPENER"`
	DirOpener  string `json:"dir_opener" yaml:"dir_opener" toml:"dir_opener" env:"DIR_OPENER"`
	// Multiline input mode: "messaging" (Shift+Enter for newline, Enter submits) or "eof" (EOF/Ctrl+D to submit)
	MultilineMode string `json:"multiline_mode" yaml:"multiline_mode" toml:"multiline_mode" env:"MULTILINE_MODE"`
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

// promptMultiline writes a message to stdout and reads multiline input.
// The behavior depends on the mode:
// - "messaging": Shift+Enter for newline, Enter submits (like messaging apps)
// - "eof": Type 'EOF' on a new line or press Ctrl+D to submit (traditional)
func promptMultiline(s string, mode string) (string, error) {
	if mode == multilineModeMessaging {
		return promptMultilineMessaging(s)
	}
	return promptMultilineEOF(s)
}

// promptMultilineEOF implements the traditional EOF-based multiline input
func promptMultilineEOF(s string) (string, error) {
	fmt.Print(s)
	fmt.Println("(Type 'EOF' on a new line when finished, or press Ctrl+D)")

	scanner := bufio.NewScanner(os.Stdin)
	var lines []string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "EOF" {
			break
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}

// promptMultilineMessaging implements messaging app style input:
// Double Enter (empty line) submits, single Enter creates newline
func promptMultilineMessaging(s string) (string, error) {
	fmt.Print(s)
	fmt.Println("(Press Enter twice (empty line) to submit, single Enter for new line)")

	scanner := bufio.NewScanner(os.Stdin)
	var lines []string

	for scanner.Scan() {
		line := scanner.Text()

		// Empty line submits (like messaging apps with double-enter)
		if line == "" && len(lines) > 0 {
			break
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}

// buildYAMLSnippet returns a YAML fragment representing an espanso match
// entry. For a single trigger, the YAML uses `trigger:`; for multiple,
// it uses an inline list with `triggers:`. Multiline replace strings use
// the YAML literal block style (|) with proper indentation.
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

	// Handle multiline replace strings with YAML literal block style
	if strings.Contains(replace, "\n") {
		b.WriteString("    replace: |\n")
		// Indent each line with 6 spaces (4 for replace + 2 for literal block content)
		for _, line := range strings.Split(replace, "\n") {
			b.WriteString("      " + line + "\n")
		}
	} else {
		b.WriteString("    replace: ")
		b.WriteString(fmt.Sprintf("%q\n", replace))
	}
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

// defineFlags wires up flags on the provided FlagSet. It supports a full and
// shorthand for each relevant option.
func defineFlags(fs *flag.FlagSet, matchPath *string, openFile *bool, openDir *bool) {
	fs.StringVar(matchPath, "matchFile", "", "Path to the espanso match file (overrides config). Accepts a directory or full file path.")
	fs.StringVar(matchPath, "m", "", "Shorthand for --matchFile")
	fs.BoolVar(openFile, "open", false, "Open the resolved match file and exit")
	fs.BoolVar(openFile, "o", false, "Shorthand for --open")
	fs.BoolVar(openDir, "openDir", false, "Open the resolved match directory and exit")
	fs.BoolVar(openDir, "d", false, "Shorthand for --openDir")
}

// checkOpenConflict ensures mutually exclusive use of --open and --dir.
func checkOpenConflict(openFile, openDir bool) error {
	if openFile && openDir {
		return fmt.Errorf("flags --open and --dir are mutually exclusive")
	}
	return nil
}

// usage prints a concise help message.
func usage() {
	fmt.Fprintf(os.Stderr, "cliesp - append espanso matches or open target file/dir\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n  cliesp [flags]\n\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	fmt.Fprintf(os.Stderr, "  -m, --matchFile string   Path to match file (dir or full file path) [flag > env/.env > config > defaults]\n")
	fmt.Fprintf(os.Stderr, "  -o, --open               Open the resolved match file and exit\n")
	fmt.Fprintf(os.Stderr, "  -d, --openDir            Open the resolved match directory and exit\n")
	fmt.Fprintf(os.Stderr, "  -h, --help               Show this help message\n\n")
	fmt.Fprintf(os.Stderr, "Configuration:\n")
	fmt.Fprintf(os.Stderr, "  Config file: ~/.config/cliesp/settings.{yaml|yml|toml|json}\n")
	fmt.Fprintf(os.Stderr, "  Env vars (.env supported): CLIESP_MATCH_DIR, CLIESP_MATCH_FILE, CLIESP_FILE_OPENER, CLIESP_DIR_OPENER\n")
	fmt.Fprintf(os.Stderr, "  Defaults: dir='%s', file='%s'\n", defaultEspansoMatchDir, defaultEspansoMatchFile)
	fmt.Fprintf(os.Stderr, "  Opener defaults: file=$EDITOR or 'vim'; dir=open|xdg-open|explorer (per platform)\n")
}

// pickFileOpener selects the command to open a file based on config and env.
func pickFileOpener(cfg AppConfig) string {
	if s := strings.TrimSpace(cfg.FileOpener); s != "" {
		return s
	}
	if ed := strings.TrimSpace(os.Getenv("EDITOR")); ed != "" {
		return ed
	}
	return "vim"
}

// pickDirOpener selects the command to open a directory based on config or platform default.
func pickDirOpener(cfg AppConfig) string {
	if s := strings.TrimSpace(cfg.DirOpener); s != "" {
		return s
	}
	switch runtime.GOOS {
	case "linux":
		return "xdg-open"
	case "windows":
		return "explorer"
	default:
		return "open"
	}
}

// runOpen executes an opener command with the target path. If the opener contains
// spaces (e.g., "code -w"), it splits into command and args.
func runOpen(opener, target string) error {
	parts := strings.Fields(opener)
	if len(parts) == 0 {
		return fmt.Errorf("invalid opener command")
	}
	name := parts[0]
	args := append(parts[1:], target)
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("could not find opener '%s' in PATH", name)
	}
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	// Preload .env files from the current working directory to ensure env
	// variables are available via process environment even if file-based
	// loading is skipped. Missing files are ignored by godotenv.Load.
	_ = godotenv.Load(".env", ".env.local", ".env.production")

	// Flags
	var matchFlag string
	var openFlag bool
	var dirFlag bool
	flag.Usage = usage
	defineFlags(flag.CommandLine, &matchFlag, &openFlag, &dirFlag)
	// Allow intermixing flags and prompts
	flag.Parse()

	// Load config from files/env via cliutils/config
	cfg, err := cfgpkg.Load(cfgpkg.Options[AppConfig]{
		AppName: "cliesp",
		ConsumerConfig: AppConfig{
			MatchDir:      defaultEspansoMatchDir,
			MatchFile:     defaultEspansoMatchFile,
			MultilineMode: defaultMultilineMode,
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

	// If open/dir flags were provided, enforce mutual exclusion and open accordingly
	if err := checkOpenConflict(openFlag, dirFlag); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if openFlag || dirFlag {
		target := filePath
		if dirFlag {
			target = filepath.Dir(filePath)
		}
		opener := pickFileOpener(cfg)
		if dirFlag {
			opener = pickDirOpener(cfg)
		}
		if err := runOpen(opener, target); err != nil {
			fmt.Fprintln(os.Stderr, "failed to open:", err)
			os.Exit(1)
		}
		fmt.Printf("Opened %s\n", target)
		return
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

	// Determine multiline mode from config
	mode := cfg.MultilineMode
	if mode == "" {
		mode = defaultMultilineMode
	}

	replaceStr, err := promptMultiline("replace with? (supports multiline): ", mode)
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
