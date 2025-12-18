//go:build darwin

package utils

import (
	"os/exec"
)

// openBrowser opens a URL in the default browser on macOS
func OpenBrowser(url string) error {
	return exec.Command("open", url).Run()
}
