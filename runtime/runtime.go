// Package runtime resolves the qoderclicn executable.
package runtime

import (
	"os"
	"os/exec"
)

// NotFoundError indicates the qoderclicn executable could not be located.
type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "qoder: qoderclicn executable not found (set QODERCLI_PATH or install qoderclicn in PATH)"
}

// ResolvePath resolves the qoderclicn executable path.
//
// Precedence: explicit override → QODERCLI_PATH env var → PATH lookup for
// "qoderclicn" (CN brand) then "qodercli" (global brand).
func ResolvePath(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if p := os.Getenv("QODERCLI_PATH"); p != "" {
		return p, nil
	}
	for _, name := range []string{"qoderclicn", "qodercli"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", &NotFoundError{}
}
