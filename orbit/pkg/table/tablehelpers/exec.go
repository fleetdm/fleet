// based on github.com/kolide/launcher/pkg/osquery/tables
package tablehelpers

// based on https://github.com/fleetdm/launcher/blob/main/pkg/osquery/tables/tablehelpers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/rs/zerolog"
)

// Exec is a wrapper over exec.CommandContext. It does a couple of
// additional things to help with table usage:
//  1. It enforces a timeout.
//  2. Second, it accepts an array of possible binaries locations, and if something is not
//     found, it will go down the list.
//  3. It moves the stderr into the return error, if needed.
//
// This is not suitable for high performance work -- it allocates new buffers each time.
func Exec(ctx context.Context, log zerolog.Logger, timeoutSeconds int, possibleBins []string, args []string, includeStderr bool) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	for _, bin := range possibleBins {
		stdout.Reset()
		stderr.Reset()

		cmd := exec.CommandContext(ctx, bin, args...)
		cmd.Stdout = &stdout
		if includeStderr {
			cmd.Stderr = &stdout
		} else {
			cmd.Stderr = &stderr
		}

		log.Debug().Str("cmd", cmd.String()).Msg("execing")

		switch err := cmd.Run(); {
		case err == nil:
			return stdout.Bytes(), nil
		case os.IsNotExist(err):
			// try the next binary
			continue
		default:
			// an actual error
			return nil, fmt.Errorf("exec '%s'. Got: '%s': %w", cmd.String(), stderr.String(), err)
		}

	}
	// Getting here means no binary was found
	return nil, fmt.Errorf("No binary found in specified paths: %v: %w", possibleBins, os.ErrNotExist)
}
