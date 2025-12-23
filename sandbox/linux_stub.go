//go:build !linux

package sandbox

import "fmt"

func newLinux(cfg Config) (Sandbox, error) {
	return nil, fmt.Errorf("linux sandbox not available on this platform")
}
