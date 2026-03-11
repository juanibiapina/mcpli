package oauth

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser opens the given URL in the user's default browser.
// If the browser cannot be opened, prints the URL for manual copy-paste.
func OpenBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		fmt.Printf("Open this URL in your browser:\n  %s\n", url)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Could not open browser automatically.\nOpen this URL in your browser:\n  %s\n", url)
	}
}
