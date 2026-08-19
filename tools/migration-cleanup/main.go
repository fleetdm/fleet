package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
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

type options struct {
	checkout    string
	branch      string
	sinceCommit string
	commit      string
	output      string
	dryRun      bool
	apply       bool
	verbose     bool

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

// newRootCmd creates and configures the root cobra command with all flags.
func newRootCmd(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "migration-cleanup",
		Short:        "Detect and fix Fleet migration renumbering",
		SilenceUsage: true,
	}

	cmd.PersistentFlags().String("config", "", "Path to a Fleet configuration file")
	cmd.Flags().StringVarP(&opts.checkout, "checkout", "c", ".", "Path to fleetdm/fleet git checkout")
	cmd.Flags().StringVarP(&opts.branch, "branch", "b", "", "Branch name")
	cmd.Flags().StringVarP(&opts.sinceCommit, "since-commit", "", "", "Scan main from this commit forward (hash or short ref)")
	cmd.Flags().StringVarP(&opts.commit, "commit", "", "", "Inspect a single commit for renames")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Write SQL to file instead of stdout")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Connect to MySQL, simulate the SQL, and verify the final state")
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

// hideUnneededFleetConfigFlags hides Fleet config flags that are not relevant
// to this tool, keeping only the MySQL-related flags visible.
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

