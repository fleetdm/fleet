// Command gw-directory-fake is a standalone fake of the Google Admin SDK
// Directory API, for load/QA testing Fleet's Google Workspace IdP integration
// without a real Google Workspace tenant.
//
// It is NOT production code. Point a Fleet server at it for testing by setting
// FLEET_TEST_GOOGLE_WORKSPACE_ENDPOINT to this server's base URL and giving the
// integration a service-account JSON whose "token_uri" points at this server's
// /token endpoint.
//
// Two subcommands:
//
//	# Generate an editable JSON fixture with synthetic users and groups.
//	gw-directory-fake generate -users 1000 -groups 50 -members-per-group 20 \
//	    -domain qa.example.com -out fixture.json
//
//	# Serve the Admin SDK API from a fixture, hot-reloading it when the file changes.
//	gw-directory-fake serve -fixture fixture.json -addr :8091
//
// While serving, edit fixture.json (add/remove/rename users or groups) and the
// server picks up the change automatically (it polls the file's modtime), so the
// next sync sees the new directory state.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	directory "google.golang.org/api/admin/directory/v1"
)

// fixture is the editable on-disk representation of a Google Workspace directory.
// It is intentionally simpler than the Admin SDK schema so QA can hand-edit it.
type fixture struct {
	Domain string         `json:"domain"`
	Users  []fixtureUser  `json:"users"`
	Groups []fixtureGroup `json:"groups"`
}

type fixtureUser struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Department string `json:"department"`
	Suspended  bool   `json:"suspended"`
	Archived   bool   `json:"archived"`
}

type fixtureGroup struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	MemberIDs []string `json:"member_ids"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("gw-directory-fake: ")

	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "generate":
		runGenerate(os.Args[2:])
	case "serve":
		runServe(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `usage:
  gw-directory-fake generate -users N -groups M -members-per-group K -domain D -out FILE
  gw-directory-fake serve -fixture FILE -addr :8091 [-latency 0s] [-error-rate 0.0]
