package packaging

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func rSign(pkgPath, cert, certPwd string) error {
	certPath := filepath.Join(os.TempDir(), "cert.p12")
	defer os.Remove(certPath)
	err := os.WriteFile(certPath, []byte(cert), 0o600)
	if err != nil {
		return fmt.Errorf("writing cert data: %e", err)
	}

	var outBuf bytes.Buffer
	cmd := exec.Command(
		"rcodesign",
		"sign",
		"--p12-file", certPath,
		"--p12-password", certPwd,
		pkgPath,
	)
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf
	if err := cmd.Run(); err != nil {
		fmt.Println(outBuf.String())
		return fmt.Errorf("rcodesign: %w", err)
	}
	return nil
}

func rNotarizeStaple(path, apiIssuer, apiKey string) error {
	var outBuf bytes.Buffer
	cmd := exec.Command("rcodesign",
		"notarize",
		"--api-issuer", apiIssuer,
		"--api-key", apiKey,
		"--staple",
		path,
	)
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf
	if err := cmd.Run(); err != nil {
		fmt.Println(outBuf.String())
		return fmt.Errorf("rcodesign notarize: %w", err)
	}
	return nil
}
