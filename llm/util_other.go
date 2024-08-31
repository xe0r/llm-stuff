//go:build !windows

package llm

import "os"

func secureGetToken() (string, error) {
	return "", os.ErrNotExist
}
