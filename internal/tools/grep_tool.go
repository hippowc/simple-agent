package tools

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepContentTool 在 workspace 内按正则搜索文件内容（语义类似 grep -r）。
type GrepContentTool struct {
	workspace string
}

func NewGrepContentTool(workspace string) *GrepContentTool {
	return &GrepContentTool{workspace: workspace}
}

func (t *GrepContentTool) Name() string { return "grep_content" }

func (t *GrepContentTool) Description() string {
	return "Search file contents with a regular expression under a file or directory (skips likely binary files)."
}

func (t *GrepContentTool) Call(ctx context.Context, input CallInput) (string, error) {
	pat := input.Arguments["pattern"]
	if pat == "" {
		return "", errors.New("pattern is required (regular expression)")
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return "", fmt.Errorf("invalid regexp: %w", err)
	}
	target := input.Arguments["path"]
	if target == "" {
		return "", errors.New("path is required (file or directory under workspace)")
	}
	globFilter := input.Arguments["glob"]
	maxLines := argInt(input.Arguments, "max_results", 200)
	if maxLines < 1 {
		maxLines = 1
	}
	if maxLines > 5000 {
		maxLines = 5000
	}

	wsRoot, err := resolvePath(t.workspace, ".")
	if err != nil {
		return "", err
	}

	abs, err := resolvePath(t.workspace, target)
	if err != nil {
		return "", err
	}

	var lines []string
	count := 0

	flush := func(fileRel, text string, lineNo int) {
		if count >= maxLines {
			return
		}
		lines = append(lines, fmt.Sprintf("%s:%d:%s", fileRel, lineNo, text))
		count++
	}

	grepOne := func(pathAbs, relDisplay string) error {
		data, err := os.ReadFile(pathAbs)
		if err != nil {
			return nil
		}
		if isLikelyBinary(data) {
			return nil
		}
		if globFilter != "" {
			ok, err := filepath.Match(globFilter, filepath.Base(pathAbs))
			if err != nil || !ok {
				return nil
			}
		}
		sc := bufio.NewScanner(bytes.NewReader(data))
		buf := make([]byte, 0, 64*1024)
		sc.Buffer(buf, 1024*1024)
		lineNo := 0
		for sc.Scan() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lineNo++
			line := sc.Text()
			if re.MatchString(line) {
				flush(relDisplay, line, lineNo)
				if count >= maxLines {
					break
				}
			}
		}
		return sc.Err()
	}

	fi, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		rel, err := filepath.Rel(wsRoot, abs)
		if err != nil {
			rel = filepath.ToSlash(filepath.Base(abs))
		} else {
			rel = filepath.ToSlash(rel)
		}
		if err := grepOne(abs, rel); err != nil {
			return "", err
		}
		return formatGrepResult(lines, count, maxLines), nil
	}

	rootAbs := abs
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
		if count >= maxLines {
			return fs.SkipAll
		}
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		return grepOne(path, relSlash)
	})
	if err != nil {
		return "", err
	}
	return formatGrepResult(lines, count, maxLines), nil
}

func formatGrepResult(lines []string, count, maxLines int) string {
	if len(lines) == 0 {
		return "(no matches)"
	}
	s := strings.Join(lines, "\n")
	if count >= maxLines {
		s += fmt.Sprintf("\n(truncated: max_results=%d)", maxLines)
	}
	return s
}

func isLikelyBinary(data []byte) bool {
	n := len(data)
	if n > 8000 {
		n = 8000
	}
	return bytes.IndexByte(data[:n], 0) >= 0
}
