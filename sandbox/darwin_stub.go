//go:build !darwin

package sandbox

import "fmt"

func newDarwin(cfg Config) (Sandbox, error) {
	return nil, fmt.Errorf("darwin sandbox not available on this platform")
}
