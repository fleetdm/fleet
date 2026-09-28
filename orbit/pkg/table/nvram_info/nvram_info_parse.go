package nvram_info

import (
	"strconv"
	"strings"
)

// amfiEnabledFromNVRAM parses `nvram -p` output (one "name<TAB>value" per line).
// Like XNU's boot-arg parsing, any non-zero value (decimal or 0x-prefixed hex)
// or a bare flag disables AMFI.
func amfiEnabledFromNVRAM(nvramOutput string) string {
	for line := range strings.SplitSeq(nvramOutput, "\n") {
		name, bootArgs, ok := strings.Cut(line, "\t")
		if !ok || name != "boot-args" {
			continue
		}
		for arg := range strings.FieldsSeq(bootArgs) {
			key, val, hasVal := strings.Cut(arg, "=")
			if key != "amfi_get_out_of_my_way" {
				continue
			}
			if !hasVal {
				return "0"
			}
			// Unparseable values are treated as disabling AMFI so the CIS check fails closed.
			if n, err := strconv.ParseUint(val, 0, 64); err != nil || n != 0 {
				return "0"
			}
		}
	}
	return "1"
}
