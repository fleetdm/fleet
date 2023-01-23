package main

import (
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/keyidentifier"
	"github.com/kolide/kit/logutil"
)

func main() {
	for _, path := range os.Args[1:] {
		if err := testIdentifyFile(path); err != nil {
			fmt.Printf("ERROR on %s: %+v\n", path, err)
		}
	}
}

func testIdentifyFile(path string) error {
	kIdentifer, _ := keyidentifier.New(keyidentifier.WithLogger(logutil.NewCLILogger(true)))

	ki, err := kIdentifer.IdentifyFile(path)
	if err != nil {
		return err
	}

	fmt.Printf("%s\t%s\n", ki.Parser, path)
	return nil
}
