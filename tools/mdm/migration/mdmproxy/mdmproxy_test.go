package main

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Test_udidFromRequestBody(t *testing.T) {
	type args struct {
		body []byte
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "apple example",
			// Modified from https://developer.apple.com/documentation/devicemanagement/implementing_device_management/sending_mdm_commands_to_a_device
			args: args{[]byte(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"> 
<dict>
    <key>UDID</key>
    <string>EFCAF06F-127C-42EA-BF01-E1923A836991</string>
    <key>CommandUUID</key>
    <string>9F09D114-BCFD-42AD-A974-371AA7D6256E</string>
    <key>Status</key>
    <string>Acknowledged</string> 
</dict>
</plist>
`)},
			want: "EFCAF06F-127C-42EA-BF01-E1923A836991",
		},
		{
			name: "fleet example",
			args: args{[]byte(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Status</key>
	<string>Idle</string>
	<key>UDID</key>
	<string>419D33EC-06E6-558D-AD52-601BA1867730</string>
</dict>
</plist>
`)},
			want: "419D33EC-06E6-558D-AD52-601BA1867730",
		},
		{
			name: "empty request",
			args: args{[]byte("")},
			want: "",
		},
		{
			name:    "invalid plist",
			args:    args{[]byte("<")},
			wantErr: true,
		},
		{
			name: "plist missing udid",
			args: args{[]byte(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Status</key>
	<string>Idle</string>
</dict>
</plist>
`)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := udidFromRequestBody(tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("udidFromRequestBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("udidFromRequestBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Run the function with the given percentage and return the number of included UDIDs
func countUdidIncludedByPercentage(percentage, runs int) int {
	included := 0
	for i := 0; i < runs; i++ {
		if udidIncludedByPercentage(uuid.NewString(), percentage) {
			included++
		}
	}
	return included
}

func Test_udidIncludedByPercentageNone(t *testing.T) {
	const percentage int = 0
	const runs int = 100000
	included := countUdidIncludedByPercentage(percentage, runs)
	require.Equal(t, 0, included, "expected no UDIDs to be included")
}

func Test_udidIncludedByPercentageAll(t *testing.T) {
	const percentage int = 100
	const runs int = 100000
	included := countUdidIncludedByPercentage(percentage, runs)
	require.Equal(t, runs, included, "expected all UDIDs to be included")
}

func Test_udidIncludedByPercentage(t *testing.T) {
	tests := []struct {
		percentage int
	}{
		{1}, {5}, {10}, {25}, {42}, {50}, {75}, {95},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.percentage), func(t *testing.T) {
			const runs int = 100000
			included := countUdidIncludedByPercentage(tt.percentage, runs)
			percentage := float64(included) / float64(runs)
			// Test is nondeterministic, so assert that the actual value is within 1% of the
			// expected value (to avoid flakiness). In 1000 runs this did not flake once.
			require.InDelta(t, float64(tt.percentage)/100, percentage, .01)
		})
	}
}
