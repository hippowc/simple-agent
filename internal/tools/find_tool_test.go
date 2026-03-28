package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindFilesTool_basic(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "a", "b"), 0o755)
	if err := os.WriteFile(filepath.Join(dir, "a", "x.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a", "b", "y.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewFindFilesTool(dir)
	out, err := tool.Call(context.Background(), CallInput{
		Arguments: map[string]string{
			"root":    ".",
			"pattern": "**/*.go",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "a/x.go") {
		t.Fatalf("expected a/x.go in %q", out)
	}
	if strings.Contains(out, "y.txt") {
		t.Fatalf("should not list .txt: %q", out)
	}
}
