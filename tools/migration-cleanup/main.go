package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	configpkg "github.com/fleetdm/fleet/v4/server/config"
	commonmysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

const (
	exitGeneral = 1
	exitDryRun  = 2
	exitApply   = 3

	tableStatusName = "migration_status_tables"
	dataStatusName  = "migration_status_data"
)

var (
	tsRE = regexp.MustCompile(`^(\d{14})_`)

	migrationDirs = []string{
		"server/datastore/mysql/migrations/tables",
		"server/datastore/mysql/migrations/data",
	}
)

type migrationRename struct {
	oldVersionID  int64
	newVersionID  int64
	migrationType string
	oldPath       string
	newPath       string
	commitSHA     string
}

type tableRow struct {
	ID        int64 `db:"id"`
	VersionID int64 `db:"version_id"`
	IsApplied bool  `db:"is_applied"`
}

type sqlStatements struct {
	tableName           string
	versionIDRemappings [][2]int64
}

type options struct {
	checkout string
	branch   string
	output   string
	dryRun   bool
	apply    bool
	verbose  bool

	dbHost     string
	dbPort     int
	dbName     string
	dbUser     string
	dbPassword string

	tlsMode string
	tlsCA   string
	tlsCert string
	tlsKey  string
}

func main() {
	log.SetFlags(0)

	opts := options{}
	rootCmd := newRootCmd(&opts)
	configManager := configpkg.NewManager(rootCmd)
	hideUnneededFleetConfigFlags(rootCmd)

	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		if err := run(cmd.Context(), configManager, opts); err != nil {
			var exitErr exitError
			if errors.As(err, &exitErr) {
				if exitErr.message != "" {
					fmt.Fprintln(os.Stderr, exitErr.message)
				}
				os.Exit(exitErr.code)
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(exitGeneral)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitGeneral)
	}
}

type exitError struct {
	code    int
	message string
}

func (e exitError) Error() string {
	return e.message
}

func newRootCmd(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "migration-cleanup",
		Short:        "Detect and fix Fleet migration renumbering",
		SilenceUsage: true,
	}

	cmd.PersistentFlags().String("config", "", "Path to a Fleet configuration file")
	cmd.Flags().StringVarP(&opts.checkout, "checkout", "c", ".", "Path to fleetdm/fleet git checkout")
	cmd.Flags().StringVarP(&opts.branch, "branch", "b", "", "Branch name")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Write SQL to file instead of stdout")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Connect read-only and verify SQL would apply")
	cmd.Flags().BoolVar(&opts.apply, "apply", false, "Execute SQL against the database in a transaction")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Verbose/debug output")

	cmd.Flags().StringVar(&opts.dbHost, "db-host", "", "MySQL hostname")
	cmd.Flags().IntVar(&opts.dbPort, "db-port", 3306, "MySQL port")
	cmd.Flags().StringVar(&opts.dbName, "db-name", "", "MySQL database name")
	cmd.Flags().StringVar(&opts.dbUser, "db-user", "", "MySQL username")
	cmd.Flags().StringVarP(&opts.dbPassword, "db-password", "p", "", "MySQL password")

	cmd.Flags().StringVar(&opts.tlsMode, "tls-mode", "", "TLS verification mode: skip-verify, verify-ca, verify-identity")
	cmd.Flags().StringVar(&opts.tlsCA, "tls-ca", "", "CA certificate PEM path")
	cmd.Flags().StringVar(&opts.tlsCert, "tls-cert", "", "Client certificate PEM path")
	cmd.Flags().StringVar(&opts.tlsKey, "tls-key", "", "Client key PEM path")
	return cmd
}

