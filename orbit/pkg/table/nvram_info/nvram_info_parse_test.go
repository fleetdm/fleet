package nvram_info

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAMFIEnabledFromNVRAM(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   string
	}{
		{"no boot-args", "SystemAudioVolume\t%80\n", "1"},
		{"empty boot-args", "boot-args\t\n", "1"},
		{"decimal one", "boot-args\tamfi_get_out_of_my_way=1\n", "0"},
		{"hex one", "boot-args\tamfi_get_out_of_my_way=0x1\n", "0"},
		{"other non-zero", "boot-args\tamfi_get_out_of_my_way=0xff\n", "0"},
		{"bare flag", "boot-args\tamfi_get_out_of_my_way\n", "0"},
		{"decimal zero", "boot-args\tamfi_get_out_of_my_way=0\n", "1"},
		{"hex zero", "boot-args\tamfi_get_out_of_my_way=0x0\n", "1"},
		{"unparseable value", "boot-args\tamfi_get_out_of_my_way=yes\n", "0"},
		{"among other args", "SystemAudioVolume\t%80\nboot-args\t-v amfi_get_out_of_my_way=0x1 debug=0x144\n", "0"},
		{"similarly named arg", "boot-args\tamfi_get_out_of_my_way_not=1\n", "1"},
		{"not in boot-args", "some-var\tamfi_get_out_of_my_way=1\nboot-args\t-v\n", "1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, amfiEnabledFromNVRAM(c.output))
		})
	}
}
