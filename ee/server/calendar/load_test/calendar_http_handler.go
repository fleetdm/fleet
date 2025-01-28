// Package calendartest is not imported in production code, so it will not be compiled for Fleet server.
package calendartest

import (
	"context"
	"crypto/md5" //nolint:gosec // (only used in testing)
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/api/calendar/v3"
)

// This calendar does not support all-day events.

var db *sql.DB
var timezones = []string{
	"America/Chicago",
	"America/New_York",
	"America/Los_Angeles",
	"America/Anchorage",
	"Pacific/Honolulu",
	"America/Argentina/Buenos_Aires",
	"Asia/Kolkata",
	"Europe/London",
	"Europe/Paris",
	"Australia/Sydney",
}

func Configure(dbPath string) (http.Handler, error) {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	logger.Println("Server is starting...")

	// Initialize the database schema if needed
	err = initializeSchema()
	if err != nil {
		return nil, err
	}

	router := http.NewServeMux()
	router.HandleFunc("/settings", getSetting)
	router.HandleFunc("/events", getEvent)
	router.HandleFunc("/events/list", getEvents)
	router.HandleFunc("/events/add", addEvent)
	router.HandleFunc("/events/delete", deleteEvent)
	return logging(logger)(router), nil
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				defer func() {
					logger.Println(r.Method, r.URL.String(), r.RemoteAddr)
				}()
				next.ServeHTTP(w, r)
			},
		)
	}
}

func Close() {
	_ = db.Close()
}

func getSetting(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}
	if name != "timezone" {
		http.Error(w, "unsupported setting", http.StatusNotFound)
		return
	}
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "missing email", http.StatusBadRequest)
		return
	}
	timezone := getTimezone(email)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	setting := calendar.Setting{Value: timezone}
	err := json.NewEncoder(w).Encode(setting)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// The timezone is determined by the user's email address
func getTimezone(email string) string {
	index := hash(email) % uint32(len(timezones)) //nolint:gosec // dismiss G115 (only used for tests)
	timezone := timezones[index]
	return timezone
}

func hash(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

// getEvent handles GET /events?id=123
func getEvent(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	sqlStmt := "SELECT email, start, end, summary, description, status FROM events WHERE id = ?"
	var start, end int64
	var email, summary, description, status string
	err := db.QueryRow(sqlStmt, id).Scan(&email, &start, &end, &summary, &description, &status)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	timezone := getTimezone(email)
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	calEvent := calendar.Event{}
	calEvent.Id = id
	calEvent.Start = &calendar.EventDateTime{DateTime: time.Unix(start, 0).In(loc).Format(time.RFC3339)}
	calEvent.End = &calendar.EventDateTime{DateTime: time.Unix(end, 0).In(loc).Format(time.RFC3339)}
	calEvent.Summary = summary
	calEvent.Description = description
	calEvent.Status = status
	calEvent.Etag = computeETag(start, end, summary, description, status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(calEvent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getEvents(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "missing email", http.StatusBadRequest)
		return
	}
	timeMin := r.URL.Query().Get("timemin")
	if email == "" {
		http.Error(w, "missing timemin", http.StatusBadRequest)
		return
	}
	timeMax := r.URL.Query().Get("timemax")
	if email == "" {
		http.Error(w, "missing timemax", http.StatusBadRequest)
		return
	}
	minTime, err := parseDateTime(r.Context(), &calendar.EventDateTime{DateTime: timeMin})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	maxTime, err := parseDateTime(r.Context(), &calendar.EventDateTime{DateTime: timeMax})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sqlStmt := "SELECT id, start, end, summary, description, status FROM events WHERE email = ? AND end > ? AND start < ?"
	rows, err := db.Query(sqlStmt, email, minTime.Unix(), maxTime.Unix())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	timezone := getTimezone(email)
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	events := calendar.Events{}
	events.Items = make([]*calendar.Event, 0)
	for rows.Next() {
		var id, start, end int64
		var summary, description, status string
		err = rows.Scan(&id, &start, &end, &summary, &description, &status)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		calEvent := calendar.Event{}
		calEvent.Id = fmt.Sprintf("%d", id)
		calEvent.Start = &calendar.EventDateTime{DateTime: time.Unix(start, 0).In(loc).Format(time.RFC3339)}
		calEvent.End = &calendar.EventDateTime{DateTime: time.Unix(end, 0).In(loc).Format(time.RFC3339)}
		calEvent.Summary = summary
		calEvent.Description = description
		calEvent.Status = status
		calEvent.Etag = computeETag(start, end, summary, description, status)
		events.Items = append(events.Items, &calEvent)
	}
	if err = rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(events)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// addEvent handles POST /events/add?email=user@example.com
func addEvent(w http.ResponseWriter, r *http.Request) {
	var event calendar.Event
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(body, &event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "missing email", http.StatusBadRequest)
		return
	}
	start, err := parseDateTime(r.Context(), event.Start)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	end, err := parseDateTime(r.Context(), event.End)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status := "confirmed"
	sqlStmt := `INSERT INTO events (email, start, end, summary, description, status) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := db.Exec(sqlStmt, email, start.Unix(), end.Unix(), event.Summary, event.Description, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	event.Id = fmt.Sprintf("%d", id)
	event.Etag = computeETag(start.Unix(), end.Unix(), event.Summary, event.Description, status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func computeETag(args ...any) string {
	h := md5.New() //nolint:gosec // (only used for tests)
	_, _ = fmt.Fprint(h, args...)
	checksum := h.Sum(nil)
	return hex.EncodeToString(checksum)
}

// deleteEvent handles DELETE /events/delete?id=123
func deleteEvent(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	sqlStmt := "DELETE FROM events WHERE id = ?"
	_, err := db.Exec(sqlStmt, id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "not found", http.StatusGone)
		return
	}
}

func initializeSchema() error {
	createTableSQL := `CREATE TABLE IF NOT EXISTS events (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"email" TEXT NOT NULL,
		"start" INTEGER NOT NULL,
		"end" INTEGER NOT NULL,
		"summary" TEXT NOT NULL,
		"description" TEXT NOT NULL,
		"status" TEXT NOT NULL
	);`
	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

func parseDateTime(ctx context.Context, eventDateTime *calendar.EventDateTime) (*time.Time, error) {
	var t time.Time
	var err error
	if eventDateTime.TimeZone != "" {
		var loc *time.Location
		loc, err = time.LoadLocation(eventDateTime.TimeZone)
		if err == nil {
			t, err = time.ParseInLocation(time.RFC3339, eventDateTime.DateTime, loc)
		}
	} else {
		t, err = time.Parse(time.RFC3339, eventDateTime.DateTime)
	}
	if err != nil {
		return nil, ctxerr.Wrap(
			ctx, err, fmt.Sprintf("parsing calendar event time: %s", eventDateTime.DateTime),
		)
	}
	return &t, nil
}
