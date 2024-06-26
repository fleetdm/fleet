package mdm

import (
	"testing"
)

func TestNilResolved(t *testing.T) {
	var e *Enrollment
	_ = e.Resolved()
}

func TestResolved(t *testing.T) {
	for _, test := range []struct {
		testName  string
		e         Enrollment
		expectNil bool
		t         EnrollType
		deviceId  string
		userId    string
	}{
		{
			"empty",
			Enrollment{},
			true,
			0,
			"",
			"",
		},
		{
			"UDID",
			Enrollment{
				UDID: "a",
			},
			false,
			Device,
			"a",
			"",
		},
		{
			"UserID",
			Enrollment{
				UDID:   "b",
				UserID: "c",
			},
			false,
			User,
			"b",
			"c",
		},
		{
			"EnrollmentID",
			Enrollment{
				EnrollmentID: "d",
			},
			false,
			UserEnrollmentDevice,
			"d",
			"",
		},
		{
			"EnrollmentUserID",
			Enrollment{
				EnrollmentID:     "e",
				EnrollmentUserID: "f",
			},
			false,
			UserEnrollment,
			"e",
			"f",
		},
		{
			"SharediPad",
			Enrollment{
				UDID:          "g",
				UserID:        SharediPadUserID,
				UserShortName: "appleid@example.com",
			},
			false,
			SharediPad,
			"g",
			"appleid@example.com",
		},
	} {
		test := test
		t.Run(test.testName, func(t *testing.T) {
			r := test.e.Resolved()
			if r == nil {
				if test.expectNil != true {
					t.Error("nil received unexpectedly")
				}
				return
			}
			if msg, have, want := "wrong type", r.Type, test.t; have != want {
				t.Errorf("%s: have: %q, want: %q", msg, have, want)
			}
			if msg, have, want := "wrong device channel ID", r.DeviceChannelID, test.deviceId; have != want {
				t.Errorf("%s: have: %q, want: %q", msg, have, want)
			}
			if msg, have, want := "wrong user channel ID ", r.UserChannelID, test.userId; have != want {
				t.Errorf("%s: have: %q, want: %q", msg, have, want)
			}
		})
	}
}
