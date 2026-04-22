package example

// --- team / teams should be flagged ---

type teamsBad struct {
	A string `json:"team"`        // want `json tag "team": uses deprecated "team"/"teams"`
	B string `json:"teams"`       // want `json tag "teams": uses deprecated "team"/"teams"`
	C string `json:"team_id"`     // want `json tag "team_id": uses deprecated "team"/"teams"`
	D string `url:"host_team_id"` // want `url tag "host_team_id": uses deprecated "team"/"teams"`
	E string `query:"my_teams"`   // want `query tag "my_teams": uses deprecated "team"/"teams"`
	F string `json:"numTeams"`    // want `json tag "numTeams": uses deprecated "team"/"teams"`
	G string `json:"team,omitempty"` // want `json tag "team": uses deprecated "team"/"teams"`
}

// --- query / queries as part of a larger name should be flagged ---

type queriesBad struct {
	H string `json:"query_id"`       // want `json tag "query_id": uses "query"/"queries" as part of a name`
	I string `json:"saved_queries"`  // want `json tag "saved_queries": uses "query"/"queries" as part of a name`
	J string `url:"my_query"`        // want `url tag "my_query": uses "query"/"queries" as part of a name`
	K string `json:"queryName"`      // want `json tag "queryName": uses "query"/"queries" as part of a name`
	L string `json:"scheduled_query_id"` // want `json tag "scheduled_query_id": uses "query"/"queries" as part of a name`
}

// --- allowed ---

type okStruct struct {
	M string `json:"fleet_id"`
	N string `json:"report_id"`
	O string `json:"query"`
	P string `json:"queries"`
	Q string `json:"osquery_version"`
	R string `json:"osquery"`
	S string `json:"stream_name"`
	T string `json:"query,omitempty"`
	U string `db:"team_id"` // db tag is not checked
	V string `json:"-"`
}
