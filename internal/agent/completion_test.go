package agent

import (
	"strings"
	"testing"

	"simple-agent/internal/common"
)

func TestCompletionsRoot(t *testing.T) {
	cfg := common.DefaultConfig()
	out := completionsForConfig(&cfg, "/m")
	if len(out) == 0 {
		t.Fatal("expected /m completions")
	}
	if out[0].Insert != "/model " {
		t.Fatalf("first insert = %q", out[0].Insert)
	}
}

func TestCompletionsModelUse(t *testing.T) {
	cfg := common.DefaultConfig()
	out := completionsForConfig(&cfg, "/model use ")
	if len(out) == 0 {
		t.Fatal("expected profile completions")
	}
	if !strings.HasPrefix(out[0].Insert, "/model use ") {
		t.Fatalf("insert = %q", out[0].Insert)
	}
}
