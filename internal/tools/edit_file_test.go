package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditFileTool_unique(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("hello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tool := NewEditFileTool(dir)
	out, err := tool.Call(context.Background(), CallInput{
		Arguments: map[string]string{
			"path":       "a.txt",
			"old_string": "world",
			"new_string": "there",
		},
	})
	if err != nil || out != "ok" {
		t.Fatalf("got %q %v", out, err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "hello there\n" {
		t.Fatalf("content %q", b)
	}
}

func TestEditFileTool_notFound(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	_ = os.WriteFile(p, []byte("x"), 0o644)
	tool := NewEditFileTool(dir)
	_, err := tool.Call(context.Background(), CallInput{
		Arguments: map[string]string{
			"path":       "a.txt",
			"old_string": "missing",
			"new_string": "y",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestEditFileTool_ambiguous(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	_ = os.WriteFile(p, []byte("aa aa"), 0o644)
	tool := NewEditFileTool(dir)
	_, err := tool.Call(context.Background(), CallInput{
		Arguments: map[string]string{
			"path":       "a.txt",
			"old_string": "aa",
			"new_string": "b",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "2 times") {
		t.Fatalf("want ambiguous error, got %v", err)
	}
}

func TestEditFileTool_replaceAll(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	_ = os.WriteFile(p, []byte("aa aa"), 0o644)
	tool := NewEditFileTool(dir)
	out, err := tool.Call(context.Background(), CallInput{
		Arguments: map[string]string{
			"path":         "a.txt",
			"old_string":   "aa",
			"new_string":   "b",
			"replace_all":  "true",
		},
	})
	if err != nil || out != "ok" {
		t.Fatalf("got %q %v", out, err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "b b" {
		t.Fatalf("content %q", b)
	}
}

func TestEditFileTool_emptyOldString(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	_ = os.WriteFile(p, []byte("x"), 0o644)
	tool := NewEditFileTool(dir)
	_, err := tool.Call(context.Background(), CallInput{
		Arguments: map[string]string{
			"path":       "a.txt",
			"old_string": "",
			"new_string": "y",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("got %v", err)
	}
}
