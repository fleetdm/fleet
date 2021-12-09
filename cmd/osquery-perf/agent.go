package main

import (
	"bytes"
	"crypto/tls"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

//go:embed *.tmpl
var templatesFS embed.FS

type Stats struct {
	errors            int
	enrollments       int
	distributedwrites int

	l sync.Mutex
}

func (s *Stats) RecordStats(errors int, enrollments int, distributedwrites int) {
	s.l.Lock()
	defer s.l.Unlock()

	s.errors += errors
	s.enrollments += enrollments
	s.distributedwrites += distributedwrites
}

func (s *Stats) Log() {
	s.l.Lock()
	defer s.l.Unlock()

	fmt.Printf(
		"%s :: error rate: %.2f \t enrollments: %d \t writes: %d\n",
		time.Now().String(),
		float64(s.errors)/float64(s.enrollments),
		s.enrollments,
		s.distributedwrites,
	)
}

func (s *Stats) runLoop() {
	ticker := time.Tick(10 * time.Second)
	for range ticker {
		s.Log()
	}
}

type nodeKeyManager struct {
	filepath string

	l        sync.Mutex
	nodekeys []string
}

func (n *nodeKeyManager) LoadKeys() {
	if n.filepath == "" {
		return
	}

	n.l.Lock()
	defer n.l.Unlock()

	data, err := os.ReadFile(n.filepath)
	if err != nil {
		fmt.Println("WARNING (ignore if creating a new node key file): error loading nodekey file:", err)
		return
	}
	n.nodekeys = strings.Split(string(data), "\n")
	n.nodekeys = n.nodekeys[:len(n.nodekeys)-1] // remove last empty node key due to new line.
	fmt.Printf("loaded %d node keys\n", len(n.nodekeys))
}

func (n *nodeKeyManager) Get(i int) string {
	n.l.Lock()
	defer n.l.Unlock()

	if len(n.nodekeys) > i {
		return n.nodekeys[i]
	}
	return ""
}

func (n *nodeKeyManager) Add(nodekey string) {
	if n.filepath == "" {
		return
	}

	// we lock just to make sure we write one at a time
	n.l.Lock()
	defer n.l.Unlock()

	f, err := os.OpenFile(n.filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Println("error opening nodekey file:", err.Error())
		return
	}
	defer f.Close()
	if _, err := f.WriteString(nodekey + "\n"); err != nil {
		fmt.Println("error writing nodekey file:", err)
	}
}

type agent struct {
	agentIndex     int
	softwareCount  entityCount
	userCount      entityCount
	policyPassProb float64
	strings        map[string]string
	serverAddress  string
	fastClient     fasthttp.Client
	stats          *Stats
	nodeKeyManager *nodeKeyManager
	nodeKey        string
	templates      *template.Template

	scheduledQueries []string

	// The following are exported to be used by the templates.

	EnrollSecret   string
	UUID           string
	ConfigInterval time.Duration
	QueryInterval  time.Duration
}

type entityCount struct {
	common int
	unique int
}

func newAgent(
	agentIndex int,
	serverAddress, enrollSecret string, templates *template.Template,
	configInterval, queryInterval time.Duration, softwareCount, userCount entityCount,
	policyPassProb float64,
) *agent {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	transport.DisableCompression = true
	return &agent{
		agentIndex:     agentIndex,
		serverAddress:  serverAddress,
		softwareCount:  softwareCount,
		userCount:      userCount,
		strings:        make(map[string]string),
		policyPassProb: policyPassProb,
		fastClient: fasthttp.Client{
			TLSConfig: &tls.Config{InsecureSkipVerify: true},
		},
		templates: templates,

		EnrollSecret:   enrollSecret,
		ConfigInterval: configInterval,
		QueryInterval:  queryInterval,
		UUID:           uuid.New().String(),
	}
}

type enrollResponse struct {
	NodeKey string `json:"node_key"`
}

type distributedReadResponse struct {
	Queries map[string]string `json:"queries"`
}

func (a *agent) runLoop(i int, onlyAlreadyEnrolled bool) {
	if err := a.enroll(i, onlyAlreadyEnrolled); err != nil {
		return
	}

	a.config()
	resp, err := a.DistributedRead()
	if err != nil {
		log.Println(err)
	} else {
		if len(resp.Queries) > 0 {
			a.DistributedWrite(resp.Queries)
		}
	}

	configTicker := time.Tick(a.ConfigInterval)
	liveQueryTicker := time.Tick(a.QueryInterval)
	for {
		select {
		case <-configTicker:
			a.config()
		case <-liveQueryTicker:
			resp, err := a.DistributedRead()
			if err != nil {
				log.Println(err)
			} else {
				if len(resp.Queries) > 0 {
					a.DistributedWrite(resp.Queries)
				}
			}
		}
	}
}

func (a *agent) waitingDo(req *fasthttp.Request, res *fasthttp.Response) {
	err := a.fastClient.Do(req, res)
	for err != nil || res.StatusCode() != http.StatusOK {
		fmt.Println(err, res.StatusCode())
		a.stats.RecordStats(1, 0, 0)
		<-time.Tick(time.Duration(rand.Intn(120)+1) * time.Second)
		err = a.fastClient.Do(req, res)
	}
}

func (a *agent) enroll(i int, onlyAlreadyEnrolled bool) error {
	a.nodeKey = a.nodeKeyManager.Get(i)
	if a.nodeKey != "" {
		a.stats.RecordStats(0, 1, 0)
		return nil
	}

	if onlyAlreadyEnrolled {
		return errors.New("not enrolled")
	}

	var body bytes.Buffer
	if err := a.templates.ExecuteTemplate(&body, "enroll", a); err != nil {
		log.Println("execute template:", err)
		return err
	}

	req := fasthttp.AcquireRequest()
	req.SetBody(body.Bytes())
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Add("User-Agent", "osquery/4.6.0")
	req.SetRequestURI(a.serverAddress + "/api/v1/osquery/enroll")
	res := fasthttp.AcquireResponse()

	a.waitingDo(req, res)

	fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	if res.StatusCode() != http.StatusOK {
		log.Println("enroll status:", res.StatusCode())
		return fmt.Errorf("status code: %d", res.StatusCode())
	}

	var parsedResp enrollResponse
	if err := json.Unmarshal(res.Body(), &parsedResp); err != nil {
		log.Println("json parse:", err)
		return err
	}

	a.nodeKey = parsedResp.NodeKey
	a.stats.RecordStats(0, 1, 0)

	a.nodeKeyManager.Add(a.nodeKey)

	return nil
}

func (a *agent) config() {
	body := bytes.NewBufferString(`{"node_key": "` + a.nodeKey + `"}`)

	req := fasthttp.AcquireRequest()
	req.SetBody(body.Bytes())
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Add("User-Agent", "osquery/4.6.0")
	req.SetRequestURI(a.serverAddress + "/api/v1/osquery/config")
	res := fasthttp.AcquireResponse()

	a.waitingDo(req, res)

	fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	if res.StatusCode() != http.StatusOK {
		log.Println("config status:", res.StatusCode())
		return
	}

	parsedResp := struct {
		Packs map[string]struct {
			Queries map[string]interface{} `json:"queries"`
		} `json:"packs"`
	}{}
	if err := json.Unmarshal(res.Body(), &parsedResp); err != nil {
		log.Println("json parse at config:", err)
		return
	}

	var scheduledQueries []string
	for packName, pack := range parsedResp.Packs {
		for queryName := range pack.Queries {
			scheduledQueries = append(scheduledQueries, packName+"_"+queryName)
		}
	}
	a.scheduledQueries = scheduledQueries

	// No need to read the config body
}

const stringVals = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_."

func (a *agent) randomString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(stringVals[rand.Int63()%int64(len(stringVals))])
	}
	return sb.String()
}

