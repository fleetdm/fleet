package falcon_kernel_check

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParseStatusErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		status         string
		expectedResult map[string]string
	}{
		{
			name: "no status",
		},
		{
			name:   "bad output",
			status: "\n\n\n\n",
		},
		{
			name:   "no supported string",
			status: "Host OS 5.13.0-51-generic #58~20.04.1-Ubuntu SMP Tue Jun 14 11:29:12 UTC 2022 might be supported. idk lol.",
		},
		{
			name:   "no sensor version",
			status: "Host OS 5.13.0-51-generic #58~20.04.1-Ubuntu SMP Tue Jun 14 11:29:12 UTC 2022 is supported",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseStatus(tt.status)
			require.Error(t, err, "parseStatus")
		})
	}
}

func Test_ParseStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		status         string
		expectedResult map[string]string
	}{
		{
			name:           "is supported",
			status:         "Host OS 5.13.0-51-generic #58~20.04.1-Ubuntu SMP Tue Jun 14 11:29:12 UTC 2022 is supported by Sensor version 14006.",
			expectedResult: map[string]string{"kernel": "5.13.0-51-generic #58~20.04.1-Ubuntu SMP Tue Jun 14 11:29:12 UTC 2022", "supported": "1", "sensor_version": "14006"},
		},
		{
			name:           "is not supported",
			status:         "Host OS Linux 5.15.0-46-generic #49~20.04.1-Ubuntu SMP Thu Aug 4 19:15:44 UTC 2022 is not supported by Sensor version 14006.",
			expectedResult: map[string]string{"kernel": "Linux 5.15.0-46-generic #49~20.04.1-Ubuntu SMP Thu Aug 4 19:15:44 UTC 2022", "supported": "0", "sensor_version": "14006"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := parseStatus(tt.status)
			require.NoError(t, err, "parseStatus")

			assert.Equal(t, tt.expectedResult, data)
		})
	}
}
