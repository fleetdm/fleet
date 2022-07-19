package packaging

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func RSign(pkgPath, cert string) error {
	certPwd, ok := os.LookupEnv("MACOS_DEVID_CERTIFICATE_PASSWORD")
	if !ok {
		return errors.New("MACOS_DEVID_CERTIFICATE_PASSWORD must be set in environment")
	}

	certPath := filepath.Join(os.TempDir(), "cert.p12")
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

func RNotarizeStaple(path string) error {
	apiIssuer, ok := os.LookupEnv("AC_API_ISSUER")
	if !ok {
		return errors.New("AC_API_ISSUER must be set in environment")
	}

	apiKey, ok := os.LookupEnv("AC_API_KEY")
	if !ok {
		return errors.New("AC_API_KEY must be set in environment")
	}

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
