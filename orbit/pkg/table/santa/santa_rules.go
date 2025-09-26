//go:build darwin
// +build darwin

// Package santa implements the tables for getting Santa data
// (logs, rules, status) on macOS.
//
// Santa is an open source macOS endpoint security system with
// binary whitelisting and blacklisting capabilities.
// Based on https://github.com/allenhouchins/fleet-extensions/tree/main/santa
package santa

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

const (
	busyTimeout        = 5000 // in milliseconds
	defaultRulesDBPath = "/var/db/santa/rules.db"
)

// RuleType represents the type of Santa rule
type ruleType int

const (
	ruleTypeBinary ruleType = iota
	ruleTypeCertificate
	ruleTypeTeamID
	ruleTypeSigningID
	ruleTypeCDHash
	ruleTypeUnknown
)

// RuleState represents the state of a Santa rule
type ruleState int

const (
	ruleStateAllowlist ruleState = iota
	ruleStateBlocklist
	ruleStateUnknown
)

// RuleEntry represents a Santa rule entry
type ruleEntry struct {
	ruleType      ruleType
	ruleState     ruleState
	identifier    string // SHA256, Team ID, Signing ID, CDHash value
	customMessage string
}

func RulesColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("identifier"),
		table.TextColumn("type"),
		table.TextColumn("state"),
		table.TextColumn("custom_message"),
	}
}

// generateSantaRules generates data for the santa_rules table
func GenerateRules(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	rules, err := collectSantaRules(ctx)
	if err != nil {
		return nil, err
	}

	var results []map[string]string
	for _, rule := range rules {
		row := map[string]string{
			"identifier":     rule.identifier,
			"type":           getRuleTypeName(rule.ruleType),
			"state":          getRuleStateName(rule.ruleState),
			"custom_message": rule.customMessage,
		}
		results = append(results, row)
	}

	return results, nil
}

// GetRuleTypeName returns the string representation of a rule type
func getRuleTypeName(ruleType ruleType) string {
	switch ruleType {
	case ruleTypeBinary:
		return "Binary"
	case ruleTypeCertificate:
		return "Certificate"
	case ruleTypeTeamID:
		return "TeamID"
	case ruleTypeSigningID:
		return "SigningID"
	case ruleTypeCDHash:
		return "CDHash"
	default:
		return "Unknown"
	}
}

func collectSantaRules(ctx context.Context) ([]ruleEntry, error) {
	return collectSantaRulesFromPath(ctx, defaultRulesDBPath, busyTimeout)
}

// collectSantaRules reads Santa rules from the database
func collectSantaRulesFromPath(ctx context.Context, dbPath string, timeoutMS int) ([]ruleEntry, error) {
	// Check if Santa database exists
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			log.Warn().Msg("Santa database not found")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to stat Santa database: %w", err)
	}

	// Open read-only with a busy timeout so SQLite retries internally before returning SQLITE_BUSY.
	// Read-only mode avoids issues with WAL mode databases.
	dsn := fmt.Sprintf("file:%s?mode=ro&_busy_timeout=%d", dbPath, timeoutMS)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open read-only database: %w", err)
	}
	defer db.Close()

	// Query the rules table with all available columns
	rows, err := db.QueryContext(ctx, `
		SELECT 
			identifier,
			state,
			type,
			custommsg
		FROM rules
		ORDER BY identifier
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	var rules []ruleEntry
	for rows.Next() {
		var identifier, customMsg sql.NullString
		var stateInt, typeInt sql.NullInt64

		if err := rows.Scan(&identifier, &stateInt, &typeInt, &customMsg); err != nil {
			log.Warn().Err(err).Msg("failed to scan rule row")
			continue
		}

		if !identifier.Valid {
			continue
		}

		rule := ruleEntry{
			identifier: identifier.String,
			ruleType:   ruleTypeUnknown,
			ruleState:  ruleStateUnknown,
		}

		// Parse rule type from integer
		if typeInt.Valid {
			rule.ruleType = getRuleTypeFromInt(int(typeInt.Int64))
		}

		// Parse rule state from integer
		if stateInt.Valid {
			rule.ruleState = getRuleStateFromInt(int(stateInt.Int64))
		}

		// Parse custom message
		if customMsg.Valid {
			rule.customMessage = customMsg.String
		}

		rules = append(rules, rule)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rules: %w", err)
	}

	return rules, nil
}

// getRuleTypeFromInt converts integer type value to RuleType
func getRuleTypeFromInt(typeInt int) ruleType {
	switch typeInt {
	case 1000:
		return ruleTypeBinary
	case 2000:
		return ruleTypeCertificate
	case 3000:
		return ruleTypeTeamID
	case 4000:
		return ruleTypeSigningID
	case 5000:
		return ruleTypeCDHash
	default:
		return ruleTypeUnknown
	}
}

// getRuleStateFromInt converts integer state value to RuleState
func getRuleStateFromInt(stateInt int) ruleState {
	switch stateInt {
	case 1:
		return ruleStateAllowlist
	case 2:
		return ruleStateBlocklist
	default:
		return ruleStateUnknown
	}
}

// GetRuleStateName returns the string representation of a rule state
func getRuleStateName(ruleState ruleState) string {
	switch ruleState {
	case ruleStateAllowlist:
		return "Allowlist"
	case ruleStateBlocklist:
		return "Blocklist"
	default:
		return "Unknown"
	}
}
