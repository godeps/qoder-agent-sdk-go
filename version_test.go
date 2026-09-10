package qodersdk

import "testing"

func TestCheckProtocolVersion(t *testing.T) {
	cases := []struct {
		ver  string
		want bool // true = compatible
	}{
		{"1.4.0", true},
		{"1.0.0", true},
		{"1.99.0", true},
		{"2.0.0", false},
		{"0.9.0", false},
		{"", false}, // empty → no major match
	}
	for _, c := range cases {
		err := checkProtocolVersion(c.ver)
		if c.want && err != nil {
			t.Errorf("checkProtocolVersion(%q) = %v, want nil", c.ver, err)
		}
		if !c.want && err == nil {
			t.Errorf("checkProtocolVersion(%q) = nil, want error", c.ver)
		}
	}
}
