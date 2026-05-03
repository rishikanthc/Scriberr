package api

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const forbiddenAPIImport = "scriberr/internal/database"

func TestProductionAPIDatabaseAccessInventory(t *testing.T) {
	// This is a stop-the-line guard for the backend service-boundary refactor.
	// Sprint 0 freezes existing API database access. Later sprints must shrink
	// this allowlist as handlers move behind service interfaces.
	expected := []string{
		"admin_handlers.go",
		"chat_handlers.go",
		"file_handlers.go",
		"summary_handlers.go",
		"summary_widget_handlers.go",
		"transcription_handlers.go",
	}

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
		importsDatabase, err := fileImports(path, forbiddenAPIImport)
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

func fileImports(path string, importPath string) (bool, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return false, err
	}
	quoted := `"` + importPath + `"`
	for _, spec := range parsed.Imports {
		if spec.Path != nil && spec.Path.Value == quoted {
			return true, nil
		}
	}
	return false, nil
}
