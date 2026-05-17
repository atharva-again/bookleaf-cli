package cmd

import "os"

// isInteractive returns true when stdin is a terminal and /dev/tty is
// accessible.  The os.ModeCharDevice check alone is not sufficient —
// PTY-like environments can pass that but bubbletea (used by huh) still
// needs /dev/tty to render TUIs.  When running in CI, piped input, or
// cron this returns false and prompts are skipped.
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		return false
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	tty.Close()
	return true
}
