//go:build !windows

package variables

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Every case defines two variables, the corpus value first (names are emitted in
// sorted order) and a live payload second. A single-variable case can't detect
// the multi-byte break: it needs a following quote to swallow.
const execPayloadVar = "HOST_UUID"

const valueVar = "HOST_HARDWARE_SERIAL"

type execShape struct {
	name string
	// body must contain OUT, replaced with the round-trip target path
	body string
	// want transforms the input value into what the body should write
	want func(value string) string
}

func execShapes() []execShape {
	self := func(v string) string { return v }
	return []execShape{
		{"double-quoted", "#!/bin/sh\nprintf %s \"$FLEET_VAR_HOST_HARDWARE_SERIAL\" > OUT\n", self},
		{"braced", "#!/bin/sh\nprintf %s \"${FLEET_VAR_HOST_HARDWARE_SERIAL}\" > OUT\n", self},
		{"no-shebang", "printf %s \"$FLEET_VAR_HOST_HARDWARE_SERIAL\" > OUT\n", self},
		{"child-shell", "#!/bin/sh\nsh -c 'printf %s \"$FLEET_VAR_HOST_HARDWARE_SERIAL\"' > OUT\n", self},
		{"path-concat", "#!/bin/sh\nprintf %s \"/var/tmp/c/$FLEET_VAR_HOST_HARDWARE_SERIAL/x\" > OUT\n",
			func(v string) string { return "/var/tmp/c/" + v + "/x" }},
	}
}

// byteFaithfulLocale reports whether the shell reproduces a UTF-8 value
// byte-for-byte in this locale. In an EUC locale it does not, for any variable
// holding UTF-8, preamble or environment alike; that is the admin's locale
// acting on the admin's script, so only safety is asserted there.
func byteFaithfulLocale(locale string) bool {
	return locale == "C" || strings.HasSuffix(locale, ".UTF-8")
}

func availableLocales() []string {
	locales := []string{"C"}
	out, err := exec.Command("locale", "-a").Output()
	if err != nil {
		return locales
	}
	installed := make(map[string]struct{})
	for line := range strings.SplitSeq(string(out), "\n") {
		installed[strings.TrimSpace(line)] = struct{}{}
	}
	// libc decoders for these accept 0x27 as a trail byte
	for _, l := range []string{"en_US.UTF-8", "ja_JP.eucJP", "ko_KR.eucKR", "zh_CN.eucCN"} {
		if _, ok := installed[l]; ok {
			locales = append(locales, l)
		}
	}
	return locales
}

func availableShells() []string {
	shells := []string{"/bin/sh"}
	for _, s := range []string{"/bin/bash", "/bin/zsh"} {
		if _, err := os.Stat(s); err == nil {
			shells = append(shells, s)
		}
	}
	return shells
}

func runScript(t *testing.T, shell, script, locale string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "script")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o600))
	cmd := exec.Command(shell, path)
	cmd.Env = append(os.Environ(), "LC_ALL="+locale, "LANG="+locale)
	_ = cmd.Run() // a payload that fails to parse is still a pass
}

func TestPreambleNeverExecutesValues(t *testing.T) {
	shells := availableShells()
	locales := availableLocales()

	for name, value := range payloadCorpus() {
		t.Run(name, func(t *testing.T) {
			for _, shape := range execShapes() {
				for _, shell := range shells {
					for _, locale := range locales {
						dir := t.TempDir()
						marker := filepath.Join(dir, "MARKER")
						out := filepath.Join(dir, "out")

						vars := map[string]string{
							valueVar:       value,
							execPayloadVar: "x$(touch " + marker + ");#",
						}
						script, err := InsertPreamble(
							strings.ReplaceAll(shape.body, "OUT", out),
							Preamble(vars, DialectPOSIX), DialectPOSIX)
						require.NoError(t, err)

						runScript(t, shell, script, locale)

						require.NoFileExists(t, marker,
							"value executed: shape=%s shell=%s locale=%s", shape.name, shell, locale)

						if !byteFaithfulLocale(locale) {
							continue
						}
						got, err := os.ReadFile(out)
						require.NoError(t, err, "shape=%s shell=%s locale=%s", shape.name, shell, locale)
						require.Equal(t, shape.want(value), string(got),
							"value did not round-trip: shape=%s shell=%s locale=%s", shape.name, shell, locale)
					}
				}
			}
		})
	}
}

