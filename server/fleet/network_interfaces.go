package fleet

type NetworkInterface struct {
	UpdateCreateTimestamps
	ID uint `json:"id"`
	// HostID foreign key establishes one host to many NetworkInterface relationship
	HostID       uint   `json:"-" db:"host_id"`
	Interface    string `json:"interface"`
	IPAddress    string `json:"address" db:"ip_address"`
	Mask         string `json:"mask"`
	Broadcast    string `json:"broadcast"`
	PointToPoint string `json:"point_to_point" db:"point_to_point"`
	MAC          string `json:"mac"`
	Type         int    `json:"type"`
	MTU          int    `json:"mtu"`
	Metric       int    `json:"metric"`
	IPackets     int64  `json:"ipackets"`
	OPackets     int64  `json:"opackets"`

	IBytes  int64 `json:"ibytes"`
	OBytes  int64 `json:"obytes"`
	IErrors int64 `json:"ierrors"`

	OErrors    int64 `json:"oerrors"`
	LastChange int64 `json:"last_change" db:"last_change"`
}
