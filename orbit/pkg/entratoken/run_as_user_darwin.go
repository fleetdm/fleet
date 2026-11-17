//go:build darwin

package entratoken

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	osuser "os/user"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/user"
)

// GetIDTokenAsLoggedInUser runs this executable's entra-device-token command as the logged-in
// user, because the SSO extension only serves requests in that user's session. The child keeps
// orbit's signing identifier, which the SSO extension's AppAllowList must include.
func GetIDTokenAsLoggedInUser(ctx context.Context, r Request) (string, error) {
	if err := r.Validate(); err != nil {
		return "", err
	}
	username, err := user.UserLoggedInViaGui()
	if err != nil {
		return "", fmt.Errorf("check logged in user: %w", err)
	}
	if username == nil || *username == "" {
		return "", ErrNoUserLoggedIn
	}
	u, err := osuser.Lookup(*username)
	if err != nil {
		return "", fmt.Errorf("lookup user: %w", err)
	}
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("get executable: %w", err)
	}

	cmd := exec.CommandContext(ctx, "/bin/launchctl", "asuser", u.Uid,
		"/usr/bin/sudo", "-u", *username,
		executable, "entra-device-token",
		"--tenant-id", r.TenantID,
		"--client-id", r.ClientID,
		"--redirect-uri", r.RedirectURI,
		"--nonce", r.Nonce,
	)
	cmd.Env = []string{"PATH=/usr/bin:/bin:/usr/sbin:/sbin"}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("run entra-device-token: %w: %s", err, lastLine(stderr.String()))
	}
	token := lastLine(stdout.String())
	if strings.Count(token, ".") != 2 || strings.ContainsAny(token, " \t{") {
		return "", fmt.Errorf("run entra-device-token: unexpected output: %s", lastLine(stderr.String()))
	}
	return token, nil
}

func lastLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndexByte(s, '\n'); i >= 0 {
		return s[i+1:]
	}
	return s
}
