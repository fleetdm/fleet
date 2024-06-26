package packaging

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/pkg/secure"
)

func rSign(pkgPath, cert string) error {
	pemPath := filepath.Join(os.TempDir(), "cert.pem")
	defer os.Remove(pemPath)
	err := os.WriteFile(pemPath, []byte(cert), 0o600)
	if err != nil {
		return fmt.Errorf("writing cert data: %s", err)
	}

	return retry.Do(func() error {
		var outBuf bytes.Buffer
		cmd := exec.Command(
			"rcodesign",
			"sign",
			pkgPath,
			"--pem-source", pemPath,
		)
		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf
		if err := cmd.Run(); err != nil {
			fmt.Println(outBuf.String())
			return fmt.Errorf("rcodesign: %w", err)
		}
		return nil
	}, retry.WithMaxAttempts(3))
}

func rNotarizeStaple(pkg, apiKeyID, apiKeyIssuer, apiKeyContent string) error {
	path, err := writeAPIKeys(apiKeyIssuer, apiKeyID, apiKeyContent)
	defer os.Remove(path)
	if err != nil {
		return fmt.Errorf("writing API keys: %s", err)
	}

	return retry.Do(func() error {
		var outBuf bytes.Buffer
		cmd := exec.Command("rcodesign",
			"notarize",
			pkg,
			"--api-issuer", apiKeyIssuer,
			"--api-key", apiKeyID,
			"--staple",
		)
		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf
		if err := cmd.Run(); err != nil {
			fmt.Println(outBuf.String())
			return fmt.Errorf("rcodesign notarize: %w", err)
		}
		return nil
	}, retry.WithMaxAttempts(3))
}

func writeAPIKeys(issuer, id, content string) (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home dir: %s", err)
	}

	// The underliying tools (rcodesign and Transporter) expect to find a
	// certificate key in this path.
	path := filepath.Join(homedir, ".appstoreconnect", "private_keys")
	if err = secure.MkdirAll(path, 0o600); err != nil {
		return "", fmt.Errorf("finding home dir: %s", err)
	}

	keyPath := filepath.Join(path, fmt.Sprintf("AuthKey_%s.p8", id))
	if err = os.WriteFile(keyPath, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("writing api key contents: %s", err)
	}

	return keyPath, nil
}
