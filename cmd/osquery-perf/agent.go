package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
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

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

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
	for {
		ticker := time.Tick(10 * time.Second)
		select {
		case <-ticker:
			s.Log()
		}
	}
}

type NodeKeyManager struct {
	filepath string

	l        sync.Mutex
	nodekeys []string
}

func (n *NodeKeyManager) LoadKeys() {
	if n.filepath == "" {
		return
	}

	n.l.Lock()
	defer n.l.Unlock()

	data, err := os.ReadFile(n.filepath)
	if err != nil {
		fmt.Println("error loading nodekey file:", err)
		return
	}
	n.nodekeys = strings.Split(string(data), "\n")
	fmt.Printf("loaded %d node keys\n", len(n.nodekeys))
}

func (n *NodeKeyManager) Get(i int) string {
	n.l.Lock()
	defer n.l.Unlock()

	if len(n.nodekeys) > i {
		return n.nodekeys[i]
	}
	return ""
}

func (n *NodeKeyManager) Add(nodekey string) {
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

type Agent struct {
	ServerAddress   string
	EnrollSecret    string
	NodeKey         string
	UUID            string
	FastClient      fasthttp.Client
	Client          http.Client
	ConfigInterval  time.Duration
	QueryInterval   time.Duration
	Templates       *template.Template
	strings         map[string]string
	configTicker    <-chan time.Time
	liveQueryTicker <-chan time.Time
	Stats           *Stats
	NodeKeyManager  *NodeKeyManager
}

func NewAgent(serverAddress, enrollSecret string, templates *template.Template, configInterval, queryInterval time.Duration) *Agent {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	transport.DisableCompression = true
	return &Agent{
		ServerAddress:  serverAddress,
		EnrollSecret:   enrollSecret,
		Templates:      templates,
		ConfigInterval: configInterval,
		QueryInterval:  queryInterval,
		UUID:           uuid.New().String(),
		FastClient:     fasthttp.Client{
			//MaxConnsPerHost: 9999999999999,
			//ReadTimeout:        30 * time.Second,
			//WriteTimeout:       30 * time.Second,
			//MaxConnWaitTimeout: 10 * time.Second,
		},
		Client:  http.Client{Transport: transport},
		strings: make(map[string]string),
	}
}

type enrollResponse struct {
	NodeKey string `json:"node_key"`
}

type distributedReadResponse struct {
	Queries map[string]string `json:"queries"`
}

func (a *Agent) runLoop(i int, onlyAlreadyEnrolled bool) {
	if err := a.Enroll(i, onlyAlreadyEnrolled); err != nil {
		return
	}

	a.Config()
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
			a.Config()
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

func (a *Agent) waitingDo(req *fasthttp.Request, res *fasthttp.Response) {
	err := fasthttp.Do(req, res)
	for err != nil || res.StatusCode() != http.StatusOK {
		//fmt.Println(err, res.StatusCode())
		a.Stats.RecordStats(1, 0, 0)
		<-time.Tick(time.Duration(rand.Intn(120)+1) * time.Second)
		err = fasthttp.Do(req, res)
	}
}

func (a *Agent) Enroll(i int, onlyAlreadyEnrolled bool) error {
	a.NodeKey = a.NodeKeyManager.Get(i)
	if a.NodeKey != "" {
		a.Stats.RecordStats(0, 1, 0)
		return nil
	}

	if onlyAlreadyEnrolled {
		return fmt.Errorf("not enrolled")
	}

	// Give it a bit of time before enrolling so not all come at the "same" time
	time.Sleep(time.Duration(100*i) * time.Millisecond)
	var body bytes.Buffer
	if err := a.Templates.ExecuteTemplate(&body, "enroll", a); err != nil {
		log.Println("execute template:", err)
		return err
	}

	req := fasthttp.AcquireRequest()
	req.SetBody(body.Bytes())
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Add("User-Agent", "osquery/4.6.0")
	req.SetRequestURI(a.ServerAddress + "/api/v1/osquery/enroll")
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

	a.NodeKey = parsedResp.NodeKey
	a.Stats.RecordStats(0, 1, 0)

	a.NodeKeyManager.Add(a.NodeKey)

	return nil
}

func (a *Agent) Config() {
	body := bytes.NewBufferString(`{"node_key": "` + a.NodeKey + `"}`)

	req := fasthttp.AcquireRequest()
	req.SetBody(body.Bytes())
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Add("User-Agent", "osquery/4.6.0")
	req.SetRequestURI(a.ServerAddress + "/api/v1/osquery/config")
	res := fasthttp.AcquireResponse()

	a.waitingDo(req, res)

	fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	if res.StatusCode() != http.StatusOK {
		log.Println("config status:", res.StatusCode())
		return
	}

	// No need to read the config body
}

const stringVals = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_."

func (a *Agent) randomString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(stringVals[rand.Int63()%int64(len(stringVals))])
	}
	return sb.String()
}

