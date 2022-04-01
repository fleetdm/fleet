package open

import "fmt"

// Browser opens the default browser at the given url and returns.
func Browser(url string) error {
	if err := browser(url); err != nil {
		return fmt.Errorf("open in browser: %w", err)
	}
	return nil
}