`)
	os.Exit(2)
}

// ---------------------------------------------------------------------------
// generate
// ---------------------------------------------------------------------------

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	users := fs.Int("users", 100, "number of users to generate")
	groups := fs.Int("groups", 10, "number of groups to generate")
	membersPerGroup := fs.Int("members-per-group", 25, "members assigned to each group")
	domain := fs.String("domain", "qa.example.com", "primary domain")
	out := fs.String("out", "", "output file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	fx := buildFixture(*users, *groups, *membersPerGroup, *domain)
	data, err := json.MarshalIndent(fx, "", "  ")
	if err != nil {
		log.Fatalf("marshal fixture: %v", err)
	}
	data = append(data, '\n')

	if *out == "" {
		_, _ = os.Stdout.Write(data)
		return
	}
	if err := os.WriteFile(*out, data, 0o600); err != nil {
		log.Fatalf("write %s: %v", *out, err)
	}
	log.Printf("wrote %s (%d users, %d groups, %d members/group)", *out, *users, *groups, *membersPerGroup)
}

func buildFixture(numUsers, numGroups, membersPerGroup int, domain string) fixture {
	depts := []string{"Engineering", "Sales", "Marketing", "Support", "Finance", "People", "IT", "Security"}

	users := make([]fixtureUser, 0, numUsers)
	for i := range numUsers {
		n := i + 1
		users = append(users, fixtureUser{
			ID:         strconv.Itoa(100000 + i),
			Email:      fmt.Sprintf("user%d@%s", n, domain),
			GivenName:  fmt.Sprintf("User%d", n),
			FamilyName: "Test",
			Department: depts[i%len(depts)],
		})
	}

	if membersPerGroup > numUsers {
		membersPerGroup = numUsers
	}
	groups := make([]fixtureGroup, 0, numGroups)
	for g := range numGroups {
		memberIDs := make([]string, 0, membersPerGroup)
		for j := range membersPerGroup {
			memberIDs = append(memberIDs, users[(g*membersPerGroup+j)%numUsers].ID)
		}
		groups = append(groups, fixtureGroup{
			ID:        fmt.Sprintf("g%d", 1000+g),
			Name:      fmt.Sprintf("Group %d", g+1),
			Email:     fmt.Sprintf("group%d@%s", g+1, domain),
			MemberIDs: memberIDs,
		})
	}

	return fixture{Domain: domain, Users: users, Groups: groups}
}

// ---------------------------------------------------------------------------
// serve
// ---------------------------------------------------------------------------

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	fixturePath := fs.String("fixture", "", "path to fixture JSON (required)")
	addr := fs.String("addr", ":8091", "listen address")
	latency := fs.Duration("latency", 0, "artificial latency added to each Directory API request")
	errorRate := fs.Float64("error-rate", 0, "fraction [0..1] of Directory API requests to fail with 429/503")
	reloadInterval := fs.Duration("reload-interval", 2*time.Second, "how often to check the fixture file for changes")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *fixturePath == "" {
		usage()
	}

	st := &store{}
	if err := st.reload(*fixturePath); err != nil {
		log.Fatalf("load fixture %s: %v", *fixturePath, err)
	}
	go st.watch(*fixturePath, *reloadInterval)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", handleToken)
	mux.HandleFunc("GET /admin/directory/v1/users", withChaos(*latency, *errorRate, st.handleUsers))
	mux.HandleFunc("GET /admin/directory/v1/groups", withChaos(*latency, *errorRate, st.handleGroups))
	mux.HandleFunc("GET /admin/directory/v1/groups/{groupKey}/members", withChaos(*latency, *errorRate, st.handleMembers))

	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	log.Printf("serving fake Admin SDK Directory API on %s (fixture=%s)", *addr, *fixturePath)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// store holds the current fixture and is safe for concurrent reads while the
// watcher swaps in a freshly-loaded fixture.
type store struct {
	mu      sync.RWMutex
	fx      fixture
	modTime time.Time
}

func (s *store) reload(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var fx fixture
	if err := json.Unmarshal(data, &fx); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	s.mu.Lock()
	s.fx = fx
	s.modTime = info.ModTime()
	s.mu.Unlock()
	log.Printf("loaded fixture: %d users, %d groups (domain=%s)", len(fx.Users), len(fx.Groups), fx.Domain)
	return nil
}

// watch polls the fixture file's modtime and reloads when it changes, so QA can
// edit the JSON in place and have the server pick it up without a restart.
func (s *store) watch(path string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		info, err := os.Stat(path)
		if err != nil {
			log.Printf("watch: stat %s: %v", path, err)
			continue
		}
		s.mu.RLock()
		changed := info.ModTime().After(s.modTime)
		s.mu.RUnlock()
		if !changed {
			continue
		}
		if err := s.reload(path); err != nil {
			log.Printf("watch: reload failed, keeping previous fixture: %v", err)
		}
	}
}

func (s *store) snapshot() fixture {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fx
}

func (s *store) handleUsers(w http.ResponseWriter, r *http.Request) {
	fx := s.snapshot()
	start, end, next := page(r, len(fx.Users), 500)
	out := make([]*directory.User, 0, end-start)
	for _, u := range fx.Users[start:end] {
		out = append(out, toDirectoryUser(u))
	}
	writeJSON(w, &directory.Users{
		Kind:          "admin#directory#users",
		Users:         out,
		NextPageToken: next,
	})
}

func (s *store) handleGroups(w http.ResponseWriter, r *http.Request) {
	fx := s.snapshot()
	start, end, next := page(r, len(fx.Groups), 200)
	out := make([]*directory.Group, 0, end-start)
	for _, g := range fx.Groups[start:end] {
		out = append(out, &directory.Group{
			Kind:  "admin#directory#group",
			Id:    g.ID,
			Name:  g.Name,
			Email: g.Email,
		})
	}
	writeJSON(w, &directory.Groups{
		Kind:          "admin#directory#groups",
		Groups:        out,
		NextPageToken: next,
	})
}

func (s *store) handleMembers(w http.ResponseWriter, r *http.Request) {
	groupKey := r.PathValue("groupKey")
	fx := s.snapshot()

	memberIDs := []string{}
	for _, g := range fx.Groups {
		if g.ID == groupKey {
			memberIDs = g.MemberIDs
			break
		}
	}

	start, end, next := page(r, len(memberIDs), 200)
	out := make([]*directory.Member, 0, end-start)
	for _, id := range memberIDs[start:end] {
		out = append(out, &directory.Member{
			Kind: "admin#directory#member",
			Id:   id,
			Type: "USER",
		})
	}
	writeJSON(w, &directory.Members{
		Kind:          "admin#directory#members",
		Members:       out,
		NextPageToken: next,
	})
}

func toDirectoryUser(u fixtureUser) *directory.User {
	du := &directory.User{
		Kind:         "admin#directory#user",
		Id:           u.ID,
		PrimaryEmail: u.Email,
		Suspended:    u.Suspended,
		Archived:     u.Archived,
		Name: &directory.UserName{
			GivenName:  u.GivenName,
			FamilyName: u.FamilyName,
			FullName:   u.GivenName + " " + u.FamilyName,
		},
		// Suspended/Archived are meaningful even when false; force them so the
		// client sees the real (active) state instead of an omitted field.
		ForceSendFields: []string{"Suspended", "Archived"},
	}
	if u.Department != "" {
		du.Organizations = []map[string]any{{"department": u.Department, "primary": true}}
	}
	return du
}

// handleToken fakes the OAuth2 JWT token exchange. It does not verify the signed
// assertion; it just returns a static bearer token so the Directory API client
// can proceed.
func handleToken(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"access_token": "fake-access-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
}

// page parses Google's maxResults/pageToken query params (pageToken is the next
// offset, encoded as a decimal string) and returns the slice bounds plus the
// nextPageToken to advertise (empty when the last page is reached).
func page(r *http.Request, total, defaultSize int) (start, end int, next string) {
	size := defaultSize
	if v := r.URL.Query().Get("maxResults"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			size = n
		}
	}
	if t := r.URL.Query().Get("pageToken"); t != "" {
		if n, err := strconv.Atoi(t); err == nil && n > 0 {
			start = n
		}
	}
	if start > total {
		start = total
	}
	end = start + size
	if end >= total {
		end = total
	} else {
		next = strconv.Itoa(end)
	}
	return start, end, next
}

// withChaos optionally adds latency and injects retryable errors, to exercise
// the client's pagination and retry/backoff under throttling.
func withChaos(latency time.Duration, errorRate float64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if latency > 0 {
			time.Sleep(latency)
		}
		if errorRate > 0 && rand.Float64() < errorRate {
			code := http.StatusTooManyRequests
			if rand.IntN(2) == 0 {
				code = http.StatusServiceUnavailable
			}
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"error":{"code":` + strconv.Itoa(code) + `,"message":"injected failure"}}`))
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}
