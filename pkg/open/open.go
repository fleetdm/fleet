package open

import "fmt"

// Browser opens the default browser at the given url and returns.
// Returns the output of the command opening the URL and its error, if any.
func Browser(url string) (string, error) {
	out, err := browser(url)
	if err != nil {
		return out, fmt.Errorf("open in browser: %w", err)
	}
	return out, nil
}
