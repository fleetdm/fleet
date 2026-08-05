package msi

import (
	_ "embed"
	"fmt"
	"os"
	"time"
)

//go:embed wixca/WixCA.dll
var wixCADLL []byte

//go:embed wixca/WixCA_A64.dll
var wixCAA64DLL []byte

// Build assembles the fleetd MSI from the prepared orbit root directory and
// writes it to outPath. rootDir is the directory formerly harvested by
// heat.exe (secret.txt, osquery.flags, certs.pem, bin/..., etc. — prepared
// by packaging.BuildMSI).
func Build(opt Options, rootDir, outPath string) error {
	if _, err := summaryTemplate(opt.Architecture); err != nil {
		return err
	}

	h, err := harvestRoot(rootDir)
	if err != nil {
		return fmt.Errorf("harvest orbit root: %w", err)
	}

	productCode, err := newGUID()
	if err != nil {
		return err
	}
	packageCode, err := newGUID()
	if err != nil {
		return err
	}

	db := newDatabase()
	if err := db.addValidationRows(); err != nil {
		return fmt.Errorf("build validation rows: %w", err)
	}
	filesInCabOrder, err := populate(db, opt, h, productCode)
	if err != nil {
		return fmt.Errorf("build installer database: %w", err)
	}

	// Build the embedded cabinet.
	cabFiles := make([]cabFile, 0, len(filesInCabOrder))
	for _, f := range filesInCabOrder {
		info, err := os.Stat(f.Path)
		if err != nil {
			return fmt.Errorf("stat %s: %w", f.Path, err)
		}
		cabFiles = append(cabFiles, cabFile{
			Key:     f.FileID,
			Path:    f.Path,
			Size:    f.Size,
			ModTime: info.ModTime(),
			Hash:    f.Hash,
		})
	}
	cabData, err := writeCab(cabFiles)
	if err != nil {
		return fmt.Errorf("build cabinet: %w", err)
	}

	template, err := summaryTemplate(opt.Architecture)
	if err != nil {
		return err
	}
	summary := buildSummaryInfo(template, packageCode, time.Now())

	dbStreams, err := db.streams()
	if err != nil {
		return fmt.Errorf("serialize installer database: %w", err)
	}

	// Assemble the CFB streams in the order WiX creates them: catalog
	// streams, summary information, the cabinet, the Binary streams, then
	// the table streams (which db.streams already orders).
	var streams []cfbStream
	appendDB := func(from, to int) {
		for _, s := range dbStreams[from:to] {
			streams = append(streams, cfbStream{Name: encodeStreamName(s.Name), Data: s.Data})
		}
	}
	appendDB(0, 3) // !_Tables, !_StringPool, !_StringData
	streams = append(streams, cfbStream{Name: "\x05SummaryInformation", Data: summary})
	streams = append(streams, cfbStream{Name: encodeStreamName("cab1.cab"), Data: cabData})
	if opt.Architecture == "arm64" {
		streams = append(streams, cfbStream{Name: encodeStreamName("Binary.WixCA_A64"), Data: wixCAA64DLL})
	}
	streams = append(streams, cfbStream{Name: encodeStreamName("Binary.WixCA"), Data: wixCADLL})
	appendDB(3, len(dbStreams))

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", outPath, err)
	}
	defer out.Close()
	if err := writeCFB(out, streams); err != nil {
		return fmt.Errorf("write msi: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %s: %w", outPath, err)
	}
	return nil
}
