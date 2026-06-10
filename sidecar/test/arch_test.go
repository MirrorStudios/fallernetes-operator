package test

import (
	"bufio"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func getModuleName(dir string) (string, error) {
	file, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			slog.Error("failed to close go.mod", "error", closeErr)
		}
	}()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if after, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(after), nil
		}
	}
	return "", scanner.Err()
}

type archRule struct {
	name      string
	base      string
	forbidden []string
}

func TestArchitectureBoundaries(t *testing.T) {
	moduleName, err := getModuleName("../")
	if err != nil {
		t.Fatalf("failed to read module name: %v", err)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedFiles,
		Dir:  "../",
	}

	rules := []archRule{
		{
			name: "Service layer must not import adapters",
			base: moduleName + "/internal/service/...",
			forbidden: []string{
				moduleName + "/internal/adapters",
			},
		},
		{
			name: "Ports layer must not import adapters or service",
			base: moduleName + "/internal/ports/...",
			forbidden: []string{
				moduleName + "/internal/adapters",
				moduleName + "/internal/service",
			},
		},
	}

	for _, r := range rules {
		r := r
		t.Run(r.name, func(t *testing.T) {
			pkgs, err := packages.Load(cfg, r.base)
			if err != nil {
				t.Fatalf("failed to load packages: %v", err)
			}
			for _, pkg := range pkgs {
				for imp := range pkg.Imports {
					if !strings.HasPrefix(imp, moduleName+"/") {
						continue
					}
					for _, forbidden := range r.forbidden {
						if strings.HasPrefix(imp, forbidden) {
							t.Errorf("violation: %s imports forbidden package %s", pkg.PkgPath, imp)
						}
					}
				}
			}
		})
	}
}
