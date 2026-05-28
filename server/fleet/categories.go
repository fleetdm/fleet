package fleet

type SelfServiceCategory struct {
	ID      uint   `json:"id" db:"id"`
	Name    string `json:"name" db:"name"`
	FleetID uint   `json:"fleet_id" db:"fleet_id"`
	UpdateCreateTimestamps
}