func hideUnneededFleetConfigFlags(cmd *cobra.Command) {
	visibleMysqlFlags := map[string]struct{}{
		"mysql_protocol":            {},
		"mysql_address":             {},
		"mysql_username":            {},
		"mysql_password":            {},
		"mysql_password_path":       {},
		"mysql_database":            {},
		"mysql_tls_cert":            {},
		"mysql_tls_key":             {},
		"mysql_tls_ca":              {},
		"mysql_tls_server_name":     {},
		"mysql_tls_config":          {},
		"mysql_region":              {},
		"mysql_sts_assume_role_arn": {},
		"mysql_sts_external_id":     {},
	}
	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		if flag.Name == "config" {
			return
		}
		if _, ok := visibleMysqlFlags[flag.Name]; ok {
			return
		}
		flag.Hidden = true
	})
}

func run(ctx context.Context, configManager configpkg.Manager, opts options) error {
	if opts.branch == "" {
		return exitError{code: exitGeneral, message: "ERROR: --branch is required"}
	}
	if opts.dryRun && opts.apply {
		return exitError{code: exitGeneral, message: "ERROR: --dry-run and --apply are mutually exclusive"}
	}
	if err := validateTLSFlags(opts); err != nil {
		return exitError{code: exitGeneral, message: "ERROR: " + err.Error()}
	}

	checkout, err := filepath.Abs(opts.checkout)
	if err != nil {
		return exitError{code: exitGeneral, message: fmt.Sprintf("ERROR: %v", err)}
	}
	if st, err := os.Stat(checkout); err != nil || !st.IsDir() {
		return exitError{code: exitGeneral, message: fmt.Sprintf("ERROR: %q is not a directory", checkout)}
	}
	if opts.verbose {
		fmt.Fprintf(os.Stderr, "Checkout: %s\n", checkout)
	}

	if opts.verbose {
		fmt.Fprintln(os.Stderr, "Fetching origin...")
	}
	if _, err := git(checkout, "fetch", "origin"); err != nil {
		return exitError{code: exitGeneral, message: err.Error()}
	}

	branch, err := resolveBranch(checkout, opts.branch)
	if err != nil {
		return exitError{code: exitGeneral, message: err.Error()}
	}
	if opts.verbose {
		fmt.Fprintf(os.Stderr, "Resolved branch: %s\n", branch)
	}

	mergeBase, err := getMergeBase(checkout, branch)
	if err != nil {
		return exitError{code: exitGeneral, message: err.Error()}
	}
	if opts.verbose {
		fmt.Fprintf(os.Stderr, "Merge base: %s\n", mergeBase)
	}

	commits, err := findRenameCommits(checkout, branch, mergeBase)
	if err != nil {
		return exitError{code: exitGeneral, message: err.Error()}
	}
	if opts.verbose {
		fmt.Fprintf(os.Stderr, "Rename commits found: %d\n", len(commits))
	}

	var renames []migrationRename
	for _, sha := range commits {
		rs, err := extractRenames(checkout, sha)
		if err != nil {
			return exitError{code: exitGeneral, message: err.Error()}
		}
		if opts.verbose {
			fmt.Fprintf(os.Stderr, "  %s: %d rename(s)\n", shortSHA(sha), len(rs))
		}
		renames = append(renames, rs...)
	}
	if len(renames) == 0 {
		fmt.Println("No migration renumbering detected on this branch.")
		return nil
	}
	renames = dedupeRenames(renames)

	fmt.Fprintf(os.Stderr, "Found %d migration renumber(s):\n", len(renames))
	for _, r := range renames {
		fmt.Fprintf(os.Stderr, "  [%s] %d -> %d  (%s)\n", r.migrationType, r.oldVersionID, r.newVersionID, r.commitSHA)
	}

	var tableRows, dataRows []tableRow
	var db *sqlx.DB
	if opts.dryRun || opts.apply {
		fleetConfig := configManager.LoadConfig()
		mysqlConfig, err := writerConfig(fleetConfig.Mysql, opts)
		if err != nil {
			return exitError{code: exitGeneral, message: "ERROR: " + err.Error()}
		}
		db, err = openWriterDB(mysqlConfig)
		if err != nil {
			return exitError{code: exitGeneral, message: "ERROR: DB connection failed: " + err.Error()}
		}
		defer db.Close()

		tableRows, err = queryTableRows(ctx, db, tableStatusName)
		if err != nil {
			return exitError{code: exitGeneral, message: "ERROR: " + err.Error()}
		}
		dataRows, err = queryTableRows(ctx, db, dataStatusName)
		if err != nil {
			return exitError{code: exitGeneral, message: "ERROR: " + err.Error()}
		}
		if opts.verbose {
			fmt.Fprintf(os.Stderr, "  %s: %d rows\n", tableStatusName, len(tableRows))
			fmt.Fprintf(os.Stderr, "  %s: %d rows\n", dataStatusName, len(dataRows))
		}
	}

	statements := generateStatementGroups(renames)
	sqlText := renderSQL(statements)

	if opts.dryRun {
		clean, messages := verifyDryRun(renames, tableRows, dataRows)
		for _, msg := range messages {
			fmt.Fprintln(os.Stderr, msg)
		}
		if clean {
			fmt.Fprintln(os.Stderr, "Dry-run: SQL will apply cleanly.")
		} else {
			fmt.Fprintln(os.Stderr, "Dry-run: issues detected.")
		}
		if err := writeOutput(sqlText, opts.output); err != nil {
			return exitError{code: exitGeneral, message: "ERROR: " + err.Error()}
		}
		if !clean {
			return exitError{code: exitDryRun}
		}
		return nil
	}

	if opts.apply {
		if err := applyStatements(ctx, db, statements); err != nil {
			return exitError{code: exitApply, message: "ERROR: apply failed: " + err.Error()}
		}
		fmt.Fprintln(os.Stderr, "SQL applied successfully.")
		return nil
	}

	if err := writeOutput(sqlText, opts.output); err != nil {
		return exitError{code: exitGeneral, message: "ERROR: " + err.Error()}
	}
	return nil
}

