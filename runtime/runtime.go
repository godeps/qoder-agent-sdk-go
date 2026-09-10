// Package runtime resolves the qoderclicn/qodercli executable.
//
// Two brands are supported, mirroring the two npm packages:
//   - global (@qoder-ai/qoder-agent-sdk): binary "qodercli"
//   - cn     (@qodercn-ai/qodercn-agent-sdk): binary "qoderclicn"
package runtime

import (
	"os"
	"os/exec"

	"github.com/godeps/qoder-agent-sdk-go/auth"
)

// NotFoundError indicates the CLI executable could not be located.
type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "qoder: qoderclicn/qodercli executable not found (set QODERCLI_PATH or install it in PATH)"
}

// BinaryNames returns the CLI binary names for a brand, preferred order first.
// For BrandAuto, the cn binary is tried first (matching the cn SDK default).
func BinaryNames(brand auth.Brand) []string {
	switch brand {
	case auth.BrandGlobal:
		return []string{"qodercli", "qoderclicn"}
	case auth.BrandCN:
		return []string{"qoderclicn", "qodercli"}
	}
	return []string{"qoderclicn", "qodercli"}
}

// ResolvePath resolves the CLI executable path for the given brand.
//
// Precedence: explicit override → QODERCLI_PATH env var → PATH lookup over the
// brand's binary names.
func ResolvePath(brand auth.Brand, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if p := os.Getenv("QODERCLI_PATH"); p != "" {
		return p, nil
	}
	for _, name := range BinaryNames(brand) {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", &NotFoundError{}
}
