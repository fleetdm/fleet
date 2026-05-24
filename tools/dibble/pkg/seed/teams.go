package seed

import (
	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/themes"
)

// Team is the minimal team representation we need to chain seeders. Many
// downstream seeders (policies, profiles, scripts) need the integer ID.
type Team struct {
	ID   uint
	Name string
}

type teamCreateResp struct {
	Team struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
	// The newer API renames "team" → "fleet" in some responses.
	Fleet struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	} `json:"fleet"`
}

// Teams creates `count` teams (aka fleets) and returns the resulting list.
// Existing teams with the same name are looked up so callers always get an ID.
func Teams(c Client, log Logger, theme themes.Theme, count int) ([]Team, Result) {
	res := Result{Entity: "teams"}
	teams := make([]Team, 0, count)
	for i := 0; i < count; i++ {
		name := themes.TeamName(theme, i)
		body := map[string]any{"name": name}
		var resp teamCreateResp
		err := c.Post("/api/latest/fleet/fleets", body, &resp)
		id := resp.Team.ID
		if id == 0 {
			id = resp.Fleet.ID
		}
		switch {
		case err == nil:
			res.Created++
			teams = append(teams, Team{ID: id, Name: name})
			log.Printf("team %s (id=%d)", name, id)
		case IsAlreadyExists(err):
			res.Skipped++
			// Best-effort lookup so downstream seeders can scope by team.
			if got, lookupErr := findTeamByName(c, name); lookupErr == nil {
				teams = append(teams, got)
			}
		default:
			res.Errors = append(res.Errors, err)
		}
	}
	return teams, res
}

type listTeamsResp struct {
	Teams []struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	} `json:"teams"`
	Fleets []struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	} `json:"fleets"`
}

func findTeamByName(c Client, name string) (Team, error) {
	var resp listTeamsResp
	if err := c.Get("/api/latest/fleet/fleets?per_page=500", &resp); err != nil {
		return Team{}, err
	}
	list := resp.Teams
	if len(list) == 0 {
		for _, f := range resp.Fleets {
			list = append(list, struct {
				ID   uint   `json:"id"`
				Name string `json:"name"`
			}{ID: f.ID, Name: f.Name})
		}
	}
	for _, t := range list {
		if t.Name == name {
			return Team{ID: t.ID, Name: t.Name}, nil
		}
	}
	return Team{}, errTeamNotFound{name: name}
}

type errTeamNotFound struct{ name string }

func (e errTeamNotFound) Error() string { return "team not found: " + e.name }