func validateTLSFlags(opts options) error {
	switch opts.tlsMode {
	case "", "skip-verify", "verify-ca", "verify-identity":
	default:
		return fmt.Errorf("--tls-mode must be one of skip-verify, verify-ca, verify-identity")
	}
	if opts.tlsMode == "verify-ca" || opts.tlsMode == "verify-identity" {
		if opts.tlsCA == "" {
			return fmt.Errorf("--tls-ca is required for verify-ca / verify-identity")
		}
		if opts.tlsCert == "" || opts.tlsKey == "" {
			return fmt.Errorf("--tls-cert and --tls-key are required for verify-ca / verify-identity")
		}
	}
	return nil
}

func git(checkout string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", checkout}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

func resolveBranch(checkout, branch string) (string, error) {
	candidates := []string{branch, "origin/" + branch}
	var lastErr error
	for _, candidate := range candidates {
		if _, err := git(checkout, "rev-parse", "--verify", candidate); err == nil {
			return candidate, nil
		} else {
			lastErr = err
			fmt.Fprintln(os.Stderr, err)
		}
	}
	if lastErr != nil {
		return "", fmt.Errorf("ERROR: cannot resolve branch %q", branch)
	}
	return "", fmt.Errorf("ERROR: cannot resolve branch %q", branch)
}

func getMergeBase(checkout, branch string) (string, error) {
	out, err := git(checkout, "merge-base", "main", branch)
	return strings.TrimSpace(out), err
}

func findRenameCommits(checkout, branch, mergeBase string) ([]string, error) {
	args := []string{"log", "-M", "--diff-filter=R", "--format=%H", mergeBase + ".." + branch, "--"}
	args = append(args, migrationDirs...)
	out, err := git(checkout, args...)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

func extractRenames(checkout, commitSHA string) ([]migrationRename, error) {
	out, err := git(checkout, "diff-tree", "-M", "-r", "--diff-filter=R", "--name-status", "--no-commit-id", commitSHA)
	if err != nil {
		return nil, err
	}
	var renames []migrationRename
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		oldPath, newPath := parts[1], parts[2]
		oldMatch := tsRE.FindStringSubmatch(filepath.Base(oldPath))
		newMatch := tsRE.FindStringSubmatch(filepath.Base(newPath))
		if oldMatch == nil || newMatch == nil {
			continue
		}
		oldVID, err := strconv.ParseInt(oldMatch[1], 10, 64)
		if err != nil {
			return nil, err
		}
		newVID, err := strconv.ParseInt(newMatch[1], 10, 64)
		if err != nil {
			return nil, err
		}
		if oldVID == newVID || strings.HasSuffix(oldPath, "_test.go") {
			continue
		}
		var mtype string
		switch {
		case strings.Contains(oldPath, "/tables"):
			mtype = "tables"
		case strings.Contains(oldPath, "/data"):
			mtype = "data"
		default:
			continue
		}
		renames = append(renames, migrationRename{
			oldVersionID:  oldVID,
			newVersionID:  newVID,
			migrationType: mtype,
			oldPath:       oldPath,
			newPath:       newPath,
			commitSHA:     shortSHA(commitSHA),
		})
	}
	return renames, nil
}

func shortSHA(sha string) string {
	if len(sha) < 12 {
		return sha
	}
	return sha[:12]
}

func dedupeRenames(renames []migrationRename) []migrationRename {
	seen := map[string]struct{}{}
	unique := make([]migrationRename, 0, len(renames))
	for _, r := range renames {
		key := fmt.Sprintf("%s:%d:%d", r.migrationType, r.oldVersionID, r.newVersionID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, r)
	}
	return unique
}

func generateStatementGroups(renames []migrationRename) map[string][]string {
	groups := map[string][]string{}
	for _, item := range []struct {
		mtype string
		table string
	}{
		{"tables", tableStatusName},
		{"data", dataStatusName},
	} {
		var tableRenames []migrationRename
		for _, r := range renames {
			if r.migrationType == item.mtype {
				tableRenames = append(tableRenames, r)
			}
		}
		if len(tableRenames) == 0 {
			continue
		}
		stmts := computeSQLForTable(item.table, tableRenames)
		groups[item.table] = buildSQL(item.table, stmts, tableRenames)
	}
	return groups
}

func computeSQLForTable(tableName string, renames []migrationRename) sqlStatements {
	stmts := sqlStatements{tableName: tableName}
	for _, r := range renames {
		stmts.versionIDRemappings = append(stmts.versionIDRemappings, [2]int64{r.oldVersionID, r.newVersionID})
	}
	return stmts
}

func buildSQL(tableName string, stmts sqlStatements, renames []migrationRename) []string {
	lines := make([]string, 0)
	for _, pair := range stmts.versionIDRemappings {
		lines = append(lines, fmt.Sprintf("UPDATE `%s` SET version_id = %d WHERE version_id = %d;", tableName, pair[1], pair[0]))
	}
	if len(renames) == 0 {
		return lines
	}

	lines = append(lines,
		fmt.Sprintf("CREATE TEMPORARY TABLE `_fix_dups_%s` (id INT);", tableName),
		fmt.Sprintf("INSERT INTO `_fix_dups_%s` (id) SELECT id FROM `%s` WHERE (version_id, id) NOT IN (SELECT version_id, MIN(id) FROM `%s` GROUP BY version_id);", tableName, tableName, tableName),
		fmt.Sprintf("DELETE FROM `%s` WHERE id IN (SELECT id FROM `_fix_dups_%s`);", tableName, tableName),
		fmt.Sprintf("DROP TEMPORARY TABLE `_fix_dups_%s`;", tableName),
	)

	minNewVID, maxNewVID := renames[0].newVersionID, renames[0].newVersionID
	movesUp := false
	for _, r := range renames {
		if r.newVersionID < minNewVID {
			minNewVID = r.newVersionID
		}
		if r.newVersionID > maxNewVID {
			maxNewVID = r.newVersionID
		}
		if r.newVersionID > r.oldVersionID {
			movesUp = true
		}
	}

	varName := "increment_by_" + tableName
	if !movesUp {
		varName += "_shift"
	}
	maxMovedVar := "max_moved_down_" + tableName

	lines = append(lines, fmt.Sprintf("SELECT (SELECT MAX(id) FROM `%s` WHERE id > (SELECT id FROM `%s` WHERE version_id = %d)) - (SELECT id FROM `%s` WHERE version_id = %d) + 1 INTO @%s;", tableName, tableName, maxNewVID, tableName, minNewVID, varName))

	var whereClause string
	if movesUp {
		whereClause = fmt.Sprintf("WHERE version_id BETWEEN %d AND %d", minNewVID, maxNewVID)
		lines = append(lines, fmt.Sprintf("SELECT %d INTO @%s;", maxNewVID, maxMovedVar))
	} else {
		whereClause = fmt.Sprintf("WHERE id < (SELECT id FROM `%s` WHERE version_id = %d) AND version_id > %d", tableName, minNewVID, maxNewVID)
		lines = append(lines, fmt.Sprintf("SELECT MAX(version_id) INTO @%s FROM `%s` %s ;", maxMovedVar, tableName, whereClause))
	}

	lines = append(lines,
		fmt.Sprintf("UPDATE `%s` SET id = id + COALESCE(@%s, 0) WHERE version_id > @%s ORDER BY id DESC;", tableName, varName, maxMovedVar),
		fmt.Sprintf("UPDATE `%s` SET id = id + COALESCE(@%s, 0) %s ORDER BY id DESC;", tableName, varName, whereClause),
	)
	return lines
}

func renderSQL(groups map[string][]string) string {
	if len(groups) == 0 {
		return "-- No changes needed.\n"
	}
	var b strings.Builder
	b.WriteString("-- Migration renumber fix\n-- Generated by migration-cleanup\n\nSTART TRANSACTION;\n")
	for _, tableName := range []string{tableStatusName, dataStatusName} {
		lines, ok := groups[tableName]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "-- %s\n", tableName)
		for _, line := range lines {
			b.WriteString(line)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	b.WriteString("COMMIT;\n")
	return b.String()
}

func queryTableRows(ctx context.Context, db *sqlx.DB, tableName string) ([]tableRow, error) {
	var rows []tableRow
	if err := sqlx.SelectContext(ctx, db, &rows, fmt.Sprintf("SELECT id, version_id, is_applied FROM `%s` ORDER BY id", tableName)); err != nil {
		return nil, fmt.Errorf("query %s: %w", tableName, err)
	}
	return rows, nil
}

func verifyDryRun(renames []migrationRename, tableRows, dataRows []tableRow) (bool, []string) {
	var issues []string
	var messages []string
	for _, item := range []struct {
		table string
		mtype string
		rows  []tableRow
	}{
		{tableStatusName, "tables", tableRows},
		{dataStatusName, "data", dataRows},
	} {
		var tableRenames []migrationRename
		for _, r := range renames {
			if r.migrationType == item.mtype {
				tableRenames = append(tableRenames, r)
			}
		}
		if len(tableRenames) == 0 {
			continue
		}
		simulated, tableMessages, tableIssues := simulateTableSQL(item.table, item.rows, tableRenames)
		messages = append(messages, tableMessages...)
		issues = append(issues, tableIssues...)
		if len(tableIssues) == 0 {
			issues = append(issues, validateFinalTableState(item.table, simulated)...)
		}
	}
	return len(issues) == 0, append(messages, issues...)
}

func simulateTableSQL(tableName string, rows []tableRow, renames []migrationRename) ([]tableRow, []string, []string) {
	var messages []string
	var issues []string
	renameMap := map[int64]int64{}
	existing := map[int64]struct{}{}
	for _, row := range rows {
		existing[row.VersionID] = struct{}{}
	}
	for _, r := range renames {
		renameMap[r.oldVersionID] = r.newVersionID
		if _, ok := existing[r.oldVersionID]; ok {
			messages = append(messages, fmt.Sprintf("  %s: will remap %d -> %d", tableName, r.oldVersionID, r.newVersionID))
		} else {
			messages = append(messages, fmt.Sprintf("  %s: %d not present (UPDATE will be no-op)", tableName, r.oldVersionID))
		}
	}

	simulated := make([]tableRow, 0, len(rows))
	for _, row := range rows {
		newVID := row.VersionID
		if mapped, ok := renameMap[row.VersionID]; ok {
			newVID = mapped
		}
		simulated = append(simulated, tableRow{ID: row.ID, VersionID: newVID, IsApplied: row.IsApplied})
	}

	byVID := map[int64][]tableRow{}
	for _, row := range simulated {
		byVID[row.VersionID] = append(byVID[row.VersionID], row)
	}
	simulated = simulated[:0]
	for vid, vidRows := range byVID {
		sort.Slice(vidRows, func(i, j int) bool { return vidRows[i].ID < vidRows[j].ID })
		keep := vidRows[0]
		simulated = append(simulated, keep)
		if len(vidRows) > 1 {
			deleted := make([]string, 0, len(vidRows)-1)
			for _, row := range vidRows[1:] {
				deleted = append(deleted, strconv.FormatInt(row.ID, 10))
			}
			messages = append(messages, fmt.Sprintf("  %s: duplicate version_id=%d; would keep id=%d, delete ids=[%s]", tableName, vid, keep.ID, strings.Join(deleted, " ")))
		}
	}

	minNewVID, maxNewVID := renames[0].newVersionID, renames[0].newVersionID
	movesUp := false
	for _, r := range renames {
		if r.newVersionID < minNewVID {
			minNewVID = r.newVersionID
		}
		if r.newVersionID > maxNewVID {
			maxNewVID = r.newVersionID
		}
		if r.newVersionID > r.oldVersionID {
			movesUp = true
		}
	}

	minRows := rowsForVID(simulated, minNewVID)
	maxRows := rowsForVID(simulated, maxNewVID)
	var targetRows []tableRow
	if movesUp {
		for _, row := range simulated {
			if row.VersionID >= minNewVID && row.VersionID <= maxNewVID {
				targetRows = append(targetRows, row)
			}
		}
	} else {
		if len(minRows) == 0 {
			messages = append(messages, fmt.Sprintf("  %s: no row for min_new_vid=%d; id shift would affect 0 row(s)", tableName, minNewVID))
			return simulated, messages, issues
		}
		for _, row := range simulated {
			if row.ID < minRows[0].ID && row.VersionID > maxNewVID {
				targetRows = append(targetRows, row)
			}
		}
	}
	if len(targetRows) == 0 {
		messages = append(messages, fmt.Sprintf("  %s: id shift would affect 0 row(s)", tableName))
		return simulated, messages, issues
	}
	if len(minRows) != 1 {
		issues = append(issues, fmt.Sprintf("  %s: expected one row for min_new_vid=%d, found %d", tableName, minNewVID, len(minRows)))
		return simulated, messages, issues
	}
	if len(maxRows) != 1 {
		issues = append(issues, fmt.Sprintf("  %s: expected one row for max_new_vid=%d, found %d", tableName, maxNewVID, len(maxRows)))
		return simulated, messages, issues
	}

	var idsAfterMax []int64
	for _, row := range simulated {
		if row.ID > maxRows[0].ID {
			idsAfterMax = append(idsAfterMax, row.ID)
		}
	}
	var offset int64
	if len(idsAfterMax) == 0 {
		messages = append(messages, fmt.Sprintf("  %s: generated offset would be NULL; COALESCE will shift by +0", tableName))
		offset = 0
	} else {
		offset = maxInt64(idsAfterMax) - minRows[0].ID + 1
		if offset <= 0 {
			issues = append(issues, fmt.Sprintf("  %s: generated offset would be %d, expected a positive value", tableName, offset))
			return simulated, messages, issues
		}
	}

	maxMovedDownVID := targetRows[0].VersionID
	for _, row := range targetRows {
		if row.VersionID > maxMovedDownVID {
			maxMovedDownVID = row.VersionID
		}
	}
	spaceIDs := map[int64]struct{}{}
	for _, row := range simulated {
		if row.VersionID > maxMovedDownVID {
			spaceIDs[row.ID] = struct{}{}
		}
	}
	withSpace := shiftRows(simulated, spaceIDs, offset)
	messages = append(messages, fmt.Sprintf("  %s: would make space by shifting %d row(s) after version_id=%d by +%d", tableName, len(spaceIDs), maxMovedDownVID, offset))

	targetIDs := map[int64]struct{}{}
	for _, row := range targetRows {
		targetIDs[row.ID] = struct{}{}
	}
	shifted := shiftRows(withSpace, targetIDs, offset)
	messages = append(messages, fmt.Sprintf("  %s: would shift %d row(s) by +%d", tableName, len(targetRows), offset))
	return shifted, messages, issues
}

func rowsForVID(rows []tableRow, vid int64) []tableRow {
	var out []tableRow
	for _, row := range rows {
		if row.VersionID == vid {
			out = append(out, row)
		}
	}
	return out
}

func maxInt64(values []int64) int64 {
	maxVal := values[0]
	for _, val := range values[1:] {
		if val > maxVal {
			maxVal = val
		}
	}
	return maxVal
}

func shiftRows(rows []tableRow, ids map[int64]struct{}, offset int64) []tableRow {
	shifted := make([]tableRow, 0, len(rows))
	for _, row := range rows {
		newID := row.ID
		if _, ok := ids[row.ID]; ok {
			newID += offset
		}
		shifted = append(shifted, tableRow{ID: newID, VersionID: row.VersionID, IsApplied: row.IsApplied})
	}
	return shifted
}

func validateFinalTableState(tableName string, rows []tableRow) []string {
	var issues []string
	ids := map[int64][]int64{}
	for _, row := range rows {
		ids[row.ID] = append(ids[row.ID], row.VersionID)
	}
	for id, vids := range ids {
		if len(vids) > 1 {
			issues = append(issues, fmt.Sprintf("  %s: duplicate id=%d after simulated fix (version_ids=%v)", tableName, id, vids))
		}
	}

	var applied []tableRow
	for _, row := range rows {
		if row.IsApplied && row.VersionID > 0 {
			applied = append(applied, row)
		}
	}
	sort.Slice(applied, func(i, j int) bool { return applied[i].ID < applied[j].ID })

	appliedVIDs := map[int64][]int64{}
	for _, row := range applied {
		appliedVIDs[row.VersionID] = append(appliedVIDs[row.VersionID], row.ID)
	}
	for vid, ids := range appliedVIDs {
		if len(ids) > 1 {
			issues = append(issues, fmt.Sprintf("  %s: duplicate applied version_id=%d after simulated fix (ids=%v)", tableName, vid, ids))
		}
	}
	for i := 0; i < len(applied)-1; i++ {
		if applied[i].VersionID > applied[i+1].VersionID {
			issues = append(issues, fmt.Sprintf("  %s: ordering violation after simulated fix -- id=%d (vid=%d) > id=%d (vid=%d)", tableName, applied[i].ID, applied[i].VersionID, applied[i+1].ID, applied[i+1].VersionID))
		}
	}
	return issues
}

func writerConfig(conf configpkg.MysqlConfig, opts options) (*configpkg.MysqlConfig, error) {
	if opts.dbHost != "" {
		conf.Address = fmt.Sprintf("%s:%d", opts.dbHost, opts.dbPort)
	}
	if opts.dbName != "" {
		conf.Database = opts.dbName
	}
	if opts.dbUser != "" {
		conf.Username = opts.dbUser
	}
	if opts.dbPassword != "" {
		conf.Password = opts.dbPassword
	} else if env := os.Getenv("FLEET_DB_PASSWORD"); env != "" {
		conf.Password = env
	}
	if conf.Password == "" && conf.PasswordPath == "" && conf.Region == "" {
		pass, err := promptPassword()
		if err != nil {
			return nil, err
		}
		conf.Password = pass
	}
	if opts.tlsMode == "skip-verify" {
		conf.TLSConfig = "skip-verify"
	}
	if opts.tlsMode == "verify-ca" || opts.tlsMode == "verify-identity" {
		conf.TLSCA = opts.tlsCA
		conf.TLSCert = opts.tlsCert
		conf.TLSKey = opts.tlsKey
	}
	return &conf, nil
}

func promptPassword() (string, error) {
	fmt.Fprint(os.Stderr, "MySQL password: ")
	if term.IsTerminal(int(os.Stdin.Fd())) {
		bytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		return string(bytes), err
	}
	reader := bufio.NewReader(os.Stdin)
	pass, err := reader.ReadString('\n')
	return strings.TrimSpace(pass), err
}

func openWriterDB(conf *configpkg.MysqlConfig) (*sqlx.DB, error) {
	if conf.PasswordPath != "" && conf.Password != "" {
		return nil, errors.New("a MySQL password and password file were provided; specify only one")
	}
	if conf.PasswordPath != "" {
		contents, err := os.ReadFile(conf.PasswordPath)
		if err != nil {
			return nil, err
		}
		conf.Password = strings.TrimSpace(string(contents))
	}
	if conf.TLSCA != "" {
		tlsConfigName := fmt.Sprintf("migration-cleanup-%d", time.Now().UnixNano())
		tlsOpts := configpkg.TLS{
			TLSCert:       conf.TLSCert,
			TLSKey:        conf.TLSKey,
			TLSCA:         conf.TLSCA,
			TLSServerName: conf.TLSServerName,
		}
		tlsConfig, err := tlsOpts.ToTLSConfig()
		if err != nil {
			return nil, err
		}
		if err := gomysql.RegisterTLSConfig(tlsConfigName, tlsConfig); err != nil {
			return nil, err
		}
		conf.TLSConfig = tlsConfigName
	}

	commonConf := &commonmysql.MysqlConfig{
		Protocol:        conf.Protocol,
		Address:         conf.Address,
		Username:        conf.Username,
		Password:        conf.Password,
		PasswordPath:    conf.PasswordPath,
		Database:        conf.Database,
		TLSCert:         conf.TLSCert,
		TLSKey:          conf.TLSKey,
		TLSCA:           conf.TLSCA,
		TLSServerName:   conf.TLSServerName,
		TLSConfig:       conf.TLSConfig,
		MaxOpenConns:    conf.MaxOpenConns,
		MaxIdleConns:    conf.MaxIdleConns,
		ConnMaxLifetime: conf.ConnMaxLifetime,
		SQLMode:         conf.SQLMode,
		Region:          conf.Region,
	}
	return commonmysql.NewDB(commonConf, &commonmysql.DBOptions{
		MaxAttempts: 1,
		Logger:      slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}, "mysql")
}

func applyStatements(ctx context.Context, db *sqlx.DB, groups map[string][]string) error {
	var statements []string
	for _, tableName := range []string{tableStatusName, dataStatusName} {
		statements = append(statements, groups[tableName]...)
	}
	return commonmysql.WithTxx(ctx, db, func(tx sqlx.ExtContext) error {
		for _, stmt := range statements {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("execute %q: %w", stmt, err)
			}
		}
		return nil
	}, slog.New(slog.NewTextHandler(os.Stderr, nil)))
}

func writeOutput(sqlText, outputPath string) error {
	if outputPath == "" {
		fmt.Print(sqlText)
		return nil
	}
	if err := os.WriteFile(outputPath, []byte(sqlText), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "SQL written to %s\n", outputPath)
	return nil
}
