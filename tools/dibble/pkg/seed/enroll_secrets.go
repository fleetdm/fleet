package seed

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// EnrollSecrets seeds a per-team enroll secret — the credential a fleetd
// agent presents to join a team. This is distinct from "Fleet secrets" /
// secret variables, which are template variables substituted into profiles
// and scripts; those are a separate seeder (not yet implemented).
//
// The global enroll secret is left alone because changing it would
// invalidate any already-enrolled host. Rotate it deliberately, not via seed.
func EnrollSecrets(c Client, log Logger, teams []Team) Result {
	res := Result{Entity: "enroll-secrets"}
	for _, t := range teams {
		secret := randomEnrollSecret()
		body := map[string]any{
			"secrets": []map[string]string{{"secret": secret}},
		}
		// PATCH replaces the team's enroll-secret list with this one. Find
		// them in the UI under Settings → [team] → Add hosts → Show enroll secret.
		err := c.Patch(fmt.Sprintf("/api/latest/fleet/fleets/%d/secrets", t.ID), body, nil)
		switch {
		case err == nil:
			res.Created++
			log.Printf("enroll secret for team=%s (id=%d): %s", t.Name, t.ID, secret)
		case IsAlreadyExists(err):
			res.Skipped++
		default:
			res.Errors = append(res.Errors, err)
		}
	}
	return res
}

func randomEnrollSecret() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Vanishingly unlikely; fall back to a fixed-but-clearly-fake value.
		return "dibble-fallback-secret"
	}
	return "dibble-" + hex.EncodeToString(b[:])
}
