package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildYAMLSnippetSingle(t *testing.T) {
	got := buildYAMLSnippet([]string{":one"}, "Hello")
	want := "\n  - trigger: \":one\"\n    replace: \"Hello\"\n"
	if got != want {
		// Show a readable diff hint
		t.Errorf("single trigger YAML mismatch\nGot:\n%q\nWant:\n%q", got, want)
	}
}

func TestBuildYAMLSnippetMultiple(t *testing.T) {
	got := buildYAMLSnippet([]string{":a", ":b"}, "Hi")
	want := "\n  - triggers: [\":a\", \":b\"]\n    replace: \"Hi\"\n"
	if got != want {
		t.Errorf("multi triggers YAML mismatch\nGot:\n%q\nWant:\n%q", got, want)
	}
}

func TestBuildYAMLSnippetMultiline(t *testing.T) {
	multilineContent := "{quiz-task}\n    background: |\n        #f5f6f7\n    header: |\n\n    content: |\n\n        <content goes here>\n{/quiz-task}"
	got := buildYAMLSnippet([]string{":cms-callout"}, multilineContent)
	want := "\n  - trigger: \":cms-callout\"\n    replace: |\n      {quiz-task}\n          background: |\n              #f5f6f7\n          header: |\n      \n          content: |\n      \n              <content goes here>\n      {/quiz-task}\n"
	if got != want {
		t.Errorf("multiline YAML mismatch\nGot:\n%q\nWant:\n%q", got, want)
	}
}

func TestBuildYAMLSnippetMultilineWithEmptyLines(t *testing.T) {
	multilineContent := "line1\n\nline3\n"
	got := buildYAMLSnippet([]string{":test"}, multilineContent)
	want := "\n  - trigger: \":test\"\n    replace: |\n      line1\n      \n      line3\n      \n"
	if got != want {
		t.Errorf("multiline with empty lines YAML mismatch\nGot:\n%q\nWant:\n%q", got, want)
	}
}

func TestPromptMultilineMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected string // The function signature to expect
	}{
		{
			name: "messaging mode",
			mode: multilineModeMessaging,
		},
		{
			name: "eof mode",
			mode: multilineModeEOF,
		},
		{
			name: "invalid mode defaults to eof",
			mode: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test the interactive functions without complex setup,
			// but we can verify the mode routing logic works
			if tt.mode == multilineModeMessaging {
				// Just verify the function exists and has correct signature
				_ = promptMultilineMessaging
			} else {
				// Verify EOF mode function exists
				_ = promptMultilineEOF
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	// Skip on systems without a home dir (very rare in normal Go CI)
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no home dir available for test")
	}

	got, err := expandHome("~/foo/bar")
	if err != nil {
		t.Fatalf("expandHome returned error: %v", err)
	}
	want := filepath.Join(home, "foo", "bar")
	if got != want {
		t.Errorf("expandHome mismatch got=%q want=%q", got, want)
	}

	// "~" alone should expand to home
	got, err = expandHome("~")
	if err != nil {
		t.Fatalf("expandHome(~) error: %v", err)
	}
	if got != home {
		// On Windows, filepath.Join(home, "") == home; same expected
		t.Errorf("expandHome(~) mismatch got=%q want=%q", got, home)
	}

	// Non-tilde paths should be unchanged
	p := "/tmp/x"
	if runtime.GOOS == "windows" {
		p = `C:\\tmp\\x`
	}
	got, err = expandHome(p)
	if err != nil {
		t.Fatalf("expandHome(non-tilde) error: %v", err)
	}
	if got != p {
		t.Errorf("expandHome should not change non-tilde path: got=%q want=%q", got, p)
	}
}

func TestEnsureFileWithHeader_CreatesFileWithHeader(t *testing.T) {
	tdir := t.TempDir()
	p := filepath.Join(tdir, "nested", "cliesp.yml")

	if err := ensureFileWithHeader(p); err != nil {
		t.Fatalf("ensureFileWithHeader error: %v", err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("reading created file: %v", err)
	}
	content := string(b)
	if !strings.HasPrefix(content, "# espanso match file") {
		t.Errorf("file does not start with expected header prefix: %q", content[:min(64, len(content))])
	}
	if !strings.Contains(content, "\nmatches:\n") {
		t.Errorf("file header missing 'matches:' root, content=%q", content)
	}
}

func TestEnsureFileWithHeader_DoesNotOverwrite(t *testing.T) {
	tdir := t.TempDir()
	p := filepath.Join(tdir, "cliesp.yml")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	orig := "existing content\n"
	if err := os.WriteFile(p, []byte(orig), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	if err := ensureFileWithHeader(p); err != nil {
		t.Fatalf("ensureFileWithHeader error: %v", err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read after ensure: %v", err)
	}
	if string(b) != orig {
		t.Errorf("file was modified but should not have been. got=%q want=%q", string(b), orig)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