// Keeps the test above honest: a harness that never detects execution would pass
// for the wrong reason.
func TestSubstitutingValuesIntoTheBodyExecutesThem(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "MARKER")
	out := filepath.Join(dir, "out")

	body := strings.ReplaceAll(execShapes()[0].body, "OUT", out)
	script := Replace(body, valueVar, "Eng`touch "+marker+"`")

	runScript(t, "/bin/sh", script, "C")

	require.FileExists(t, marker)
	got, _ := os.ReadFile(out)
	require.Equal(t, "Eng", string(got))
}

// Nothing but a backslash, "U" and hex digits reaches the source, so a value can
// neither close a string literal nor, outside one, parse at all.
func TestPythonEscapeNeverExecutesValues(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not installed")
	}

	// where a $FLEET_VAR_* token can sit; OUT is the round-trip target
	inString := map[string]string{
		"double-quoted": `open("OUT","w").write("TOKEN")`,
		"single-quoted": `open("OUT","w").write('TOKEN')`,
		"triple-quoted": `open("OUT","w").write("""TOKEN""")`,
		"f-string":      `open("OUT","w").write(f"TOKEN")`,
		"adjacent":      `open("OUT","w").write("TOKEN"[0:0] + "TOKEN")`,
	}

	// the shared corpus carries shell payloads; these are executable Python
	values := payloadCorpus()
	values["py-break-double"] = "X\")\nimport os\nos.system(\"touch <marker>\")\nprint(\""
	values["py-break-single"] = "X')\nimport os\nos.system('touch <marker>')\nprint('"
	values["py-break-triple"] = "X\"\"\")\nimport os\nos.system(\"touch <marker>\")\nprint(\"\"\""
	values["py-fstring-expr"] = "{__import__('os').system('touch <marker>')}"

	for name, value := range values {
		t.Run(name, func(t *testing.T) {
			for shape, body := range inString {
				dir := t.TempDir()
				marker := filepath.Join(dir, "MARKER")
				out := filepath.Join(dir, "out")
				value := strings.ReplaceAll(value, "<marker>", marker)

				src := "#!/usr/bin/env python3\n" + strings.NewReplacer(
					"OUT", out,
					"TOKEN", PythonEscape(value),
				).Replace(body) + "\n"

				script := filepath.Join(dir, "s.py")
				require.NoError(t, os.WriteFile(script, []byte(src), 0o600))
				_ = exec.Command(python, script).Run()

				require.NoFileExists(t, marker, "value executed: shape=%s", shape)
				got, err := os.ReadFile(out)
				require.NoError(t, err, "shape=%s", shape)
				require.Equal(t, value, string(got), "value did not round-trip: shape=%s", shape)
			}
		})
	}

	t.Run("bare code fails to parse", func(t *testing.T) {
		dir := t.TempDir()
		marker := filepath.Join(dir, "MARKER")
		script := filepath.Join(dir, "s.py")
		payload := `__import__(chr(111)+chr(115)).system("touch ` + marker + `")`
		src := "#!/usr/bin/env python3\nx = " + PythonEscape(payload) + "\n"
		require.NoError(t, os.WriteFile(script, []byte(src), 0o600))

		require.Error(t, exec.Command(python, script).Run())
		require.NoFileExists(t, marker)
	})
}

// Keeps the test above honest: substituting a value into Python source executes it.
func TestSubstitutingIntoPythonExecutesValues(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not installed")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "MARKER")
	script := filepath.Join(dir, "s.py")

	payload := "X\")\nimport os\nos.system(\"touch " + marker + "\")\nprint(\""
	src := "#!/usr/bin/env python3\n" + Replace(`print("v: $FLEET_VAR_HOST_UUID")`, "HOST_UUID", payload) + "\n"
	require.NoError(t, os.WriteFile(script, []byte(src), 0o600))
	_ = exec.Command(python, script).Run()

	require.FileExists(t, marker)
}
