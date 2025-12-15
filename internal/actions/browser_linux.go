//go:build linux

package actions

import (
	"os/exec"
)

// openBrowser opens a URL in the default browser on Linux
func openBrowser(url string) error {
	return exec.Command("xdg-open", url).Run()
}
