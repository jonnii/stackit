//go:build linux

package utils

import (
	"os/exec"
)

// openBrowser opens a URL in the default browser on Linux
func OpenBrowser(url string) error {
	return exec.Command("xdg-open", url).Run()
}
