package tables

import "testing"

func TestUp_20260315000000(t *testing.T) {
	db := applyUpToPrev(t)

	// "Engineering" (no whitespace) — should not change.
	execNoErr(t, db, `INSERT INTO teams (name) VALUES ('Engineering')`)
	// "  Design  " (leading+trailing) — should be trimmed to "Design".
	execNoErr(t, db, `INSERT INTO teams (name) VALUES ('  Design  ')`)
	// "Marketing " (trailing) — should be trimmed to "Marketing".
	execNoErr(t, db, `INSERT INTO teams (name) VALUES ('Marketing ')`)
	// " Sales" (leading) — should be trimmed to "Sales".
	execNoErr(t, db, `INSERT INTO teams (name) VALUES (' Sales')`)
	// "Support" and " Support " — conflict after trimming.
	// "Support" stays, " Support " gets disambiguated.
	execNoErr(t, db, `INSERT INTO teams (name) VALUES ('Support')`)
	execNoErr(t, db, `INSERT INTO teams (name) VALUES (' Support ')`)
	// "  Finance" and "Finance  " — both need trimming, both conflict after trim.
	// Both should get disambiguated with their IDs.
	execNoErr(t, db, `INSERT INTO teams (name) VALUES ('Finance ')`)
	execNoErr(t, db, `INSERT INTO teams (name) VALUES ('  Finance')`)
	// Tab and newline whitespace — should be trimmed to "DevOps" and "QA".
	execNoErr(t, db, "INSERT INTO teams (name) VALUES ('\tDevOps\t')")
	execNoErr(t, db, "INSERT INTO teams (name) VALUES ('\nQA\n')")
	// Tab-only name — should become "Unnamed team (<id>)".
	execNoErr(t, db, "INSERT INTO teams (name) VALUES ('\t\t')")
	// Whitespace-only name — should become "Unnamed team (<id>)".
	execNoErr(t, db, `INSERT INTO teams (name) VALUES ('     ')`)

	// Apply current migration.
	applyNext(t, db)

	// Verify results.
	var name string

	// "Engineering" unchanged.
	err := db.QueryRow(`SELECT name FROM teams WHERE name = 'Engineering'`).Scan(&name)
	if err != nil {
		t.Fatalf("expected 'Engineering' team: %v", err)
	}

	// "  Design  " trimmed to "Design".
	err = db.QueryRow(`SELECT name FROM teams WHERE name = 'Design'`).Scan(&name)
	if err != nil {
		t.Fatalf("expected 'Design' team (trimmed from '  Design  '): %v", err)
	}

	// "Marketing " trimmed to "Marketing".
	err = db.QueryRow(`SELECT name FROM teams WHERE name = 'Marketing'`).Scan(&name)
	if err != nil {
		t.Fatalf("expected 'Marketing' team (trimmed from 'Marketing '): %v", err)
	}

	// " Sales" trimmed to "Sales".
	err = db.QueryRow(`SELECT name FROM teams WHERE name = 'Sales'`).Scan(&name)
	if err != nil {
		t.Fatalf("expected 'Sales' team (trimmed from ' Sales'): %v", err)
	}

	// "Support" unchanged.
	err = db.QueryRow(`SELECT name FROM teams WHERE name = 'Support'`).Scan(&name)
	if err != nil {
		t.Fatalf("expected 'Support' team unchanged: %v", err)
	}

	// " Support " should have been disambiguated with its ID appended.
	var disambiguatedName string
	err = db.QueryRow(`SELECT name FROM teams WHERE name LIKE 'Support (%'`).Scan(&disambiguatedName)
	if err != nil {
		t.Fatalf("expected disambiguated 'Support (ID)' team: %v", err)
	}

	// "Finance " and "Finance  " both trimmed to "Finance" — both should be disambiguated.
	var financeCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM teams WHERE name LIKE 'Finance (%'`).Scan(&financeCount)
	if err != nil {
		t.Fatalf("error checking for disambiguated Finance teams: %v", err)
	}
	if financeCount != 2 {
		t.Fatalf("expected 2 disambiguated 'Finance (ID)' teams, got %d", financeCount)
	}
	// No bare "Finance" should exist (both were padded, neither was clean).
	var bareFinanceCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM teams WHERE name = 'Finance'`).Scan(&bareFinanceCount)
	if err != nil {
		t.Fatalf("error checking for bare Finance team: %v", err)
	}
	if bareFinanceCount != 0 {
		t.Fatalf("expected 0 bare 'Finance' teams, got %d", bareFinanceCount)
	}

	// "\tDevOps\t" trimmed to "DevOps".
	err = db.QueryRow(`SELECT name FROM teams WHERE name = 'DevOps'`).Scan(&name)
	if err != nil {
		t.Fatalf("expected 'DevOps' team (trimmed from tab-wrapped): %v", err)
	}

	// "\nQA\n" trimmed to "QA".
	err = db.QueryRow(`SELECT name FROM teams WHERE name = 'QA'`).Scan(&name)
	if err != nil {
		t.Fatalf("expected 'QA' team (trimmed from newline-wrapped): %v", err)
	}

	// Whitespace-only team should have been renamed to "Unnamed team (<id>)".
	var unnamedCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM teams WHERE name LIKE 'Unnamed team (%'`).Scan(&unnamedCount)
	if err != nil {
		t.Fatalf("error checking for unnamed teams: %v", err)
	}
	if unnamedCount != 2 {
		t.Fatalf("expected 2 'Unnamed team (ID)' teams, got %d", unnamedCount)
	}

	// Verify no team names have leading or trailing whitespace (including tabs/newlines).
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM teams WHERE name REGEXP '^[[:space:]]|[[:space:]]$'`).Scan(&count)
	if err != nil {
		t.Fatalf("error checking for untrimmed names: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 teams with untrimmed names, got %d", count)
	}

	// Verify no empty team names.
	err = db.QueryRow(`SELECT COUNT(*) FROM teams WHERE name = ''`).Scan(&count)
	if err != nil {
		t.Fatalf("error checking for empty names: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 teams with empty names, got %d", count)
	}
}
