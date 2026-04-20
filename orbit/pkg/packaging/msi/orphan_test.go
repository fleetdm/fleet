package msi_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging/msi"
	"github.com/stretchr/testify/require"
)

func TestOrphanStreams(t *testing.T) {
	goodStreams := readStreams(t, "/tmp/msi_rebuild/rebuilt.msi")
	goStreams := readStreams(t, "/tmp/test_output.msi")
	
	// Find streams in Go MSI that don't exist in rebuilt
	extra := make(map[string][]byte)
	for name, data := range goStreams {
		if _, exists := goodStreams[name]; !exists {
			extra[name] = data
		}
	}
	
	fmt.Printf("Extra streams in Go MSI:\n")
	for name, data := range extra {
		decoded := ""
		for _, c := range name {
			o := int(c)
			if o == 0x4840 { decoded += "Table." } else if o >= 0x3800 && o < 0x4800 { decoded += "(encoded)" } else { decoded += string(c) }
		}
		fmt.Printf("  %q (%s) = %d bytes\n", name, decoded, len(data))
	}
	
	// Test D: Rebuilt streams + ONLY extra Go streams added
	cw := msi.NewCFBWriterForTest()
	for name, data := range goodStreams {
		cw.AddStreamForTest(name, data)
	}
	for name, data := range extra {
		cw.AddStreamForTest(name, data)
	}
	path := "/Users/lucas/git/fleet/D_rebuilt_plus_extras.msi"
	out, _ := os.Create(path)
	require.NoError(t, cw.WriteToForTest(out))
	out.Close()
	fmt.Printf("\nCreated %s\n", path)
	
	// Test E: Rebuilt streams WITHOUT extra streams (just orbit.cab added)
	cw2 := msi.NewCFBWriterForTest()
	for name, data := range goodStreams {
		cw2.AddStreamForTest(name, data)
	}
	// Add orbit.cab from Go MSI (needed by Media table)
	if cabData, ok := goStreams["orbit.cab"]; ok {
		cw2.AddStreamForTest("orbit.cab", cabData)
	}
	path2 := "/Users/lucas/git/fleet/E_rebuilt_plus_cab.msi"
	out2, _ := os.Create(path2)
	require.NoError(t, cw2.WriteToForTest(out2))
	out2.Close()
	fmt.Printf("Created %s\n", path2)
}
