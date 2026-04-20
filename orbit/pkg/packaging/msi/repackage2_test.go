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

func TestRepackageWriteMSI(t *testing.T) {
	streams := readStreams(t, "/tmp/types_final.msi")
	
	cw := msi.NewCFBWriterForTest()
	for name, data := range streams {
		cw.AddStreamForTest(name, data)
	}
	
	var buf bytes.Buffer
	require.NoError(t, cw.WriteToForTest(&buf))
	
	path := "/Users/lucas/git/fleet/WriteMSI_repackaged.msi"
	os.WriteFile(path, buf.Bytes(), 0o644)
	
	doc, err := comdoc.ReadFile(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	entries, _ := doc.ListDir(nil)
	doc.Close()
	
	fmt.Printf("Created %s (%d bytes, %d streams)\n", path, buf.Len(), len(entries))
}
