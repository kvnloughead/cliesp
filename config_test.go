package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveMatchPath_Defaults(t *testing.T) {
	p, err := resolveMatchPath("", AppConfig{})
	if err != nil {
		t.Fatal(err)
	}
	// Expand default dir
	d, err := expandHome(defaultEspansoMatchDir)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(d, defaultEspansoMatchFile)
	if p != want {
		t.Errorf("got %q want %q", p, want)
	}
}

func TestResolveMatchPath_FlagOverridesDir(t *testing.T) {
	tdir := t.TempDir()
	cfg := AppConfig{MatchFile: "file.yml"}
	p, err := resolveMatchPath(tdir+string(os.PathSeparator), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join(tdir, "file.yml") {
		t.Errorf("expected dir+file, got %q", p)
	}
}

func TestResolveMatchPath_FlagIsFile(t *testing.T) {
	tdir := t.TempDir()
	p, err := resolveMatchPath(filepath.Join(tdir, "custom.yml"), AppConfig{MatchFile: "ignored.yml"})
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join(tdir, "custom.yml") {
		t.Errorf("expected explicit file, got %q", p)
	}
}

func TestResolveMatchPath_ConfigDirAndFile(t *testing.T) {
	tdir := t.TempDir()
	cfg := AppConfig{MatchDir: tdir, MatchFile: "abc.yml"}
	p, err := resolveMatchPath("", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join(tdir, "abc.yml") {
		t.Errorf("got %q want %q", p, filepath.Join(tdir, "abc.yml"))
	}
}
