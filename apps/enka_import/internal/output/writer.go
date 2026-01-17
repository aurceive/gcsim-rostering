package output

import (
	"fmt"
	"os"
)

func WriteTextFile(path string, contents string) error {
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
