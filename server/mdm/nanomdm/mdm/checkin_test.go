package mdm

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestInvalidMessageTypeError(t *testing.T) {
	invalidMessage := `<?xml version="1.0" encoding="UTF-8"?>
	<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
	<plist version="1.0">
	<dict>
		<key>MessageType</key>
		<string>OldManYellsAtCloud.gif</string>
	</dict>
	</plist>
	`
	_, err := DecodeCheckin([]byte(invalidMessage))
	if err == nil {
		t.Fatal("wanted error but is nil")
	}
	if !errors.Is(err, ErrUnrecognizedMessageType) {
		t.Error("incorrect error type")
	}
}

func TestAuthenticate(t *testing.T) {
	for _, test := range []struct {
		filename string
		UDID     string
		Topic    string
	}{
		{
			"testdata/Authenticate.1.plist",
			"663b07bb783e9ade1dae4fbb92ea12afc0ce5b69",
			"com.apple.mgmt.External.e0bd1eac-1f17-4c8e-8a63-dd17d3dd35d9",
		},
		{
			"testdata/Authenticate.2.plist",
			"66ADE930-5FDF-5EC4-8429-15640684C489",
			"com.apple.mgmt.External.e0bd1eac-1f17-4c8e-8a63-dd17d3dd35d9",
		},
	} {
		test := test
		t.Run(filepath.Base(test.filename), func(t *testing.T) {
			t.Parallel()
			b, err := ioutil.ReadFile(test.filename)
			if err != nil {
				t.Fatal(err)
			}
			r, err := DecodeCheckin(b)
			if err != nil {
				t.Fatal(err)
			}
			a, ok := r.(*Authenticate)
			if !ok {
				t.Fatal("incorrect type")
			}
			if msg, have, want := "incorrect UDID", a.UDID, test.UDID; have != want {
				t.Errorf("%s: %q, want: %q", msg, have, want)
			}
			if msg, have, want := "incorrect Topic", a.Topic, test.Topic; have != want {
				t.Errorf("%s: %q, want: %q", msg, have, want)
			}
		})
	}
}

func TestTokenUpdate(t *testing.T) {
	for _, test := range []struct {
		filename string
		UDID     string
		Topic    string
	}{
		{
			"testdata/TokenUpdate.1.plist",
			"663b07bb783e9ade1dae4fbb92ea12afc0ce5b69",
			"com.apple.mgmt.External.e0bd1eac-1f17-4c8e-8a63-dd17d3dd35d9",
		},
		{
			"testdata/TokenUpdate.2.plist",
			"66ADE930-5FDF-5EC4-8429-15640684C489",
			"com.apple.mgmt.External.e0bd1eac-1f17-4c8e-8a63-dd17d3dd35d9",
		},
	} {
		test := test
		t.Run(filepath.Base(test.filename), func(t *testing.T) {
			t.Parallel()
			b, err := ioutil.ReadFile(test.filename)
			if err != nil {
				t.Fatal(err)
			}
			r, err := DecodeCheckin(b)
			if err != nil {
				t.Fatal(err)
			}
			a, ok := r.(*TokenUpdate)
			if !ok {
				t.Fatal("incorrect type")
			}
			if msg, have, want := "incorrect UDID", a.UDID, test.UDID; have != want {
				t.Errorf("%s: %q, want: %q", msg, have, want)
			}
			if msg, have, want := "incorrect Topic", a.Topic, test.Topic; have != want {
				t.Errorf("%s: %q, want: %q", msg, have, want)
			}
		})
	}
}

func TestGetTokenMAID(t *testing.T) {
	test := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>MessageType</key>
	<string>GetToken</string>
	<key>UDID</key>
	<string>test</string>
	<key>TokenServiceType</key>
	<string>com.apple.maid</string>
</dict>
</plist>
`
	m, err := DecodeCheckin([]byte(test))
	if err != nil {
		t.Fatal(err)
	}
	msg, ok := m.(*GetToken)
	if !ok {
		t.Fatal("incorrect decoded check-in message type")
	}
	if err := msg.Validate(); err != nil {
		t.Fatal(err)
	}
	if msg, want, have := "invalid UDID", "test", msg.UDID; have != want {
		t.Errorf("%s: %q, want: %q", msg, have, want)
	}
	if msg, want, have := "invalid TokenServiceType", "com.apple.maid", msg.TokenServiceType; have != want {
		t.Errorf("%s: %q, want: %q", msg, have, want)
	}
}
