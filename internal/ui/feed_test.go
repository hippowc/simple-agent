package ui

import "testing"

func TestLineCount(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"   ", 0},
		{"a", 1},
		{"a\nb", 2},
		{"a\nb\n", 2},
		{"x\ny\nz", 3},
	}
	for _, tt := range tests {
		if got := lineCount(tt.in); got != tt.want {
			t.Errorf("lineCount(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestDefaultExpandedForTool(t *testing.T) {
	if !defaultExpandedForTool("1\n2\n3") {
		t.Error("3 lines should default expanded (threshold 3)")
	}
	if defaultExpandedForTool("1\n2\n3\n4") {
		t.Error("4 lines should default collapsed")
	}
}

func TestToolFriendlyName(t *testing.T) {
	if got := toolFriendlyName("read_file"); got != "读文件" {
		t.Errorf("read_file -> %q", got)
	}
	if got := toolFriendlyName("unknown_thing"); got != "unknown thing" {
		t.Errorf("unknown_thing -> %q", got)
	}
}