func (a *agent) CachedString(key string) string {
	if val, ok := a.strings[key]; ok {
		return val
	}
	val := a.randomString(12)
	a.strings[key] = val
	return val
}

func (a *agent) HostUsersMacOS() []fleet.HostUser {
	groupNames := []string{"staff", "nobody", "wheel", "tty", "daemon"}
	shells := []string{"/bin/zsh", "/bin/sh", "/usr/bin/false", "/bin/bash"}
	commonUsers := make([]fleet.HostUser, a.userCount.common)
	for i := 0; i < len(commonUsers); i++ {
		commonUsers[i] = fleet.HostUser{
			Uid:       uint(i),
			Username:  fmt.Sprintf("Common_%d", i),
			Type:      "", // Empty for macOS.
			GroupName: groupNames[i%len(groupNames)],
			Shell:     shells[i%len(shells)],
		}
	}
	uniqueUsers := make([]fleet.HostUser, a.userCount.unique)
	for i := 0; i < len(uniqueUsers); i++ {
		uniqueUsers[i] = fleet.HostUser{
			Uid:       uint(i),
			Username:  fmt.Sprintf("Unique_%d_%d", a.agentIndex, i),
			Type:      "", // Empty for macOS.
			GroupName: groupNames[i%len(groupNames)],
			Shell:     shells[i%len(shells)],
		}
	}
	users := append(commonUsers, uniqueUsers...)
	rand.Shuffle(len(users), func(i, j int) {
		users[i], users[j] = users[j], users[i]
	})
	return users
}

