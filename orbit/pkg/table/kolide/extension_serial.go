//go:build !fakeserial
// +build !fakeserial

package osquery

func serialForRow(row map[string]string) string {
	return row["hardware_serial"]
}
