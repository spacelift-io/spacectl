package internal

import (
	"os/exec"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// OpenWebBrowser will try to open the default browser
// on a the callers machine. It supports windows, darwin and windows.
func OpenWebBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		r := strings.NewReplacer("&", "^&")
		cmd = exec.Command("cmd", "/c", "start", r.Replace(url)) //#nosec
	default:
		return errors.New("unsupported platform")
	}

	err := cmd.Start()
	if err != nil {
		return errors.Wrap(err, "could not open the browser")
	}

	err = cmd.Wait()
	if err != nil {
		return errors.Wrap(err, "could not wait for the opening browser")
	}

	return nil
}
