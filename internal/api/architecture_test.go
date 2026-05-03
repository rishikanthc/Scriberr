package api

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const forbiddenDatabaseImport = "scriberr/internal/database"

func TestProductionAPIDatabaseAccessInventory(t *testing.T) {
	expected := []string{}

	actual, err := productionFilesImportingDatabase(".")
	if err != nil {
		t.Fatalf("scan API imports: %v", err)
	}

	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("production API database import inventory changed.\nexpected:\n%s\nactual:\n%s\nUpdate the backend service-boundary tracker when intentionally removing entries; do not add new entries.",
			strings.Join(expected, "\n"),
			strings.Join(actual, "\n"))
	}
}

func TestProductionInternalDatabaseImportInventory(t *testing.T) {
	expected := []string{
		"app/app.go",
	}

	actual, err := productionInternalFilesImportingDatabase("..")
	if err != nil {
		t.Fatalf("scan internal imports: %v", err)
	}

	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("production internal database import inventory changed.\nexpected:\n%s\nactual:\n%s\nOnly composition/bootstrap code should import internal/database outside the database package itself.",
			strings.Join(expected, "\n"),
			strings.Join(actual, "\n"))
	}
}

func TestProductionInternalAPIImportInventory(t *testing.T) {
	expected := []string{
		"app/app.go",
	}

	actual, err := productionInternalFilesImporting("..", "scriberr/internal/api", "api")
	if err != nil {
		t.Fatalf("scan internal imports: %v", err)
	}

	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("production internal api import inventory changed.\nexpected:\n%s\nactual:\n%s\nOnly composition/bootstrap code should import internal/api.",
			strings.Join(expected, "\n"),
			strings.Join(actual, "\n"))
	}
}

func TestBackendDependencyDirection(t *testing.T) {
	tests := []struct {
		name      string
		root      string
		forbidden []string
		allowed   []string
	}{
		{
			name:      "models stay persistence-only",
			root:      "../models",
			forbidden: []string{"scriberr/internal/"},
		},
		{
			name:      "repositories depend only on models inside internal",
			root:      "../repository",
			forbidden: []string{"scriberr/internal/"},
			allowed:   []string{"scriberr/internal/models"},
		},
		{
			name:      "engine providers do not depend on api or repositories",
			root:      "../transcription/engineprovider",
			forbidden: []string{"scriberr/internal/api", "scriberr/internal/repository"},
		},
		{
			name:      "workers do not depend on api",
			root:      "../transcription/worker",
			forbidden: []string{"scriberr/internal/api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations, err := productionImportViolations(tt.root, tt.forbidden, tt.allowed)
			if err != nil {
				t.Fatalf("scan imports: %v", err)
			}
			if len(violations) > 0 {
				t.Fatalf("backend dependency direction violations:\n%s", strings.Join(violations, "\n"))
			}
		})
	}
}

func TestProductionAPIDoesNotOwnLLMProviderConnectionTester(t *testing.T) {
	for _, symbol := range []string{
		"LLMProviderConnectionTester",
		"testLLMProviderConnection",
		"fetchOpenAICompatibleModels",
		"fetchOllamaNativeModels",
	} {
		locations, err := productionAPIFilesContainingSymbol(".", symbol)
		if err != nil {
			t.Fatalf("scan API symbols: %v", err)
		}
		if len(locations) > 0 {
			t.Fatalf("production API owns LLM provider probing symbol %q in:\n%s\nMove concrete provider probing to internal/llmprovider and keep API handlers request/response-only.",
				symbol, strings.Join(locations, "\n"))
		}
	}
}

func productionFilesImportingDatabase(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		importsDatabase, err := fileImports(path, forbiddenDatabaseImport)
		if err != nil {
			return err
		}
		if importsDatabase {
			files = append(files, filepath.Base(path))
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func productionInternalFilesImportingDatabase(root string) ([]string, error) {
	return productionInternalFilesImporting(root, forbiddenDatabaseImport, "database")
}

func productionInternalFilesImporting(root, importPath, skipPackageDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if filepath.Base(path) == skipPackageDir {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		importsTarget, err := fileImports(path, importPath)
		if err != nil {
			return err
		}
		if importsTarget {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func productionImportViolations(root string, forbidden, allowed []string) ([]string, error) {
	var violations []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		imports, err := fileImportPaths(path)
		if err != nil {
			return err
		}
		for _, importPath := range imports {
			if !matchesAny(importPath, forbidden) || matchesAny(importPath, allowed) {
				continue
			}
			violations = append(violations, fmt.Sprintf("%s imports %s", filepath.ToSlash(path), importPath))
		}
		return nil
	})
	sort.Strings(violations)
	return violations, err
}

func productionAPIFilesContainingSymbol(root, symbol string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), symbol) {
			files = append(files, filepath.ToSlash(path))
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func fileImports(path string, importPath string) (bool, error) {
	imports, err := fileImportPaths(path)
	if err != nil {
		return false, err
	}
	for _, current := range imports {
		if current == importPath {
			return true, nil
		}
	}
	return false, nil
}

func fileImportPaths(path string) ([]string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var imports []string
	for _, spec := range parsed.Imports {
		if spec.Path == nil {
			continue
		}
		imports = append(imports, strings.Trim(spec.Path.Value, `"`))
	}
	return imports, nil
}

func matchesAny(importPath string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}
