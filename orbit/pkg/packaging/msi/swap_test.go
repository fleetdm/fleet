package msi_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging/msi"
	"github.com/sassoftware/relic/v8/lib/comdoc"
	"github.com/stretchr/testify/require"
)

// TestSwapStreams takes the working rebuilt MSI and swaps in our Go-encoded
// streams one at a time to find which one causes Windows to reject the file.
func TestSwapStreams(t *testing.T) {
	// Read streams from the working rebuilt MSI
	rf, err := os.Open("/tmp/msi_rebuild/rebuilt.msi")
	if err != nil { t.Skip("rebuilt MSI not found") }
	rdoc, err := comdoc.ReadFile(rf)
	require.NoError(t, err)
	rentries, _ := rdoc.ListDir(nil)
	
	goodStreams := make(map[string][]byte)
	for _, e := range rentries {
		if e.Type == comdoc.DirStream {
			r, _ := rdoc.ReadStream(e)
			var buf bytes.Buffer
			buf.ReadFrom(r)
			goodStreams[e.Name()] = buf.Bytes()
		}
	}
	rdoc.Close()
	rf.Close()

	// Read streams from our Go MSI
	gf, err := os.Open("/tmp/test_output.msi")
	if err != nil { t.Skip("Go MSI not found") }
	gdoc, err := comdoc.ReadFile(gf)
	require.NoError(t, err)
	gentries, _ := gdoc.ListDir(nil)
	
	goStreams := make(map[string][]byte)
	for _, e := range gentries {
		if e.Type == comdoc.DirStream {
			r, _ := gdoc.ReadStream(e)
			var buf bytes.Buffer
			buf.ReadFrom(r)
			goStreams[e.Name()] = buf.Bytes()
		}
	}
	gdoc.Close()
	gf.Close()

	// For each stream that exists in both, create an MSI where ONLY that
	// stream is from our Go encoder (rest from rebuilt).
	os.MkdirAll("/tmp/swap_test", 0o755)
	
	for goName, goData := range goStreams {
		if _, exists := goodStreams[goName]; !exists {
			continue // Skip streams only in Go MSI
		}
		if bytes.Equal(goData, goodStreams[goName]) {
			continue // Skip identical streams
		}
		
		// Create MSI with all good streams except this one swapped to Go version
		cw := msi.NewCFBWriterForTest()
		for name, data := range goodStreams {
			if name == goName {
				cw.AddStreamForTest(name, goData) // Use Go version
			} else {
				cw.AddStreamForTest(name, data) // Use good version
			}
		}
		
		outPath := fmt.Sprintf("/tmp/swap_test/swap_%s.msi", sanitizeName(goName))
		out, _ := os.Create(outPath)
		require.NoError(t, cw.WriteToForTest(out))
		out.Close()
		
		fmt.Printf("Created %s (swapped stream %q: %d→%d bytes)\n",
			outPath, goName, len(goodStreams[goName]), len(goData))
	}
}

func sanitizeName(s string) string {
	var out []byte
	for _, c := range []byte(s) {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			out = append(out, c)
		} else {
			out = append(out, '_')
		}
	}
	return string(out)
}
