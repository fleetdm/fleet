package variables

import (
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/require"
)

// payloadCorpus is shared with the execution tests: each entry either executed
// under plain substitution or exercises a quoting edge.
func payloadCorpus() map[string]string {
	return map[string]string{ //nolint:gosec // G101: injection payloads, not credentials
		"plain":         "Engineering",
		"backtick":      "Eng`id`",
		"cmd-subst":     "Eng$(id)",
		"semicolon":     "Eng; id",
		"pipe":          "Eng | id",
		"embedded-sq":   "Eng'; id; echo '",
		"lone-sq":       "'",
		"only-sq":       "''''",
		"backslash":     `Eng\'; id; #`,
		"trailing-bs":   `Eng\`,
		"quote-soup":    "Eng'\"`id`\"'",
		"newline":       "Eng\nid",
		"crlf":          "Eng\r\nid",
		"tab-space":     "  Eng\tOps  ",
		"dollar-var":    "$HOME and ${PATH}",
		"fleet-secret":  "$FLEET_SECRET_FOO",
		"nested-var":    "$FLEET_VAR_HOST_UUID",
		"glob":          "*",
		"ifs":           "a${IFS}b",
		"non-ascii":     "Ops — Zürich 🙂",
		"smart-quotes":  "Ann’s ‘Team’",
		"cjk-lead-byte": "情報システム部",
		"hangul":        "정보시스템부",
		"empty":         "",
		"long":          strings.Repeat("A", 4096),
	}
}

func TestPosixQuote(t *testing.T) {
	require.Equal(t, `'abc'`, PosixQuote("abc"))
	require.Equal(t, `''`, PosixQuote(""))
	require.Equal(t, `''\'''`, PosixQuote("'"))
	require.Equal(t, `'Ann'\''s'`, PosixQuote("Ann's"))

	for name, value := range payloadCorpus() {
		t.Run(name, func(t *testing.T) {
			got := PosixQuote(value)
			require.True(t, strings.HasPrefix(got, "'"))
			require.True(t, strings.HasSuffix(got, "'"))
			// the outer pair, plus three quotes per '\'' escape sequence
			require.Equal(t, 2+3*strings.Count(value, "'"), strings.Count(got, "'"))
		})
	}
}

func TestPowerShellCharArray(t *testing.T) {
	require.Equal(t, "''", PowerShellCharArray(""))
	require.Equal(t, "([char[]](65,66) -join '')", PowerShellCharArray("AB"))

	for name, value := range payloadCorpus() {
		t.Run(name, func(t *testing.T) {
			got := PowerShellCharArray(value)

			// no byte of the value reaches the script as source text
			for _, r := range got {
				require.Contains(t, "0123456789,()[] -join'char", string(r),
					"unexpected character %q in %q", r, got)
			}

			if value == "" {
				return
			}
			inner := strings.TrimSuffix(strings.TrimPrefix(got, "([char[]]("), ") -join '')")
			var units []uint16
			for f := range strings.SplitSeq(inner, ",") {
				u, err := strconv.ParseUint(f, 10, 16)
				require.NoError(t, err)
				units = append(units, uint16(u))
			}
			require.Equal(t, value, string(utf16.Decode(units)))
		})
	}
}

func TestPreamble(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		require.Empty(t, Preamble(nil, DialectPOSIX))
		require.Empty(t, Preamble(map[string]string{}, DialectPowerShell))
	})

	t.Run("posix is sorted and locale-guarded", func(t *testing.T) {
		got := Preamble(map[string]string{"HOST_UUID": "u", "HOST_HARDWARE_SERIAL": "s"}, DialectPOSIX)
		require.Equal(t,
			"__fleet_lc=${LC_ALL-}; __fleet_lg=${LANG-}; LC_ALL=C; LANG=C\n"+
				"export FLEET_VAR_HOST_HARDWARE_SERIAL='s' FLEET_VAR_HOST_UUID='u'\n"+
				"LC_ALL=${__fleet_lc}; LANG=${__fleet_lg}; unset __fleet_lc __fleet_lg\n", got)
	})

	// a missing guard is invisible on a UTF-8-only machine
	t.Run("posix always emits the locale guard", func(t *testing.T) {
		for name, value := range payloadCorpus() {
			got := Preamble(map[string]string{"HOST_UUID": value}, DialectPOSIX)
			require.Contains(t, got, "LC_ALL=C; LANG=C", name)
			require.Contains(t, got, "LC_ALL=${__fleet_lc}", name)
		}
	})

	t.Run("powershell is one CRLF line per variable", func(t *testing.T) {
		got := Preamble(map[string]string{"HOST_UUID": "AB", "HOST_PLATFORM": "w"}, DialectPowerShell)
		require.Equal(t,
			"$FLEET_VAR_HOST_PLATFORM = ([char[]](119) -join '')\r\n"+
				"$FLEET_VAR_HOST_UUID = ([char[]](65,66) -join '')\r\n", got)
	})
}

func TestInsertPreamble(t *testing.T) {
	const pre = "PRE\n"

	for _, tc := range []struct {
		name     string
		contents string
		want     string
	}{
		{"no shebang", "echo hi\n", "PRE\necho hi\n"},
		{"sh shebang", "#!/bin/sh\necho hi\n", "#!/bin/sh\nPRE\necho hi\n"},
		{"env bash shebang", "#!/usr/bin/env bash\necho hi\n", "#!/usr/bin/env bash\nPRE\necho hi\n"},
		{"shebang with args", "#!/bin/sh -e\necho hi\n", "#!/bin/sh -e\nPRE\necho hi\n"},
		{"crlf", "#!/bin/sh\r\necho hi\r\n", "#!/bin/sh\r\nPRE\necho hi\r\n"},
		{"shebang only, no newline", "#!/bin/sh", "#!/bin/sh\nPRE\n"},
		{"comment first line", "# not a shebang\necho hi\n", "PRE\n# not a shebang\necho hi\n"},
		{"empty", "", "PRE\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := InsertPreamble(tc.contents, pre, DialectPOSIX)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	t.Run("empty preamble is a no-op", func(t *testing.T) {
		got, err := InsertPreamble("#!/bin/sh\necho hi\n", "", DialectPOSIX)
		require.NoError(t, err)
		require.Equal(t, "#!/bin/sh\necho hi\n", got)
	})

	t.Run("powershell prepends", func(t *testing.T) {
		got, err := InsertPreamble("Write-Output hi\r\n", pre, DialectPowerShell)
		require.NoError(t, err)
		require.Equal(t, "PRE\nWrite-Output hi\r\n", got)
	})
}

// A preamble above a param() block, a BOM, or a using statement is a parse error.
func TestInsertPreamblePowerShellPlacement(t *testing.T) {
	const pre = "PRE\r\n"
	const bom = "\ufeff"

	for _, tc := range []struct {
		name     string
		contents string
		want     string // "" means the insert must be refused
	}{
		{"leading param", "param($Foo)\r\n", ""},
		{"uppercase param", "PARAM($Foo)\r\n", ""},
		{"attribute then param", "[CmdletBinding()]\r\nparam($Foo)\r\n", ""},
		{"two attributes then param", "[CmdletBinding()]\r\n[OutputType([int])]\r\nparam($Foo)\r\n", ""},
		{"comments above param", "# c\r\n#Requires -Version 5\r\nparam($Foo)\r\n", ""},
		{"help block above param", "<#\r\n.SYNOPSIS\r\n#>\r\nparam($Foo)\r\n", ""},
		{"using above param", "using namespace System.Text\r\nparam($Foo)\r\n", ""},
		{"bom above param", bom + "param($Foo)\r\n", ""},

		{"ordinary script", "Write-Output hi\r\n", pre + "Write-Output hi\r\n"},
		{"empty", "", pre},
		{"comment only", "# nothing\r\n", pre + "# nothing\r\n"},
		{"requires only", "#Requires -Version 5\r\nWrite-Output hi\r\n",
			pre + "#Requires -Version 5\r\nWrite-Output hi\r\n"},
		{"help block, no param", "<#\r\n.SYNOPSIS\r\n#>\r\nWrite-Output hi\r\n",
			pre + "<#\r\n.SYNOPSIS\r\n#>\r\nWrite-Output hi\r\n"},
		{"unterminated help block", "<#\r\nWrite-Output hi\r\n", pre + "<#\r\nWrite-Output hi\r\n"},
		{"param inside function", "function F { param($a) $a }\r\n", pre + "function F { param($a) $a }\r\n"},
		{"parameter-like name", "paramX($Foo)\r\n", pre + "paramX($Foo)\r\n"},
		{"type accelerator", "[Net.ServicePointManager]::SecurityProtocol = 1\r\n",
			pre + "[Net.ServicePointManager]::SecurityProtocol = 1\r\n"},
		{"array literal", "[int[]]$a = 1,2\r\n", pre + "[int[]]$a = 1,2\r\n"},

		// moving the BOM off line 1 breaks the script
		{"bom", bom + "Write-Output hi\r\n", bom + pre + "Write-Output hi\r\n"},
		{"bom and comment", bom + "# c\r\nWrite-Output hi\r\n", bom + pre + "# c\r\nWrite-Output hi\r\n"},

		// using statements must precede every other statement
		{"using", "using namespace System.Text\r\nWrite-Output hi\r\n",
			"using namespace System.Text\r\n" + pre + "Write-Output hi\r\n"},
		{"two using", "using namespace System.Text\r\nusing namespace System.IO\r\nWrite-Output hi\r\n",
			"using namespace System.Text\r\nusing namespace System.IO\r\n" + pre + "Write-Output hi\r\n"},
		{"comment then using", "# c\r\nusing namespace System.Text\r\nWrite-Output hi\r\n",
			"# c\r\nusing namespace System.Text\r\n" + pre + "Write-Output hi\r\n"},
		{"bom and using", bom + "using namespace System.Text\r\nWrite-Output hi\r\n",
			bom + "using namespace System.Text\r\n" + pre + "Write-Output hi\r\n"},
		{"using without trailing newline", "using namespace System.Text",
			"using namespace System.Text\r\n" + pre},
		{"using-like name", "usingX 1\r\n", pre + "usingX 1\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := InsertPreamble(tc.contents, pre, DialectPowerShell)
			if tc.want == "" {
				require.ErrorIs(t, err, ErrPowerShellLeadingParamBlock)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
