package main

import (
	"os"
	"path/filepath"
	"testing"

	cfgpkg "github.com/kvnloughead/cliutils/config"
)

func TestConfigLoader_FromFile(t *testing.T) {
	tdir := t.TempDir()
	// Write a YAML config file
	if err := os.MkdirAll(tdir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(tdir, "settings.yaml")
	yaml := []byte("match_dir: /tmp/fromfile\nmatch_file: file.yml\n")
	if err := os.WriteFile(cfgPath, yaml, 0o644); err != nil {
		t.Fatal(err)
	}

	ldr := cfgpkg.NewLoader(cfgpkg.Options[AppConfig]{
		AppName:        "cliesp",
		ConsumerConfig: AppConfig{MatchDir: "", MatchFile: ""},
	})
	ldr.SetConfigPath(tdir)

	cfg, err := ldr.Load()
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if cfg.MatchDir != "/tmp/fromfile" || cfg.MatchFile != "file.yml" {
		t.Fatalf("unexpected config from file: %+v", cfg)
	}
}

func TestConfigLoader_FromEnvFile(t *testing.T) {
	tdir := t.TempDir()
	envPath := filepath.Join(tdir, ".env")
	// Use default prefix CLIESP_
	env := []byte("CLIESP_MATCH_DIR=/tmp/fromenv\nCLIESP_MATCH_FILE=env.yml\n")
	if err := os.WriteFile(envPath, env, 0o644); err != nil {
		t.Fatal(err)
	}

	ldr := cfgpkg.NewLoader(cfgpkg.Options[AppConfig]{
		AppName:        "cliesp",
		ConsumerConfig: AppConfig{MatchDir: "", MatchFile: ""},
	})
	ldr.SetEnvFiles([]string{envPath})

	cfg, err := ldr.Load()
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if cfg.MatchDir != "/tmp/fromenv" || cfg.MatchFile != "env.yml" {
		t.Fatalf("unexpected config from env file: %+v", cfg)
	}
}

func TestConfigLoader_EnvOverridesFile(t *testing.T) {
	tdir := t.TempDir()
	// Write config file
	cfgPath := filepath.Join(tdir, "settings.yaml")
	yaml := []byte("match_dir: /tmp/fromfile\nmatch_file: file.yml\n")
	if err := os.WriteFile(cfgPath, yaml, 0o644); err != nil {
		t.Fatal(err)
	}

	// Set process env to override dir only
	old := os.Getenv("CLIESP_MATCH_DIR")
	defer os.Setenv("CLIESP_MATCH_DIR", old)
	if err := os.Setenv("CLIESP_MATCH_DIR", "/tmp/fromenvvar"); err != nil {
		t.Fatal(err)
	}

	ldr := cfgpkg.NewLoader(cfgpkg.Options[AppConfig]{
		AppName:        "cliesp",
		ConsumerConfig: AppConfig{MatchDir: "", MatchFile: ""},
	})
	ldr.SetConfigPath(tdir)

	cfg, err := ldr.Load()
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if cfg.MatchDir != "/tmp/fromenvvar" {
		t.Fatalf("env var should override file for MatchDir: %+v", cfg)
	}
	if cfg.MatchFile != "file.yml" { // unchanged from file
		t.Fatalf("MatchFile should remain from file: %+v", cfg)
	}
}

func TestResolveMatchPath_PrecendenceFlagOverConfig(t *testing.T) {
	tdir := t.TempDir()
	cfg := AppConfig{MatchDir: tdir, MatchFile: "file.yml"}
	flagPath := filepath.Join(tdir, "override.yml")
	p, err := resolveMatchPath(flagPath, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if p != flagPath {
		t.Fatalf("flag path should win: got=%q want=%q", p, flagPath)
	}
}
