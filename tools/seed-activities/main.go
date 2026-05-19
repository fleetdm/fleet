// seed-activities inserts one of every activity type into a local Fleet
// database so the activity feeds and host activity cards can be tested in
// the UI without driving each flow manually.
//
// Usage (from the repo root):
//
//	go run ./tools/seed-activities
//	go run ./tools/seed-activities -host-id 3 -actor admin@example.com
//
// The tool writes directly to `activity_past` (and `activity_host_past`
// for host-scoped activities). It does not call the service layer, so no
// webhooks fire and no other side effects occur — making it safe to run
// repeatedly against a dev DB.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"reflect"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// seedHostID is the host that host-scoped activities will be linked to.
// Overridable via -host-id; reflection sets struct fields named "HostID"
// (or json tagged "host_id") to this value.
var seedHostID uint = 3

// stringExample returns a deterministic example value based on the field's
// JSON tag (preferred) or Go field name. The aim is to produce something
// the UI templates can render meaningfully (e.g. "GitHub Desktop" for
// software_title) rather than the generic "example".
func stringExample(jsonTag, fieldName string) string {
	name := strings.ToLower(jsonTag)
	if name == "" {
		name = strings.ToLower(fieldName)
	}
	switch {
	case strings.Contains(name, "host_display") || name == "hostname":
		return "example-host"
	case name == "software_display_name" || strings.HasPrefix(name, "software_title"):
		return "GitHub Desktop"
	case strings.Contains(name, "software_package"):
		return "GitHubDesktop-arm64.dmg"
	case strings.Contains(name, "software_icon_url"):
		return "https://example.com/icon.png"
	case strings.Contains(name, "app_store_id"):
		return "497799835"
	case strings.Contains(name, "team_name") || strings.Contains(name, "fleet_name"):
		return "Marketing"
	case strings.Contains(name, "user_full") || strings.Contains(name, "actor_full"):
		return "Example User"
	case strings.Contains(name, "user_email") || name == "email":
		return "user@example.com"
	case strings.Contains(name, "user_name"):
		return "user@example.com"
	case strings.Contains(name, "policy_name"):
		return "Failing policy"
	case strings.Contains(name, "policy_critical"):
		return "false"
	case strings.Contains(name, "profile_name"):
		return "Example profile"
	case strings.Contains(name, "label_name"):
		return "Example label"
	case strings.Contains(name, "script_name"):
		return "example.sh"
	case strings.Contains(name, "script_execution_id"):
		return "00000000-0000-0000-0000-000000000001"
	case strings.Contains(name, "command_uuid"):
		return "00000000-0000-0000-0000-000000000001"
	case strings.Contains(name, "install_uuid"):
		return "00000000-0000-0000-0000-000000000002"
	case strings.Contains(name, "uuid"):
		return "00000000-0000-0000-0000-000000000003"
	case strings.Contains(name, "url"):
		return "https://example.com/"
	case strings.Contains(name, "platform"):
		return "darwin"
	case strings.Contains(name, "status"):
		return "installed"
	case strings.Contains(name, "role"):
		return "admin"
	case strings.Contains(name, "location"):
		return "United States"
	case strings.Contains(name, "mode"):
		return "all"
	case strings.Contains(name, "name"):
		return "Example name"
	default:
		return "example"
	}
}

