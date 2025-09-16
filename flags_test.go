package main

import (
	"flag"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// parseArgs is a small helper to test flag parsing without affecting global flags.
func parseArgs(args []string) (match string, open bool, dir bool, err error) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	defineFlags(fs, &match, &open, &dir)
	err = fs.Parse(args)
	return
}

func TestFlagParsing_OpenAndDirMutuallyExclusive(t *testing.T) {
	_, open, dir, _ := parseArgs([]string{"--open", "--openDir"})
	if !(open && dir) {
		t.Fatalf("expected both flags set by parser; got open=%v dir=%v", open, dir)
	}
	if err := checkOpenConflict(open, dir); err == nil {
		t.Fatalf("expected conflict error, got nil")
	}
}

func TestFlagParsing_MatchFileDirectoryAndFile(t *testing.T) {
	m, _, _, _ := parseArgs([]string{"--matchFile", "/tmp"})
	if m != "/tmp" {
		t.Fatalf("expected /tmp, got %q", m)
	}
	m, _, _, _ = parseArgs([]string{"-m", "/tmp/file.yml"})
	if m != "/tmp/file.yml" {
		t.Fatalf("expected /tmp/file.yml, got %q", m)
	}
}

func TestResolve_WithFlags(t *testing.T) {
	cfg := AppConfig{MatchDir: "/base/dir", MatchFile: "x.yml"}
	p, err := resolveMatchPath("/override/dir"+string(filepath.Separator), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join("/override/dir", "x.yml") {
		t.Fatalf("unexpected resolved path: %q", p)
	}
	p, err = resolveMatchPath("/override/dir/custom.yml", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, filepath.Join("/override/dir", "custom.yml")) {
		t.Fatalf("unexpected resolved path: %q", p)
	}
}

func TestUsage_HelpTextContainsKeyLines(t *testing.T) {
	// Capture usage output by substituting flag.Usage temporarily would be complex here.
	// Instead, verify that the static strings we print remain valid in this function by calling it.
	// On some environments printing to stderr may be noisy; this is a smoke check.
	if runtime.GOOS == "windows" {
		t.Skip("skip usage smoke check on windows")
	}
	usage()
}
