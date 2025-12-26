package weaponroster

import (
	"os"
)

func withSilencedStdoutStderr(fn func() error) error {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err == nil {
		os.Stdout = devNull
		os.Stderr = devNull
		defer func() {
			devNull.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr
		}()
	}

	return fn()
}
