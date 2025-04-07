package mdm

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestCommandAndReportResults(t *testing.T) {
	for _, test := range []struct {
		filename    string
		UDID        string
		Status      string
		CommandUUID string
	}{
		{
			"testdata/DeviceInformation.1.plist",
			"66ADE930-5FDF-5EC4-8429-15640684C489",
			"Acknowledged",
			"76eda240-5488-4989-8339-f2ae160113c4",
		},
	} {
		test := test
		t.Run(filepath.Base(test.filename), func(t *testing.T) {
			t.Parallel()
			b, err := ioutil.ReadFile(test.filename)
			if err != nil {
				t.Fatal(err)
			}
			a, err := DecodeCommandResults(b)
			if err != nil {
				t.Fatal(err)
			}
			if msg, have, want := "incorrect UDID", a.UDID, test.UDID; have != want {
				t.Errorf("%s: %q, want: %q", msg, have, want)
			}
			if msg, have, want := "incorrect Status", a.Status, test.Status; have != want {
				t.Errorf("%s: %q, want: %q", msg, have, want)
			}
			if msg, have, want := "incorrect CommandUUID", a.CommandUUID, test.CommandUUID; have != want {
				t.Errorf("%s: %q, want: %q", msg, have, want)
			}
		})
	}
}
