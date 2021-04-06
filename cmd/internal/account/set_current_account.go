package account

import (
	"fmt"
	"os"
)

func setCurrentAccount() error {
	if _, err := os.Lstat(currentPath); err == nil {
		if err := os.Remove(currentPath); err != nil {
			return fmt.Errorf("failed to unlink current config file: %v", err)
		}
	}

	if err := os.Symlink(aliasPath, currentPath); err != nil {
		return fmt.Errorf("could not symlink the config file for %s: %w", accountAlias, err)
	}

	return nil
}
