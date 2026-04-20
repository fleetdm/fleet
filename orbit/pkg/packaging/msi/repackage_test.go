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

func TestRepackageRebuiltMSI(t *testing.T) {
	// Read streams from the working rebuilt MSI
	f, err := os.Open("/tmp/msi_rebuild/rebuilt.msi")
	if err != nil { t.Skip("rebuilt MSI not found") }
	doc, err := comdoc.ReadFile(f)
	require.NoError(t, err)
	entries, _ := doc.ListDir(nil)
	
	type stream struct {
		name string
		data []byte
	}
	var streams []stream
	for _, e := range entries {
		if e.Type == comdoc.DirStream {
			r, _ := doc.ReadStream(e)
			var buf bytes.Buffer
			buf.ReadFrom(r)
			streams = append(streams, stream{name: e.Name(), data: buf.Bytes()})
		}
	}
	doc.Close()
	f.Close()

	// Write using our Go CFB writer
	var out bytes.Buffer
	cw := msi.NewCFBWriterForTest()
	for _, s := range streams {
		cw.AddStreamForTest(s.name, s.data)
	}
	require.NoError(t, cw.WriteToForTest(&out))
	
	// Save for Windows testing
	os.WriteFile("/tmp/go-repackaged.msi", out.Bytes(), 0o644)
	fmt.Printf("Wrote /tmp/go-repackaged.msi (%d bytes)\n", out.Len())
	
	// Verify readable
	doc2, err := comdoc.ReadFile(bytes.NewReader(out.Bytes()))
	require.NoError(t, err)
	entries2, _ := doc2.ListDir(nil)
	require.Equal(t, len(streams), len(entries2))
	doc2.Close()
}
