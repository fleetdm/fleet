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

func TestSwapSummaryOnly(t *testing.T) {
	// Read all streams from both MSIs
	goodStreams := readStreams(t, "/tmp/msi_rebuild/rebuilt.msi")
	goStreams := readStreams(t, "/tmp/test_output.msi")
	
	summaryName := "\x05SummaryInformation"
	
	// Test A: All rebuilt streams + Go SummaryInformation
	{
		cw := msi.NewCFBWriterForTest()
		for name, data := range goodStreams {
			if name == summaryName {
				cw.AddStreamForTest(name, goStreams[name])
			} else {
				cw.AddStreamForTest(name, data)
			}
		}
		writeTest(t, cw, "/tmp/swap_test/A_go_summary.msi")
		fmt.Println("A: rebuilt streams + Go SummaryInfo")
	}
	
	// Test B: All Go streams + rebuilt SummaryInformation
	{
		cw := msi.NewCFBWriterForTest()
		for name, data := range goStreams {
			if name == summaryName {
				cw.AddStreamForTest(name, goodStreams[name])
			} else {
				cw.AddStreamForTest(name, data)
			}
		}
		writeTest(t, cw, "/tmp/swap_test/B_rebuilt_summary.msi")
		fmt.Println("B: Go streams + rebuilt SummaryInfo")
	}
	
	// Test C: All Go streams (same as our Go MSI but through CFB rewrite)
	{
		cw := msi.NewCFBWriterForTest()
		for name, data := range goStreams {
			cw.AddStreamForTest(name, data)
		}
		writeTest(t, cw, "/tmp/swap_test/C_all_go.msi")
		fmt.Println("C: all Go streams (control)")
	}
	
	// Copy to fleet repo for easy access
	for _, f := range []string{"A_go_summary", "B_rebuilt_summary", "C_all_go"} {
		src := fmt.Sprintf("/tmp/swap_test/%s.msi", f)
		dst := fmt.Sprintf("/Users/lucas/git/fleet/%s.msi", f)
		data, _ := os.ReadFile(src)
		os.WriteFile(dst, data, 0o644)
		fmt.Printf("  → %s (%d bytes)\n", dst, len(data))
	}
}

func readStreams(t *testing.T, path string) map[string][]byte {
	f, err := os.Open(path)
	if err != nil { t.Skipf("file not found: %s", path) }
	doc, err := comdoc.ReadFile(f)
	require.NoError(t, err)
	entries, _ := doc.ListDir(nil)
	result := make(map[string][]byte)
	for _, e := range entries {
		if e.Type == comdoc.DirStream {
			r, _ := doc.ReadStream(e)
			var buf bytes.Buffer
			buf.ReadFrom(r)
			result[e.Name()] = buf.Bytes()
		}
	}
	doc.Close()
	f.Close()
	return result
}

func writeTest(t *testing.T, cw *msi.CfbWriterWrapper, path string) {
	out, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, cw.WriteToForTest(out))
	out.Close()
}
