package fsutil

import "testing"

// A local (non-well-known) account SID: a grant to it is not a world grant.
const sidLocalUser = "S-1-5-21-1111111111-2222222222-3333333333-1001"

func TestWorldPermFromACEs(t *testing.T) {
	allow := func(sid string, mask uint32) aceEntry {
		return aceEntry{SID: sid, Allow: true, Mask: mask}
	}
	deny := func(sid string, mask uint32) aceEntry {
		return aceEntry{SID: sid, Mask: mask}
	}

	cases := []struct {
		name      string
		aces      []aceEntry
		wantRead  bool
		wantWrite bool
	}{
		{
			name: "empty DACL grants nobody anything",
			aces: nil,
		},
		{
			// icacls <file> /grant Everyone:F — the reproduction from the bug report.
			name:      "Everyone full control",
			aces:      []aceEntry{allow(sidEveryone, fileAllAccess)},
			wantRead:  true,
			wantWrite: true,
		},
		{
			name:     "Everyone read-only",
			aces:     []aceEntry{allow(sidEveryone, fileGenericRead)},
			wantRead: true,
		},
		{
			name:      "Authenticated Users write",
			aces:      []aceEntry{allow(sidAuthenticatedUsers, fileWriteData)},
			wantWrite: true,
		},
		{
			name:      "GENERIC_ALL counts as both read and write",
			aces:      []aceEntry{allow(sidEveryone, genericAll)},
			wantRead:  true,
			wantWrite: true,
		},
		{
			// Being able to rewrite the DACL is being able to grant yourself write.
			name:      "WRITE_DAC alone counts as write",
			aces:      []aceEntry{allow(sidEveryone, stdWriteDAC)},
			wantWrite: true,
		},
		{
			name:      "DELETE alone counts as write",
			aces:      []aceEntry{allow(sidEveryone, stdDelete)},
			wantWrite: true,
		},
		{
			// Canonical DACL order puts deny first; it must win over the later allow
			// for the rights it names — here only FILE_READ_DATA, icacls (RD).
			name: "deny read ahead of allow full leaves write only",
			aces: []aceEntry{
				deny(sidEveryone, fileReadData),
				allow(sidEveryone, fileAllAccess),
			},
			wantWrite: true,
		},
		{
			// icacls /deny <sid>:(WD,AD) /grant <sid>:(F). The deny names only the
			// two data-write rights, so Everyone keeps DELETE and WRITE_DAC from the
			// allow and can still replace the file or re-ACL it into writability.
			// Letting the deny settle write wholesale would report this as safe.
			name: "partial write deny leaves the escalation rights a later allow grants",
			aces: []aceEntry{
				deny(sidEveryone, fileWriteData|fileAppendData),
				allow(sidEveryone, fileAllAccess),
			},
			wantRead:  true,
			wantWrite: true,
		},
		{
			// icacls /deny Everyone:(F) — a deny that does name every right.
			name: "deny full ahead of allow full grants nothing",
			aces: []aceEntry{
				deny(sidEveryone, fileAllAccess),
				allow(sidEveryone, fileAllAccess),
			},
		},
		{
			// A deny and an allow can name the same rights in different
			// vocabularies. GENERIC_ALL stands for FILE_ALL_ACCESS, so this deny
			// settles every right the following allow asks for.
			name: "generic deny ahead of specific allow grants nothing",
			aces: []aceEntry{
				deny(sidEveryone, genericAll),
				allow(sidEveryone, fileAllAccess),
			},
		},
		{
			// The same in reverse: the deny names specific rights and the allow uses
			// the generic alias for them, so it adds nothing.
			name: "specific deny ahead of generic allow grants nothing",
			aces: []aceEntry{
				deny(sidEveryone, fileAllAccess),
				allow(sidEveryone, genericAll),
			},
		},
		{
			// GENERIC_WRITE maps to FILE_GENERIC_WRITE, which carries none of DELETE,
			// WRITE_DAC or WRITE_OWNER — so like the (WD,AD) case above, the allow's
			// escalation rights survive the deny.
			name: "generic write deny leaves the escalation rights a later allow grants",
			aces: []aceEntry{
				deny(sidEveryone, genericWrite),
				allow(sidEveryone, fileAllAccess),
			},
			wantRead:  true,
			wantWrite: true,
		},
		{
			name: "inherit-only ACE does not apply to the object itself",
			aces: []aceEntry{
				{SID: sidEveryone, Allow: true, InheritOnly: true, Mask: fileAllAccess},
			},
		},
		{
			name: "grant to a specific local account is not a world grant",
			aces: []aceEntry{allow(sidLocalUser, fileAllAccess)},
		},
		{
			// A normal user-profile file: SYSTEM, Administrators, and the owner.
			name: "typical user profile ACL is not world-accessible",
			aces: []aceEntry{
				allow("S-1-5-18", fileAllAccess),     // NT AUTHORITY\SYSTEM
				allow("S-1-5-32-544", fileAllAccess), // BUILTIN\Administrators
				allow(sidLocalUser, fileAllAccess),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := worldPermFromACEs(c.aces)
			if !got.Known {
				t.Error("Known=false; a DACL we successfully read is a known posture")
			}
			if got.WorldReadable != c.wantRead {
				t.Errorf("WorldReadable=%v want %v", got.WorldReadable, c.wantRead)
			}
			if got.WorldWritable != c.wantWrite {
				t.Errorf("WorldWritable=%v want %v", got.WorldWritable, c.wantWrite)
			}
		})
	}
}
