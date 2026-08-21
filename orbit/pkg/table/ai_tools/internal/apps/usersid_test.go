package apps

import "testing"

func TestIsRealUserHive(t *testing.T) {
	const localUser = "S-1-5-21-1111111111-2222222222-3333333333-1001"

	cases := []struct {
		name string
		want bool
	}{
		{localUser, true},
		{"S-1-5-21-1111111111-2222222222-3333333333-500", true},        // built-in Administrator
		{"S-1-12-1-1234567890-1234567890-1234567890-1234567890", true}, // Entra ID account
		{"s-1-5-21-1111111111-2222222222-3333333333-1001", true},       // hive names are case-insensitive

		// The interactive user's own class registrations, not a second user.
		{localUser + "_Classes", false},

		// Service and machine accounts always have a loaded hive, and none of
		// them installs a desktop app.
		{"S-1-5-18", false}, // NT AUTHORITY\SYSTEM
		{"S-1-5-19", false}, // LOCAL SERVICE
		{"S-1-5-20", false}, // NETWORK SERVICE
		{"S-1-5-80-3139157870-2983391045-3678747466-658725712-1809340420", false}, // service SID
		{"S-1-5-90-0-1", false}, // window manager
		{"S-1-5-96-0-0", false}, // font driver host

		{".DEFAULT", false}, // the default user profile template
		{"", false},
		{"NotASid", false},
		{"S-1-5-21", false}, // prefix alone, no account RID
	}

	for _, c := range cases {
		if got := isRealUserHive(c.name); got != c.want {
			t.Errorf("isRealUserHive(%q)=%v want %v", c.name, got, c.want)
		}
	}
}