// run executes the migration cleanup workflow: validates options, scans git
// history for migration renames, generates SQL, and optionally dry-runs or applies.
func run(ctx context.Context, configManager configpkg.Manager, opts options) error {
	if countModeFlags(opts) != 1 {
		return exitError{code: exitGeneral, message: "ERROR: exactly one of --branch, --since-commit, or --commit is required"}
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

	var renames []migrationRename
	switch {
	case opts.branch != "":
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

	case opts.sinceCommit != "":
		resolvedSHA, err := resolveRef(checkout, opts.sinceCommit)
		if err != nil {
			return exitError{code: exitGeneral, message: err.Error()}
		}
		if opts.verbose {
			fmt.Fprintf(os.Stderr, "Resolved since-commit: %s\n", resolvedSHA)
		}

		mainRef, err := resolveMainRef(checkout)
		if err != nil {
			return exitError{code: exitGeneral, message: err.Error()}
		}

		if !isAncestor(checkout, resolvedSHA, mainRef) {
			return exitError{code: exitGeneral, message: fmt.Sprintf("ERROR: %q is not an ancestor of %s", opts.sinceCommit, mainRef)}
		}

		// Include resolvedSHA itself in the scan, then scan descendants.
		rs, err := extractRenames(checkout, resolvedSHA)
		if err != nil {
			return exitError{code: exitGeneral, message: err.Error()}
		}
		if opts.verbose {
			fmt.Fprintf(os.Stderr, "  %s: %d rename(s)\n", shortSHA(resolvedSHA), len(rs))
		}
		renames = append(renames, rs...)

		commits, err := findRenameCommits(checkout, mainRef, resolvedSHA)
		if err != nil {
			return exitError{code: exitGeneral, message: err.Error()}
		}
		if opts.verbose {
			fmt.Fprintf(os.Stderr, "Rename commits found: %d\n", len(commits))
		}

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

	case opts.commit != "":
		resolvedSHA, err := resolveRef(checkout, opts.commit)
		if err != nil {
			return exitError{code: exitGeneral, message: err.Error()}
		}
		if opts.verbose {
			fmt.Fprintf(os.Stderr, "Resolved commit: %s\n", resolvedSHA)
		}

		rs, err := extractRenames(checkout, resolvedSHA)
		if err != nil {
			return exitError{code: exitGeneral, message: err.Error()}
		}
		renames = rs
	}

	if len(renames) == 0 {
		fmt.Println("No migration renumbering detected in the selected scope.")
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

// validateTLSFlags checks that --tls-mode is a valid value.
func validateTLSFlags(opts options) error {
	switch opts.tlsMode {
	case "", "skip-verify", "verify-ca", "verify-identity":
	default:
		return errors.New("--tls-mode must be one of skip-verify, verify-ca, verify-identity")
	}
	return nil
}

// validateEffectiveTLSConfig validates the combined TLS configuration
// after merging CLI flags and Fleet config.
func validateEffectiveTLSConfig(conf configpkg.MysqlConfig, tlsMode string) error {
	if tlsMode == "verify-ca" || tlsMode == "verify-identity" {
		if conf.TLSCA == "" {
			return errors.New("--tls-ca or mysql_tls_ca is required for verify-ca / verify-identity")
		}
	}
	if conf.TLSConfig != "skip-verify" && (conf.TLSCert == "") != (conf.TLSKey == "") {
		return errors.New("TLS client certificate and key must be provided together")
	}
	return nil
}

// git runs a git command in the given checkout directory and returns stdout.
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

// countModeFlags returns the number of scan-mode flags set in opts.
// Exactly one must be set; zero or more than one is an error.
func countModeFlags(opts options) int {
	n := 0
	if opts.branch != "" {
		n++
	}
	if opts.sinceCommit != "" {
		n++
	}
	if opts.commit != "" {
		n++
	}
	return n
}

// resolveRef resolves a git ref to a full commit SHA, peeling annotated tags.
func resolveRef(checkout, ref string) (string, error) {
	out, err := git(checkout, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("cannot resolve ref %q: %w", ref, err)
	}
	return strings.TrimSpace(out), nil
}

// resolveMainRef returns the best available ref for the main branch,
// preferring origin/main (freshly fetched) over local main.
func resolveMainRef(checkout string) (string, error) {
	for _, candidate := range []string{"origin/main", "main"} {
		_, err := git(checkout, "rev-parse", "--verify", candidate)
		if err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("cannot resolve main ref (neither origin/main nor main exists)")
}

// isAncestor reports whether ancestor is an ancestor of descendant.
func isAncestor(checkout, ancestor, descendant string) bool {
	_, err := git(checkout, "merge-base", "--is-ancestor", ancestor, descendant)
	return err == nil
}

// resolveBranch resolves a branch name, trying local then origin/<branch>.
func resolveBranch(checkout, branch string) (string, error) {
	candidates := []string{branch, "origin/" + branch}
	var lastErr error
	for _, candidate := range candidates {
		_, err := git(checkout, "rev-parse", "--verify", candidate)
		if err == nil {
			return candidate, nil
		}

		lastErr = err
		fmt.Fprintln(os.Stderr, err)
	}
	if lastErr != nil {
		return "", fmt.Errorf("ERROR: cannot resolve branch %q", branch)
	}
	return "", fmt.Errorf("ERROR: cannot resolve branch %q", branch)
}

// getMergeBase returns the merge base between main and the given branch.
func getMergeBase(checkout, branch string) (string, error) {
	out, err := git(checkout, "merge-base", "main", branch)
	return strings.TrimSpace(out), err
}

// findRenameCommits returns commit SHAs in mergeBase..branch that contain
// renames in the migration directories, oldest first. Commit order matters:
// applying chained renames (A -> B in one commit, B -> C in a later one)
// oldest-first moves rows through the chain to the terminal version ID.
func findRenameCommits(checkout, branch, mergeBase string) ([]string, error) {
	args := []string{"log", "--reverse", "-M", "--diff-filter=R", "--format=%H", mergeBase + ".." + branch, "--"}
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

// extractRenames parses a single commit for migration file renames and returns
// the detected version ID changes.
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

// shortSHA returns the first 12 characters of a SHA, or the full string if shorter.
func shortSHA(sha string) string {
	if len(sha) < 12 {
		return sha
	}
	return sha[:12]
}

// dedupeRenames removes duplicate renames based on migration type and version IDs.
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

// generateStatementGroups groups renames by migration type and generates
// SQL statements for each table.
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
		groups[item.table] = buildSQL(item.table, tableRenames)
	}
	return groups
}

// buildSQL generates the full SQL for a table: version remaps (in commit
// order, so chained renames resolve to the terminal version ID), duplicate
// cleanup, and a full rebuild of row ids into version order. Rebuilding the
// total order instead of computing a minimal shift handles every renumber
// shape identically: moves up, moves down, mixed batches in one scope, and
// remapped rows stranded at the table tail.
func buildSQL(tableName string, renames []migrationRename) []string {
	lines := make([]string, 0)
	for _, r := range renames {
		lines = append(lines, fmt.Sprintf("UPDATE `%s` SET version_id = %d WHERE version_id = %d;", tableName, r.newVersionID, r.oldVersionID))
	}
	if len(renames) == 0 {
		return lines
	}

	lines = append(lines,
		fmt.Sprintf("CREATE TEMPORARY TABLE `_fix_dups_%s` (id BIGINT UNSIGNED);", tableName),
		fmt.Sprintf("INSERT INTO `_fix_dups_%s` (id) SELECT id FROM `%s` WHERE (version_id, id) NOT IN (SELECT version_id, MIN(id) FROM `%s` GROUP BY version_id);", tableName, tableName, tableName),
		fmt.Sprintf("DELETE FROM `%s` WHERE id IN (SELECT id FROM `_fix_dups_%s`);", tableName, tableName),
		fmt.Sprintf("DROP TEMPORARY TABLE `_fix_dups_%s`;", tableName),
	)

	// Rebuild ids in two passes: rebase every row above the current MAX(id)
	// (targets are all beyond existing ids, so no transient duplicate keys
	// regardless of update order), then compact down to 1..N. Compacting keeps
	// the final ids below the table's AUTO_INCREMENT counter, which UPDATEs do
	// not advance, so future goose inserts cannot collide with shifted rows.
	lines = append(lines,
		fmt.Sprintf("SELECT MAX(id) INTO @rebase_%s FROM `%s`;", tableName, tableName),
		fmt.Sprintf("CREATE TEMPORARY TABLE `_fix_order_%s` (id BIGINT UNSIGNED PRIMARY KEY, rn BIGINT UNSIGNED);", tableName),
		fmt.Sprintf("INSERT INTO `_fix_order_%s` (id, rn) SELECT id, ROW_NUMBER() OVER (ORDER BY version_id ASC, id ASC) FROM `%s`;", tableName, tableName),
		fmt.Sprintf("UPDATE `%s` t JOIN `_fix_order_%s` o ON t.id = o.id SET t.id = @rebase_%s + o.rn;", tableName, tableName, tableName),
		fmt.Sprintf("UPDATE `%s` t JOIN `_fix_order_%s` o ON t.id = @rebase_%s + o.rn SET t.id = o.rn;", tableName, tableName, tableName),
		fmt.Sprintf("DROP TEMPORARY TABLE `_fix_order_%s`;", tableName),
	)
	return lines
}

// renderSQL formats statement groups into a complete SQL transaction.
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

// queryTableRows fetches all rows from a migration status table.
func queryTableRows(ctx context.Context, db *sqlx.DB, tableName string) ([]tableRow, error) {
	var rows []tableRow
	if err := sqlx.SelectContext(ctx, db, &rows, fmt.Sprintf("SELECT id, version_id, is_applied FROM `%s` ORDER BY id", tableName)); err != nil {
		return nil, fmt.Errorf("query %s: %w", tableName, err)
	}
	return rows, nil
}

// verifyDryRun simulates the generated SQL against real table data and reports
// whether the final state would be valid.
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

// simulateTableSQL applies version remaps (sequentially, in the same order as
// the generated SQL so chained renames resolve identically), duplicate
// removal, and the version-order id rebuild to a copy of the table rows,
// returning the simulated result plus messages and issues.
func simulateTableSQL(tableName string, rows []tableRow, renames []migrationRename) ([]tableRow, []string, []string) {
	var messages []string
	var issues []string

	simulated := make([]tableRow, len(rows))
	copy(simulated, rows)

	for _, r := range renames {
		found := false
		for i := range simulated {
			if simulated[i].VersionID == r.oldVersionID {
				simulated[i].VersionID = r.newVersionID
				found = true
			}
		}
		if found {
			messages = append(messages, fmt.Sprintf("  %s: will remap %d -> %d", tableName, r.oldVersionID, r.newVersionID))
		} else {
			messages = append(messages, fmt.Sprintf("  %s: %d not present (UPDATE will be no-op)", tableName, r.oldVersionID))
		}
	}

	byVID := map[int64][]tableRow{}
	for _, row := range simulated {
		byVID[row.VersionID] = append(byVID[row.VersionID], row)
	}
	simulated = simulated[:0]
	for _, vid := range slices.Sorted(maps.Keys(byVID)) {
		vidRows := byVID[vid]
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

	violations := countOrderingViolations(simulated)
	sort.Slice(simulated, func(i, j int) bool {
		if simulated[i].VersionID != simulated[j].VersionID {
			return simulated[i].VersionID < simulated[j].VersionID
		}
		return simulated[i].ID < simulated[j].ID
	})
	renumbered := 0
	for i := range simulated {
		if simulated[i].ID != int64(i+1) {
			renumbered++
			simulated[i].ID = int64(i + 1)
		}
	}
	messages = append(messages, fmt.Sprintf("  %s: would renumber %d row id(s) into version order, fixing %d ordering violation(s)", tableName, renumbered, violations))
	return simulated, messages, issues
}

// countOrderingViolations returns the number of adjacent applied-row pairs
// (ordered by id) whose version_ids are out of order.
func countOrderingViolations(rows []tableRow) int {
	var applied []tableRow
	for _, row := range rows {
		if row.IsApplied && row.VersionID > 0 {
			applied = append(applied, row)
		}
	}
	sort.Slice(applied, func(i, j int) bool { return applied[i].ID < applied[j].ID })
	violations := 0
	for i := 0; i < len(applied)-1; i++ {
		if applied[i].VersionID > applied[i+1].VersionID {
			violations++
		}
	}
	return violations
}

// validateFinalTableState checks the simulated table for duplicate IDs,
// duplicate applied version IDs, and ordering violations.
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

// writerConfig merges CLI database options with Fleet config to produce
// the effective MySQL configuration.
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
	if opts.tlsCA != "" {
		conf.TLSCA = opts.tlsCA
	}
	if opts.tlsCert != "" {
		conf.TLSCert = opts.tlsCert
	}
	if opts.tlsKey != "" {
		conf.TLSKey = opts.tlsKey
	}
	if err := validateEffectiveTLSConfig(conf, opts.tlsMode); err != nil {
		return nil, err
	}
	return &conf, nil
}

// promptPassword reads a password from stdin, using secure terminal read if available.
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

// openWriterDB opens a connection to the MySQL database using the given config.
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
	if conf.TLSCA != "" && conf.TLSConfig != "skip-verify" {
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

// applyStatements executes all SQL statements in a transaction.
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

// writeOutput writes SQL to stdout or to a file if outputPath is set.
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