func (a *Agent) CachedString(key string) string {
	if val, ok := a.strings[key]; ok {
		return val
	}
	val := a.randomString(12)
	a.strings[key] = val
	return val
}

func (a *Agent) DistributedRead() (*distributedReadResponse, error) {
	req := fasthttp.AcquireRequest()
	req.SetBody([]byte(`{"node_key": "` + a.NodeKey + `"}`))
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Add("User-Agent", "osquery/4.6.0")
	req.SetRequestURI(a.ServerAddress + "/api/v1/osquery/distributed/read")
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

type distributedWriteRequest struct {
	Queries  map[string]json.RawMessage `json:"queries"`
	Statuses map[string]string          `json:"statuses"`
	NodeKey  string                     `json:"node_key"`
}

var defaultQueryResult = json.RawMessage(`[{"foo": "bar"}]`)

const statusSuccess = "0"

func (a *Agent) DistributedWrite(queries map[string]string) {
	var body bytes.Buffer

	if _, ok := queries["fleet_detail_query_network_interface"]; ok {
		// Respond to label/detail queries
		a.Templates.ExecuteTemplate(&body, "distributed_write", a)
	} else {
		// Return a generic response for any other queries
		req := distributedWriteRequest{
			Queries:  make(map[string]json.RawMessage),
			Statuses: make(map[string]string),
			NodeKey:  a.NodeKey,
		}

		for name := range queries {
			req.Queries[name] = defaultQueryResult
			req.Statuses[name] = statusSuccess
		}
		json.NewEncoder(&body).Encode(req)
	}

	req := fasthttp.AcquireRequest()
	req.SetBody(body.Bytes())
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Add("User-Agent", "osquery/4.6.0")
	req.SetRequestURI(a.ServerAddress + "/api/v1/osquery/distributed/write")
	res := fasthttp.AcquireResponse()

	a.waitingDo(req, res)

	fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	a.Stats.RecordStats(0, 0, 1)

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

	flag.Parse()

	rand.Seed(*randSeed)

	tmpl, err := template.ParseGlob("*.tmpl")
	if err != nil {
		log.Fatal("parse templates: ", err)
	}

	// Spread starts over the interval to prevent thunering herd
	sleepTime := *startPeriod / time.Duration(*hostCount)

	stats := &Stats{}
	go stats.runLoop()

	nodeKeyManager := &NodeKeyManager{}
	if nodeKeyFile != nil {
		nodeKeyManager.filepath = *nodeKeyFile
		nodeKeyManager.LoadKeys()
	}

	var agents []*Agent
	for i := 0; i < *hostCount; i++ {
		a := NewAgent(*serverURL, *enrollSecret, tmpl, *configInterval, *queryInterval)
		a.Stats = stats
		a.NodeKeyManager = nodeKeyManager
		agents = append(agents, a)
		go a.runLoop(i, onlyAlreadyEnrolled != nil && *onlyAlreadyEnrolled)
		time.Sleep(sleepTime)
	}

	fmt.Println("Agents running. Kill with C-c.")
	<-make(chan struct{})
}
