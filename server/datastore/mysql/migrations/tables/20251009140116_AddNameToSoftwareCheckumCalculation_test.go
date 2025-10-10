package tables

import "testing"

func TestUp_20251009140116(t *testing.T) {
	db := applyUpToPrev(t)

	sql := `INSERT INTO software
		(name, version, source, bundle_identifier, ` + "`release`" + `, arch, vendor, extension_for, extension_id, checksum, title_id)
	VALUES
		(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	h := md5.New() //nolint:gosec
	cols := []string{sw.Name, sw.Version, sw.Source, sw.BundleIdentifier, sw.Release, sw.Arch, sw.Vendor, sw.ExtensionFor, sw.ExtensionID}
	_, err := fmt.Fprint(h, strings.Join(cols, "\x00"))
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil

	// Apply current migration.
	applyNext(t, db)

	//
	// Check data, insert new entries, e.g. to verify migration is safe.
	//
	// ...
}
