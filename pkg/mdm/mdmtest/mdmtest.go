// Package mdmtest contains types and methods useful for testing MDM servers.
package mdmtest

// TestMDMClientOption allows configuring a TestMDMClient.
type TestMDMClientOption func(*TestAppleMDMClient)

// TestMDMClientDebug configures the TestMDMClient to run in debug mode.
func TestMDMClientDebug() TestMDMClientOption {
	return func(c *TestAppleMDMClient) {
		c.debug = true
	}
}
