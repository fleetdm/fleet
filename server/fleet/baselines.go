package fleet

// BaselineManifest describes an available security baseline bundle.
// These are NOT database entities — they are read from embedded Go assets.
type BaselineManifest struct {
	ID          string             `json:"id" yaml:"id"`
	Name        string             `json:"name" yaml:"name"`
	Version     string             `json:"version" yaml:"version"`
	Platform    string             `json:"platform" yaml:"platform"`
	Description string             `json:"description" yaml:"description"`
	Categories  []BaselineCategory `json:"categories" yaml:"categories"`
}

// BaselineCategory groups related profiles, policies, and scripts within a baseline.
type BaselineCategory struct {
	Name     string   `json:"name" yaml:"name"`
	Profiles []string `json:"profiles" yaml:"profiles"`
	Policies []string `json:"policies" yaml:"policies"`
	Scripts  []string `json:"scripts" yaml:"scripts"`
}

// ApplyBaselineRequest is the request payload for applying a baseline to a team.
type ApplyBaselineRequest struct {
	BaselineID string `json:"baseline_id"`
	TeamID     uint   `json:"team_id"`
}

// ApplyBaselineResponse is the response payload after applying a baseline.
type ApplyBaselineResponse struct {
	BaselineID      string   `json:"baseline_id"`
	TeamID          uint     `json:"team_id"`
	ProfilesCreated []string `json:"profiles_created"`
	PoliciesCreated []string `json:"policies_created"`
	ScriptsCreated  []string `json:"scripts_created"`
}
