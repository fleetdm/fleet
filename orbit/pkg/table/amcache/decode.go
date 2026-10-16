package amcache

import (
	"context"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

func decodeInt(s string) (string, bool) {
	if _, err := strconv.ParseUint(s, 10, 64); err != nil {
		return "", false
	}
	return s, true
}

// decodeBool rejects anything but the two spellings Windows uses, rather than treating non-zero
// as true, so a column pointed at the wrong value shows up as a decode failure.
func decodeBool(s string) (string, bool) {
	switch strings.ToLower(s) {
	case "1", "true":
		return "1", true
	case "0", "false":
		return "0", true
	}
	return "", false
}

// No real value matches more than one of these layouts.
var timeLayouts = [...]string{
	"1/2/2006 15:04:05", // LinkDate, application InstallDate, MsiInstallDate, DriverLastWriteTime
	"1/2/2006",          // documented for InstallDate, not observed
	"1-2-2006",          // InventoryDevicePnp dates
	"2006-1-2",          // InventoryDriverPackage.Date
}

// epochFloor rejects well-formed placeholder dates such as "1/1/1601 00:00:00", which would
// otherwise appear in a timeline as real events.
var epochFloor = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

func plausible(t time.Time) bool {
	return !t.Before(epochFloor) && t.Before(time.Now().AddDate(1, 0, 0))
}

// decodeTime converts an Amcache date string to unix seconds. Amcache records no time zone,
// so UTC is assumed. LinkDate is copied unvalidated from PE headers and is often a version
// string instead of a date, which decodes to empty.
func decodeTime(s string) (string, bool) {
	for _, layout := range timeLayouts {
		t, err := time.ParseInLocation(layout, s, time.UTC)
		if err != nil {
			continue
		}
		if !plausible(t) {
			return "", false
		}
		return strconv.FormatInt(t.Unix(), 10), true
	}
	return "", false
}

// decodeMulti normalizes a list the appraiser already comma-joined. Elements can contain commas
// themselves, so it must not try to recover element boundaries.
func decodeMulti(s string) string {
	items := strings.Split(s, ",")
	out := items[:0]
	for _, it := range items {
		if it = strings.TrimSpace(it); it != "" {
			out = append(out, it)
		}
	}
	return strings.Join(out, ",")
}

// decodeSha1 extracts the SHA-1 from a FileId or DriverId. Windows hashes only the first
// 31,457,280 bytes of the file, so for larger files it's not a whole-file hash.
func decodeSha1(s string) (string, bool) {
	switch {
	case len(s) == 44 && strings.HasPrefix(s, "0000"):
		s = s[4:]
	case len(s) != 40:
		return "", false
	}
	if _, err := hex.DecodeString(s); err != nil {
		return "", false
	}
	return strings.ToLower(s), true
}

// cell decodes one column of one record. The second result is false only for a non-empty
// value the column rejected.
func cell(c col, rec record) (string, bool) {
	switch c.kind {
	case kKey:
		return rec.name, true
	case kKeyPath:
		return strings.ReplaceAll(rec.name, "/", `\`), true
	case kKeyLower:
		return strings.ToLower(rec.name), true
	case kKeyTime:
		if rec.lastWrite == 0 {
			return "", true
		}
		return strconv.FormatInt(rec.lastWrite, 10), true
	}

	s := rec.values[c.src]
	if s == "" {
		// Absent and present-but-empty are the same fact, and neither is a decode failure.
		return "", true
	}

	switch c.kind {
	case kInt, kEpoch:
		return decodeInt(s)
	case kBool:
		return decodeBool(s)
	case kTime:
		return decodeTime(s)
	case kMulti:
		return decodeMulti(s), true
	case kSha1:
		return decodeSha1(s)
	case kLower:
		return strings.ToLower(s), true
	default:
		return s, true
	}
}

// rowsFor builds one row per selected record; sel nil means every record. Every column is
// always set, since osquery logs a line per row for each missing key. An empty text column
// renders as "" and an empty integer column as NULL.
func (t *tables) rowsFor(ctx context.Context, s spec, recs []record, sel []int32) ([]map[string]string, error) {
	n := len(recs)
	if sel != nil {
		n = len(sel)
	}
	rows := make([]map[string]string, 0, n)
	var failed map[string]int

	for i := range n {
		if i%4096 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		rec := recs[i]
		if sel != nil {
			rec = recs[sel[i]]
		}
		row := make(map[string]string, len(s.cols))
		for _, c := range s.cols {
			v, ok := cell(c, rec)
			if !ok {
				if failed == nil {
					failed = make(map[string]int)
				}
				failed[c.name]++
			}
			row[c.name] = v
		}
		rows = append(rows, row)
	}

	for name, count := range failed {
		t.warnDecode(s.table, name, count)
	}
	return rows, nil
}

// warnDecode logs once per column per process. It never logs the value: on healthy hosts some
// columns (link_time) routinely hold values that aren't dates.
func (t *tables) warnDecode(tbl, col string, n int) {
	if _, dup := t.decodeWarned.LoadOrStore(tbl+"."+col, struct{}{}); dup {
		return
	}
	t.logger.Debug().Str("table", tbl).Str("column", col).Int("count", n).
		Msg("amcache: non-empty values could not be decoded and are reported as empty; later occurrences are not logged")
}
