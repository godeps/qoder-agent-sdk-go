package runtime

import (
	"testing"

	"github.com/godeps/qoder-agent-sdk-go/auth"
)

func TestBinaryNamesBrandOrder(t *testing.T) {
	cases := []struct {
		brand auth.Brand
		first string
	}{
		{auth.BrandGlobal, "qodercli"},
		{auth.BrandCN, "qoderclicn"},
		{auth.BrandAuto, "qoderclicn"}, // auto defaults to cn order
	}
	for _, c := range cases {
		got := BinaryNames(c.brand)
		if len(got) < 2 || got[0] != c.first {
			t.Errorf("BinaryNames(%q)[0] = %v, want %q", c.brand, got, c.first)
		}
		// both brands' binaries must be present (so auto can find either).
		seen := map[string]bool{}
		for _, n := range got {
			seen[n] = true
		}
		if !seen["qodercli"] || !seen["qoderclicn"] {
			t.Errorf("BinaryNames(%q) = %v, missing both brands", c.brand, got)
		}
	}
}
