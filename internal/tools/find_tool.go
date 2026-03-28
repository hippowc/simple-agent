package tools

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// FindFilesTool 在 workspace 内按 glob 查找文件路径（语义类似 find + name 匹配）。
type FindFilesTool struct {
	workspace string
}

func NewFindFilesTool(workspace string) *FindFilesTool {
	return &FindFilesTool{workspace: workspace}
}

func (t *FindFilesTool) Name() string { return "find_files" }

func (t *FindFilesTool) Description() string {
	return "List files under a directory whose relative path matches a glob pattern (e.g. **/*.go, **/test_*)."
}

func (t *FindFilesTool) Call(ctx context.Context, input CallInput) (string, error) {
	pattern := input.Arguments["pattern"]
	if pattern == "" {
		return "", errors.New("pattern is required (glob, use / as separator, e.g. **/*.go)")
	}
	root := input.Arguments["root"]
	if root == "" {
		root = "."
	}
	maxN := argInt(input.Arguments, "max_results", 500)
	if maxN < 1 {
		maxN = 1
	}
	if maxN > 10000 {
		maxN = 10000
	}

	rootAbs, err := resolvePath(t.workspace, root)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(rootAbs)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", errors.New("root must be a directory")
	}

	var out []string
	n := 0
	err = filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		ok, err := doublestar.PathMatch(pattern, relSlash)
		if err != nil || !ok {
			return nil
		}
		out = append(out, relSlash)
		n++
		if n >= maxN {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(out) == 0 {
		return "(no matches)", nil
	}
	return strings.Join(out, "\n"), nil
}
