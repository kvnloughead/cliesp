package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// hard-coded destination per request
	espansoMatchDir  = "~/Library/Application Support/espanso/match"
	espansoMatchFile = "cliesp.yml"
)

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
		header := `# espanso match file

# For a complete introduction, visit the official docs at: https://espanso.org/docs/

# You can use this file to define the base matches (aka snippets)
# that will be available in every application when using espanso.

# Matches are substitution rules: when you type the "trigger" string
# it gets replaced by the "replace" string.
matches:
`
		if _, err := f.WriteString(header); err != nil {
			return err
		}
	}
	return nil
}

func prompt(s string) (string, error) {
	fmt.Print(s)
	r := bufio.NewReader(os.Stdin)
	text, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

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

func main() {
	// Resolve target file
	dir, err := expandHome(espansoMatchDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error resolving home dir:", err)
		os.Exit(1)
	}
	filePath := filepath.Join(dir, espansoMatchFile)

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
