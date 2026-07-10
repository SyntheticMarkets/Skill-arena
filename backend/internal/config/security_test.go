package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	saredis "skill-arena/internal/redis"
	"skill-arena/internal/storage"
)

func TestProductionDependencyFailuresAreExplicit(t *testing.T) {
	if err := (saredis.NetworkClient{URL: "redis://127.0.0.1:1"}).Health(t.Context()); err == nil {
		t.Fatal("expected redis health failure for unavailable redis")
	}
	if err := (storage.S3CompatibleStore{}).Health(t.Context()); err == nil {
		t.Fatal("expected object storage health failure for incomplete credentials")
	}
}

func TestNoHardcodedProductionSecretsInSource(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	secretPattern := regexp.MustCompile(`(?i)(secret|password|passphrase|private[_-]?key|access[_-]?key)\s*[:=]\s*"[^"$]+?"`)
	allowed := []string{
		"test-secret",
		"server-secret",
		"local-development-puzzle-secret",
		"puzzle generation secret is required",
		"SKILL_ARENA_",
	}
	var findings []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".gocache":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range secretPattern.FindAllString(string(data), -1) {
			accepted := false
			for _, token := range allowed {
				if strings.Contains(match, token) {
					accepted = true
					break
				}
			}
			if !accepted {
				findings = append(findings, path+": "+match)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk source: %v", err)
	}
	if len(findings) > 0 {
		t.Fatalf("hardcoded secret-like values found: %v", findings)
	}
}