// setExampleFields walks the activity's struct fields and assigns
// deterministic example values to anything left at its zero value, so the
// JSON the activity templates consume looks plausible. Hosts get the seed
// host id wired in.
func setExampleFields(activity any) {
	v := reflect.ValueOf(activity)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return
	}
	s := v.Elem()
	t := s.Type()
	for i := 0; i < t.NumField(); i++ {
		f := s.Field(i)
		if !f.CanSet() {
			continue
		}
		ft := t.Field(i)
		jsonTag := strings.Split(ft.Tag.Get("json"), ",")[0]
		nameLower := strings.ToLower(jsonTag)
		if nameLower == "" {
			nameLower = strings.ToLower(ft.Name)
		}

		switch f.Kind() {
		case reflect.String:
			if f.String() == "" {
				f.SetString(stringExample(jsonTag, ft.Name))
			}
		case reflect.Bool:
			// Leave bools at false by default. Self-service software activities
			// are flipped to true so the new passive-voice rendering can be
			// inspected.
			if nameLower == "self_service" {
				f.SetBool(true)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if f.Uint() == 0 {
				if strings.Contains(nameLower, "host_id") {
					f.SetUint(uint64(seedHostID))
				} else {
					f.SetUint(1)
				}
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if f.Int() == 0 {
				f.SetInt(1)
			}
		case reflect.Ptr:
			if !f.IsNil() {
				continue
			}
			elem := f.Type().Elem()
			switch elem.Kind() {
			case reflect.String:
				val := stringExample(jsonTag, ft.Name)
				f.Set(reflect.ValueOf(&val))
			case reflect.Bool:
				val := false
				f.Set(reflect.ValueOf(&val))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				val := uint(1)
				if strings.Contains(nameLower, "host_id") {
					val = seedHostID
				}
				ptr := reflect.New(elem)
				ptr.Elem().SetUint(uint64(val))
				f.Set(ptr)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				ptr := reflect.New(elem)
				ptr.Elem().SetInt(1)
				f.Set(ptr)
			}
		case reflect.Slice:
			if !f.IsNil() {
				continue
			}
			// Host ID slices get the seed host. Other slices are left nil so
			// they marshal away and don't introduce noise in the UI.
			if ft.Name == "HostIDs" || nameLower == "host_ids" {
				f.Set(reflect.ValueOf([]uint{seedHostID}))
			}
		}
	}
}

// hostIDer matches fleet activities that report associated hosts. We use
// the runtime interface rather than the fleet-internal types.ActivityHosts
// to avoid pulling that bounded context in.
type hostIDer interface {
	HostIDs() []uint
}

// Statuses we cycle through for self-service install/uninstall activities
// when the -self-service-only mode runs, so each new passive-voice
// predicate variant gets exercised.
var selfServiceStatuses = []string{
	"installed",
	"pending_install",
	"failed_install",
	"uninstalled",
	"pending_uninstall",
	"failed_uninstall",
}

// selfServiceActivityNames is the set of activity types triggered by the
// My device API. -self-service-only restricts seeding to these and emits
// one row per status above.
var selfServiceActivityNames = map[string]struct{}{
	"installed_software":      {},
	"uninstalled_software":    {},
	"installed_app_store_app": {},
}

// insertActivity writes one row to activity_past (plus activity_host_past
// for host-scoped activities). When endUserOnly is true, user_id is left
// NULL and user_email is empty — matching the row shape NewActivity
// produces when the My device API path passes a nil user.
func insertActivity(
	ctx context.Context, db *sql.DB,
	actorID uint, actorName, actorEmail string,
	endUserOnly bool,
	activity fleet.ActivityDetails,
) (int64, error) {
	details, err := json.Marshal(activity)
	if err != nil {
		return 0, fmt.Errorf("marshal %T: %w", activity, err)
	}

	var res sql.Result
	if endUserOnly {
		const insert = `INSERT INTO activity_past
			(user_id, user_name, user_email, activity_type, details, fleet_initiated)
			VALUES (NULL, NULL, '', ?, ?, 0)`
		res, err = db.ExecContext(ctx, insert, activity.ActivityName(), details)
	} else {
		const insert = `INSERT INTO activity_past
			(user_id, user_name, user_email, activity_type, details, fleet_initiated)
			VALUES (?, ?, ?, ?, ?, 0)`
		res, err = db.ExecContext(ctx, insert,
			actorID, actorName, actorEmail, activity.ActivityName(), details)
	}
	if err != nil {
		return 0, fmt.Errorf("insert %s: %w", activity.ActivityName(), err)
	}
	actID, _ := res.LastInsertId()

	if h, ok := activity.(hostIDer); ok {
		ids := h.HostIDs()
		if len(ids) > 0 {
			const insertHost = `INSERT INTO activity_host_past (host_id, activity_id) VALUES (?, ?)`
			for _, hid := range ids {
				if _, err := db.ExecContext(ctx, insertHost, hid, actID); err != nil {
					return actID, fmt.Errorf("insert activity_host_past %d/%d: %w", hid, actID, err)
				}
			}
		}
	}
	return actID, nil
}

// fillSelfServiceVariant clones an install/uninstall activity, sets its
// status field to the given value, and returns it filled with example
// data. self_service is forced on.
func fillSelfServiceVariant(template fleet.ActivityDetails, status string) fleet.ActivityDetails {
	ptr := reflect.New(reflect.TypeOf(template))
	ptr.Elem().Set(reflect.ValueOf(template))
	setExampleFields(ptr.Interface())

	// Override Status and SelfService explicitly — reflection's default
	// pass uses "installed" for any status field, and we want the full
	// matrix here.
	s := ptr.Elem()
	if statusField := s.FieldByName("Status"); statusField.IsValid() && statusField.CanSet() && statusField.Kind() == reflect.String {
		statusField.SetString(status)
	}
	if ss := s.FieldByName("SelfService"); ss.IsValid() && ss.CanSet() && ss.Kind() == reflect.Bool {
		ss.SetBool(true)
	}
	return ptr.Elem().Interface().(fleet.ActivityDetails)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dsn := flag.String("dsn",
		"root:toor@tcp(127.0.0.1:3306)/fleet?parseTime=true&loc=UTC",
		"MySQL DSN")
	actorEmail := flag.String("actor", "admin@example.com",
		"user_email recorded on the seeded activities (ignored for self-service rows)")
	actorName := flag.String("actor-name", "Test Admin",
		"user_name recorded on the seeded activities (ignored for self-service rows)")
	actorID := flag.Uint("actor-id", 1,
		"user_id recorded on the seeded activities (must exist in users; ignored for self-service rows)")
	hostID := flag.Uint("host-id", 3,
		"host_id used for host-scoped activities and activity_host_past links")
	selfServiceOnly := flag.Bool("self-service-only", false,
		"only seed self-service software install/uninstall activities, "+
			"one row per status variant, mimicking what the My device API writes "+
			"(NULL user_id, empty user_email)")
	flag.Parse()

	seedHostID = *hostID

	db, err := sql.Open("mysql", *dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	if *selfServiceOnly {
		count := 0
		for _, a := range seedActivities {
			if _, ok := selfServiceActivityNames[a.ActivityName()]; !ok {
				continue
			}
			for _, status := range selfServiceStatuses {
				filled := fillSelfServiceVariant(a, status)
				actID, err := insertActivity(ctx, db, 0, "", "", true, filled)
				if err != nil {
					return err
				}
				count++
				fmt.Printf("[%3d] %-25s status=%-18s id=%d\n",
					count, filled.ActivityName(), status, actID)
			}
		}
		fmt.Printf("\nSeeded %d self-service activities (NULL user_id, empty user_email) on host %d.\n",
			count, seedHostID)
		return nil
	}

	for i, a := range seedActivities {
		ptr := reflect.New(reflect.TypeOf(a))
		ptr.Elem().Set(reflect.ValueOf(a))
		setExampleFields(ptr.Interface())
		filled := ptr.Elem().Interface().(fleet.ActivityDetails)

		// If reflection set self_service=true, mimic the My device API and
		// drop the actor fields too, so seeded rows match production shape.
		endUserOnly := false
		if v := reflect.ValueOf(filled).FieldByName("SelfService"); v.IsValid() && v.Kind() == reflect.Bool && v.Bool() {
			endUserOnly = true
		}

		actID, err := insertActivity(ctx, db,
			*actorID, *actorName, *actorEmail, endUserOnly, filled)
		if err != nil {
			return fmt.Errorf("[%d] %w", i+1, err)
		}

		fmt.Printf("[%3d/%d] %-50s id=%d\n",
			i+1, len(seedActivities), filled.ActivityName(), actID)
	}

	fmt.Printf("\nSeeded %d activities. Refresh the dashboard activity feed or host %d's activity card.\n",
		len(seedActivities), seedHostID)
	return nil
}
