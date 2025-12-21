//go:build windows

package utils

import (
	"os/exec"
)

// OpenBrowser opens a URL in the default browser on Windows
func OpenBrowser(url string) error {
	return exec.Command("cmd", "/c", "start", url).Run()
}
