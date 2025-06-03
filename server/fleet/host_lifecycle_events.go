package fleet

type HostLifecycleEventType string

const (
	HostLifecycleEventStartedMDMSetup       HostLifecycleEventType = "started_mdm_setup"
	HostLifecycleEventCompletedMDMSetup     HostLifecycleEventType = "completed_mdm_setup"
	HostLifecycleEventStartedMDMMigration   HostLifecycleEventType = "started_mdm_migration"
	HostLifecycleEventCompletedMDMMigration HostLifecycleEventType = "completed_mdm_migration"
)

func (e HostLifecycleEventType) Valid() bool {
	switch e {
	case HostLifecycleEventStartedMDMSetup,
		HostLifecycleEventCompletedMDMSetup,
		HostLifecycleEventStartedMDMMigration,
		HostLifecycleEventCompletedMDMMigration:
		return true
	default:
		return false
	}
}

type HostLifecycleEvent struct {
	ID         uint                   `db:"id"`
	HostSerial string                 `db:"host_serial"`
	HostUUID   string                 `db:"host_uuid"`
	HostID     uint                   `db:"host_id"`
	EventType  HostLifecycleEventType `db:"event_type"`
	CreateTimestamp
	ActivityID *uint `db:"activity_id"`
}
