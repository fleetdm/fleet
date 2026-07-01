package processes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChannelWriterFormatAndScrub(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fleet-serve.log")
	cw, err := openChannelWriter(path, logFileMaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	cw.write(1000, "stdout", "Authorization: Bearer abc123secret")
	cw.write(1001, "stderr", "col1\tcol2") // embedded tab
	cw.flush()
	cw.close()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)

	// Tab-delimited: ts\tstream\tmsg.
	if !strings.HasPrefix(body, "1000\tstdout\t") {
		t.Errorf("unexpected line format:\n%s", body)
	}
	// Secret scrubbed.
	if strings.Contains(body, "abc123secret") || !strings.Contains(body, "Bearer [redacted]") {
		t.Errorf("secret not scrubbed:\n%s", body)
	}
	// Embedded tab in the message replaced with spaces (so the format stays parseable).
	if !strings.Contains(body, "col1    col2") {
		t.Errorf("embedded tab not replaced:\n%s", body)
	}
}

func TestChannelWriterRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.log")
	cw, err := openChannelWriter(path, 10) // tiny threshold to force rotation
	if err != nil {
		t.Fatal(err)
	}
	cw.write(1, "stderr", "first line is well over ten bytes")  // bytes now > 10
	cw.write(2, "stderr", "second line after rotation trigger") // rotate fires at start of this write
	cw.flush()
	cw.close()

	rotated := path + ".1"
	if _, err := os.Stat(rotated); err != nil {
		t.Fatalf("expected rotated file %s: %v", rotated, err)
	}
	first, _ := os.ReadFile(rotated)
	if !strings.Contains(string(first), "first line") {
		t.Errorf("rotated file should hold the first line:\n%s", first)
	}
	cur, _ := os.ReadFile(path)
	if !strings.Contains(string(cur), "second line") || strings.Contains(string(cur), "first line") {
		t.Errorf("current file should hold only the second line:\n%s", cur)
	}
}
