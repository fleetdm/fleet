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

func TestGoCFBWith25Entries(t *testing.T) {
	// Read count_18 streams (works with our Go CFB, 24 entries)
	streams := readStreams(t, "/Users/lucas/git/fleet/count_18.msi")
	
	// Add dummy streams to push to 25, 26, 27 entries
	for extra := 1; extra <= 3; extra++ {
		cw := msi.NewCFBWriterForTest()
		for name, data := range streams {
			cw.AddStreamForTest(name, data)
		}
		for i := range extra {
			cw.AddStreamForTest(fmt.Sprintf("dummy%d", i), []byte("test data padding"))
		}
		
		var buf bytes.Buffer
		require.NoError(t, cw.WriteToForTest(&buf))
		
		path := fmt.Sprintf("/Users/lucas/git/fleet/go_cfb_25plus_%d.msi", len(streams)+extra+1)
		os.WriteFile(path, buf.Bytes(), 0o644)
		
		// Verify readable
		doc, err := comdoc.ReadFile(bytes.NewReader(buf.Bytes()))
		require.NoError(t, err)
		entries, _ := doc.ListDir(nil)
		doc.Close()
		
		fmt.Printf("Created %s: %d bytes, %d streams (%d dir entries)\n",
			path, buf.Len(), len(entries), len(entries)+1)
	}
}
