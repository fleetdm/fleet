package nvram_info

import (
	"strconv"
	"strings"
)

// amfiEnabledFromNVRAM parses `nvram -p` output (one "name<TAB>value" per line).
// Like XNU's boot-arg parsing, any non-zero value or a bare flag disables AMFI.
func amfiEnabledFromNVRAM(nvramOutput string) string {
	for line := range strings.SplitSeq(nvramOutput, "\n") {
		name, bootArgs, ok := strings.Cut(line, "\t")
		if !ok || name != "boot-args" {
			continue
		}
		for arg := range strings.FieldsFuncSeq(bootArgs, isBootArgSep) {
			key, val, _ := strings.Cut(arg, "=")
			if key != "amfi_get_out_of_my_way" {
				continue
			}
			// A bare flag (empty val) and an unparseable value both fail to parse;
			// treat them as disabling AMFI so the CIS check fails closed.
			if n, err := parseBootArgValue(val); err != nil || n != 0 {
				return "0"
			}
		}
	}
	return "1"
}

// isBootArgSep matches XNU's isargsep: only space and tab separate boot-args.
func isBootArgSep(r rune) bool {
	return r == ' ' || r == '\t'
}

// parseBootArgValue accepts the number syntax of XNU's getval: decimal, 0x hex,
// 0b binary, or leading-0 octal. It's narrower than strconv's base 0, which also
// takes 0X, 0B, 0o and digit-separating underscores; XNU treats those as strings
// and copies the raw bytes into the int, so they never read as zero.
func parseBootArgValue(val string) (uint64, error) {
	base := 10
	switch {
	case strings.HasPrefix(val, "0x"):
		base, val = 16, val[2:]
	case strings.HasPrefix(val, "0b"):
		base, val = 2, val[2:]
	case len(val) > 1 && val[0] == '0':
		base = 8
	}
	return strconv.ParseUint(val, base, 64)
}
