package tables

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

func init() {
	MigrationClient.AddMigration(Up_20260226220138, Down_20260226220138)
}

func Up_20260226220138(tx *sql.Tx) error {
	if err := migratePolicyBypassEnabledData(tx); err != nil {
		return fmt.Errorf("migrate policy bypass enabled field: %w", err)
	}

	if _, err := tx.Exec("ALTER TABLE policies DROP COLUMN conditional_access_bypass_enabled"); err != nil {
		return fmt.Errorf("migrate policies drop bypass enabled field: %w", err)
	}
	return nil
}

func Down_20260226220138(tx *sql.Tx) error {
	return nil
}

func migratePolicyBypassEnabledData(tx *sql.Tx) error {
	var appConfigJson []byte
	var appConfig appConfigOktaConfig
	var enabledTeams []uint

	row := tx.QueryRow("SELECT json_value FROM app_config_json")
	if err := row.Err(); err != nil {
		return fmt.Errorf("get appconfig: %w", err)
	}

	if err := row.Scan(&appConfigJson); err != nil {
		return fmt.Errorf("scan appconfig row: %w", err)
	}

	if err := json.Unmarshal(appConfigJson, &appConfig); err != nil {
		return fmt.Errorf("unmarshal appconfig: %w", err)
	}

	// Okta not configured, move on
	if !appConfig.oktaConfigured() {
		return nil
	}

	rows, err := tx.Query("SELECT id, config FROM teams")
	if err != nil {
		return fmt.Errorf("querying teams: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uint
		var configJson []byte
		var teamConfig teamConfigOktaConfig

		if err := rows.Scan(&id, &configJson); err != nil {
			return fmt.Errorf("scanning teams: %w", err)
		}

		if err := json.Unmarshal(configJson, &teamConfig); err != nil {
			return fmt.Errorf("unmarshalling team %d config: %w", id, err)
		}

		if !teamConfig.conditionalAccessEnabled() {
			continue
		}

		enabledTeams = append(enabledTeams, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating team rows: %w", err)
	}

	if len(enabledTeams) != 0 {
		questions := make([]string, len(enabledTeams))
		for i := range questions {
			questions[i] = "?"
		}

		inQuery := "(" + strings.Join(questions, ", ") + ")"

		query := fmt.Sprintf("UPDATE policies SET critical = true WHERE team_id IN %s AND conditional_access_bypass_enabled = true", inQuery)

		args := make([]any, len(enabledTeams))
		for i, id := range enabledTeams {
			args[i] = id
		}
		if _, err := tx.Exec(query, args...); err != nil {
			return fmt.Errorf("setting policies to critical: %w", err)
		}
	}

	return nil
}

type teamConfigOktaConfig struct {
	Integrations struct {
		ConditionalAccessEnabled *bool `json:"conditional_access_enabled"`
	} `json:"integrations"`
}

func (c *teamConfigOktaConfig) conditionalAccessEnabled() bool {
	return c.Integrations.ConditionalAccessEnabled != nil && *c.Integrations.ConditionalAccessEnabled
}

type appConfigOktaConfig struct {
	ConditionalAccess *struct {
		OktaIDPID                       *string `json:"okta_idp_id"`
		OktaAssertionConsumerServiceURL *string `json:"okta_assertion_consumer_service_url"`
		OktaAudienceURI                 *string `json:"okta_audience_uri"`
		OktaCertificate                 *string `json:"okta_certificate"`
		BypassDisabled                  *bool   `json:"bypass_disabled"`
	} `json:"conditional_access"`
}

func (c *appConfigOktaConfig) oktaConfigured() bool {
	return c.ConditionalAccess != nil &&
		c.ConditionalAccess.OktaIDPID != nil && *c.ConditionalAccess.OktaIDPID != "" &&
		c.ConditionalAccess.OktaAssertionConsumerServiceURL != nil && *c.ConditionalAccess.OktaAssertionConsumerServiceURL != "" &&
		c.ConditionalAccess.OktaAudienceURI != nil && *c.ConditionalAccess.OktaAudienceURI != "" &&
		c.ConditionalAccess.OktaCertificate != nil && *c.ConditionalAccess.OktaCertificate != ""
}
