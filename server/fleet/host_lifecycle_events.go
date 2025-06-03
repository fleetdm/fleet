package fleet

type HostLifecycleEvent struct {
	ID         uint   `db:"id"`
	HostSerial string `db:"host_serial"`
	HostUUID   string `db:"host_uuid"`
	HostID     uint   `db:"host_id"`
	EventType  string `db:"event_type"` // TODO enum
	CreateTimestamp
	ActivityID *uint `db:"activity_id"`
}