func (a *agent) SoftwareMacOS() []fleet.Software {
	commonSoftware := make([]fleet.Software, a.softwareCount.common)
	for i := 0; i < len(commonSoftware); i++ {
		commonSoftware[i] = fleet.Software{
			Name:             fmt.Sprintf("Common_%d", i),
			Version:          "0.0.1",
			BundleIdentifier: "com.fleetdm.osquery-perf",
			Source:           "osquery-perf",
		}
	}
	uniqueSoftware := make([]fleet.Software, a.softwareCount.unique)
	for i := 0; i < len(uniqueSoftware); i++ {
		uniqueSoftware[i] = fleet.Software{
			Name:             fmt.Sprintf("Unique_%d_%d", a.agentIndex, i),
			Version:          "1.1.1",
			BundleIdentifier: "com.fleetdm.osquery-perf",
			Source:           "osquery-perf",
		}
	}
	software := append(commonSoftware, uniqueSoftware...)
	rand.Shuffle(len(software), func(i, j int) {
		software[i], software[j] = software[j], software[i]
	})
	return software
}

func (a *agent) DistributedRead() (*distributedReadResponse, error) {
	req := fasthttp.AcquireRequest()
	req.SetBody([]byte(`{"node_key": "` + a.nodeKey + `"}`))
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Add("User-Agent", "osquery/4.6.0")
	req.SetRequestURI(a.serverAddress + "/api/v1/osquery/distributed/read")
	res := fasthttp.AcquireResponse()

	a.waitingDo(req, res)

	fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	var parsedResp distributedReadResponse
	if err := json.Unmarshal(res.Body(), &parsedResp); err != nil {
		log.Println("json parse:", err)
		return nil, err
	}

	return &parsedResp, nil
}

var defaultQueryResult = []map[string]string{
	{"foo": "bar"},
}

func (a *agent) runPolicy(query string) []map[string]string {
	if rand.Float64() <= a.policyPassProb {
		return []map[string]string{
			{"1": "1"},
		}
	}
	return nil
}

func (a *agent) randomQueryStats() []map[string]string {
	var stats []map[string]string
	for _, scheduledQuery := range a.scheduledQueries {
		stats = append(stats, map[string]string{
			"name":           scheduledQuery,
			"delimiter":      "_",
			"average_memory": fmt.Sprint(rand.Intn(200) + 10),
			"denylisted":     "false",
			"executions":     fmt.Sprint(rand.Intn(100) + 1),
			"interval":       fmt.Sprint(rand.Intn(100) + 1),
			"last_executed":  fmt.Sprint(time.Now().Unix()),
			"output_size":    fmt.Sprint(rand.Intn(100) + 1),
			"system_time":    fmt.Sprint(rand.Intn(4000) + 10),
			"user_time":      fmt.Sprint(rand.Intn(4000) + 10),
			"wall_time":      fmt.Sprint(rand.Intn(4000) + 10),
		})
	}
	return stats
}

