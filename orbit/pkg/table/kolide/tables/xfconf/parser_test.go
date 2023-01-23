//go:build linux
// +build linux

package xfconf

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_readChannelXml(t *testing.T) {
	t.Parallel()

	type readTestCase struct {
		filePath       string
		expectedResult ChannelXML
	}

	testCases := []readTestCase{
		{
			filePath: "./testdata/thunar-volman.xml",
			expectedResult: ChannelXML{
				XMLName: xml.Name{
					Local: "channel",
				},
				ChannelName: "thunar-volman",
				Properties: []Property{
					{
						Name: "automount-media",
						Type: "empty",
						Properties: []Property{
							{
								Name:  "enabled",
								Type:  "bool",
								Value: "false",
							},
						},
					},
					{
						Name: "automount-drives",
						Type: "empty",
						Properties: []Property{
							{
								Name:  "enabled",
								Type:  "bool",
								Value: "false",
							},
						},
					},
					{
						Name: "autobrowse",
						Type: "empty",
						Properties: []Property{
							{
								Name:  "enabled",
								Type:  "bool",
								Value: "false",
							},
						},
					},
					{
						Name: "autoopen",
						Type: "empty",
						Properties: []Property{
							{
								Name:  "enabled",
								Type:  "bool",
								Value: "false",
							},
						},
					},
				},
			},
		},
		{
			filePath: "./testdata/xfce4-power-manager.xml",
			expectedResult: ChannelXML{
				XMLName: xml.Name{
					Local: "channel",
				},
				ChannelName: "xfce4-power-manager",
				Properties: []Property{
					{
						Name: "xfce4-power-manager",
						Type: "empty",
						Properties: []Property{
							{
								Name: "power-button-action",
								Type: "empty",
							},
							{
								Name:  "lock-screen-suspend-hibernate",
								Type:  "bool",
								Value: "true",
							},
						},
					},
				},
			},
		},
		{
			filePath: "./testdata/xfce4-session.xml",
			expectedResult: ChannelXML{
				XMLName: xml.Name{
					Local: "channel",
				},
				ChannelName: "xfce4-session",
				Properties: []Property{
					{
						Name: "general",
						Type: "empty",
						Properties: []Property{
							{
								Name:  "FailsafeSessionName",
								Type:  "string",
								Value: "Failsafe",
							},
							{
								Name: "LockCommand",
								Type: "string",
							},
						},
					},
					{
						Name: "sessions",
						Type: "empty",
						Properties: []Property{
							{
								Name: "Failsafe",
								Type: "empty",
								Properties: []Property{
									{
										Name:  "IsFailsafe",
										Type:  "bool",
										Value: "true",
									},
									{
										Name:  "Count",
										Type:  "int",
										Value: "5",
									},
									{
										Name: "Client0_Command",
										Type: "array",
										Values: []ArrayValue{
											{
												Type:  "string",
												Value: "xfwm4",
											},
										},
									},
									{
										Name:  "Client0_Priority",
										Type:  "int",
										Value: "15",
									},
									{
										Name:  "Client0_PerScreen",
										Type:  "bool",
										Value: "false",
									},
									{
										Name: "Client1_Command",
										Type: "array",
										Values: []ArrayValue{
											{
												Type:  "string",
												Value: "xfsettingsd",
											},
										},
									},
									{
										Name:  "Client1_Priority",
										Type:  "int",
										Value: "20",
									},
									{
										Name:  "Client1_PerScreen",
										Type:  "bool",
										Value: "false",
									},
									{
										Name: "Client2_Command",
										Type: "array",
										Values: []ArrayValue{
											{
												Type:  "string",
												Value: "xfce4-panel",
											},
										},
									},
									{
										Name:  "Client2_Priority",
										Type:  "int",
										Value: "25",
									},
									{
										Name:  "Client2_PerScreen",
										Type:  "bool",
										Value: "false",
									},
									{
										Name: "Client3_Command",
										Type: "array",
										Values: []ArrayValue{
											{
												Type:  "string",
												Value: "Thunar",
											},
											{
												Type:  "string",
												Value: "--daemon",
											},
										},
									},
									{
										Name:  "Client3_Priority",
										Type:  "int",
										Value: "30",
									},
									{
										Name:  "Client3_PerScreen",
										Type:  "bool",
										Value: "false",
									},
									{
										Name: "Client4_Command",
										Type: "array",
										Values: []ArrayValue{
											{
												Type:  "string",
												Value: "xfdesktop",
											},
										},
									},
									{
										Name:  "Client4_Priority",
										Type:  "int",
										Value: "35",
									},
									{
										Name:  "Client4_PerScreen",
										Type:  "bool",
										Value: "false",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.filePath, func(t *testing.T) {
			t.Parallel()
			result, err := readChannelXml(tt.filePath)
			require.NoError(t, err, "did not expect error parsing xml")
			require.Equal(t, tt.expectedResult, result)
		})
	}
}
