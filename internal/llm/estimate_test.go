package llm

import "testing"

func TestEstimateTokens(t *testing.T) {
	if EstimateTokens("") != 0 {
		t.Fatal("empty")
	}
	if n := EstimateTokens("abcd"); n != 1 {
		t.Fatalf("4 runes -> %d", n)
	}
}