func (a *agent) DistributedWrite(queries map[string]string) {
	r := service.SubmitDistributedQueryResultsRequest{
		Results:  make(fleet.OsqueryDistributedQueryResults),
		Statuses: make(map[string]fleet.OsqueryStatus),
	}
	r.NodeKey = a.nodeKey
	const hostPolicyQueryPrefix = "fleet_policy_query_"
	const hostDetailQueryPrefix = "fleet_detail_query_"
	for name := range queries {
		r.Results[name] = defaultQueryResult
		r.Statuses[name] = fleet.StatusOK
		if strings.HasPrefix(name, hostPolicyQueryPrefix) {
			r.Results[name] = a.runPolicy(queries[name])
			continue
		}
		if name == hostDetailQueryPrefix+"scheduled_query_stats" {
			r.Results[name] = a.randomQueryStats()
			continue
		}
		if t := a.templates.Lookup(name); t == nil {
			continue
		}
		var ni bytes.Buffer
		err := a.templates.ExecuteTemplate(&ni, name, a)
		if err != nil {
			panic(err)
		}
		var m []map[string]string
		err = json.Unmarshal(ni.Bytes(), &m)
		if err != nil {
			panic(err)
		}
		r.Results[name] = m
	}
	body, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	req := fasthttp.AcquireRequest()
	req.SetBody(body)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Add("User-Agent", "osquery/4.6.0")
	req.SetRequestURI(a.serverAddress + "/api/v1/osquery/distributed/write")
	res := fasthttp.AcquireResponse()

	a.waitingDo(req, res)

	fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	a.stats.RecordStats(0, 0, 1)
	// No need to read the distributed write body
}

func main() {
	serverURL := flag.String("server_url", "https://localhost:8080", "URL (with protocol and port of osquery server)")
	enrollSecret := flag.String("enroll_secret", "", "Enroll secret to authenticate enrollment")
	hostCount := flag.Int("host_count", 10, "Number of hosts to start (default 10)")
	randSeed := flag.Int64("seed", time.Now().UnixNano(), "Seed for random generator (default current time)")
	startPeriod := flag.Duration("start_period", 10*time.Second, "Duration to spread start of hosts over")
	configInterval := flag.Duration("config_interval", 1*time.Minute, "Interval for config requests")
	queryInterval := flag.Duration("query_interval", 10*time.Second, "Interval for live query requests")
	onlyAlreadyEnrolled := flag.Bool("only_already_enrolled", false, "Only start agents that are already enrolled")
	nodeKeyFile := flag.String("node_key_file", "", "File with node keys to use")
	commonSoftwareCount := flag.Int("common_software_count", 10, "Number of common of installed applications reported to fleet")
	uniqueSoftwareCount := flag.Int("unique_software_count", 10, "Number of unique installed applications reported to fleet")
	commonUserCount := flag.Int("common_user_count", 10, "Number of common host users reported to fleet")
	uniqueUserCount := flag.Int("unique_user_count", 10, "Number of unique host users reported to fleet")
	policyPassProb := flag.Float64("policy_pass_prob", 1.0, "Probability of policies to pass [0, 1]")

	flag.Parse()

	rand.Seed(*randSeed)

	// Currently all hosts will be macOS.
	tmpl, err := template.ParseFS(templatesFS, "mac10.14.6.tmpl")
	if err != nil {
		log.Fatal("parse templates: ", err)
	}

	// Spread starts over the interval to prevent thundering herd
	sleepTime := *startPeriod / time.Duration(*hostCount)

	stats := &Stats{}
	go stats.runLoop()

	nodeKeyManager := &nodeKeyManager{}
	if nodeKeyFile != nil {
		nodeKeyManager.filepath = *nodeKeyFile
		nodeKeyManager.LoadKeys()
	}

	for i := 0; i < *hostCount; i++ {
		a := newAgent(i+1, *serverURL, *enrollSecret, tmpl, *configInterval, *queryInterval, entityCount{
			common: *commonSoftwareCount,
			unique: *uniqueSoftwareCount,
		}, entityCount{
			common: *commonUserCount,
			unique: *uniqueUserCount,
		}, *policyPassProb)
		a.stats = stats
		a.nodeKeyManager = nodeKeyManager
		go a.runLoop(i, onlyAlreadyEnrolled != nil && *onlyAlreadyEnrolled)
		time.Sleep(sleepTime)
	}

	fmt.Println("Agents running. Kill with C-c.")
	<-make(chan struct{})
}
