package contract

// EnrollOrbitRequest is the request Orbit instances use to enroll to Fleet.
type EnrollOrbitRequest struct {
	// EnrollSecret is the secret to authenticate the enroll request.
	EnrollSecret string `json:"enroll_secret"`
	// HardwareUUID is the device's hardware UUID.
	HardwareUUID string `json:"hardware_uuid"`
	// HardwareSerial is the device's serial number.
	HardwareSerial string `json:"hardware_serial"`
	// Hostname is the device's hostname.
	Hostname string `json:"hostname"`
	// Platform is the device's platform as defined by osquery.
	Platform string `json:"platform"`
	// PlatformLike is the device's platform_like as defined by osquery.
	PlatformLike string `json:"platform_like"`
	// OsqueryIdentifier holds the identifier used by osquery.
	// If not set, then the hardware UUID is used to match orbit and osquery.
	OsqueryIdentifier string `json:"osquery_identifier"`
	// ComputerName is the device's friendly name (optional).
	ComputerName string `json:"computer_name"`
	// HardwareModel is the device's hardware model.
	HardwareModel string `json:"hardware_model"`
}

type SSOURLRequest struct {
	HostUUID  string `json:"host_uuid"`
	Initiator string `json:"initiator"`
}
