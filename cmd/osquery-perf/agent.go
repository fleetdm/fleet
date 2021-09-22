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
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
)

type Agent struct {
	ServerAddress  string
	EnrollSecret   string
	NodeKey        string
	UUID           string
	Client         http.Client
	ConfigInterval time.Duration
	QueryInterval  time.Duration
	Templates      *template.Template
	strings        map[string]string
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
		Client:         http.Client{Transport: transport},
		strings:        make(map[string]string),
	}
}

type enrollResponse struct {
	NodeKey string `json:"node_key"`
}

type distributedReadResponse struct {
	Queries map[string]string `json:"queries"`
}

func (a *Agent) runLoop() {
	a.Enroll()

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

func (a *Agent) Enroll() {
	var body bytes.Buffer
	if err := a.Templates.ExecuteTemplate(&body, "enroll", a); err != nil {
		log.Println("execute template:", err)
		return
	}

	req, err := http.NewRequest("POST", a.ServerAddress+"/api/v1/osquery/enroll", &body)
	if err != nil {
		log.Println("create request:", err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "osquery/4.6.0")

	resp, err := a.Client.Do(req)
	if err != nil {
		log.Println("do request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("status:", resp.Status)
		return
	}

	var parsedResp enrollResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsedResp); err != nil {
		log.Println("json parse:", err)
		return
	}

	a.NodeKey = parsedResp.NodeKey
}

func (a *Agent) Config() {
	body := bytes.NewBufferString(`{"node_key": "` + a.NodeKey + `"}`)

	req, err := http.NewRequest("POST", a.ServerAddress+"/api/v1/osquery/config", body)
	if err != nil {
		log.Println("create config request:", err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "osquery/4.6.0")

	resp, err := a.Client.Do(req)
	if err != nil {
		log.Println("do config request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("config status:", resp.Status)
		return
	}

	// No need to read the config body
}

func (a *Agent) DistributedRead() (*distributedReadResponse, error) {
	body := bytes.NewBufferString(`{"node_key": "` + a.NodeKey + `"}`)

	req, err := http.NewRequest("POST", a.ServerAddress+"/api/v1/osquery/distributed/read", body)
	if err != nil {
		return nil, fmt.Errorf("create distributed read request:", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "osquery/4.6.0")

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do distributed read request:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("distributed read status:", resp.Status)
	}

	var parsedResp distributedReadResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsedResp); err != nil {
		return nil, fmt.Errorf("json parse distributed read response:", err)
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

		for name, _ := range queries {
			req.Queries[name] = defaultQueryResult
			req.Statuses[name] = statusSuccess
		}
		json.NewEncoder(&body).Encode(req)
	}

	req, err := http.NewRequest("POST", a.ServerAddress+"/api/v1/osquery/distributed/write", &body)
	if err != nil {
		log.Println("create distributed write request:", err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "osquery/4.6.0")

	resp, err := a.Client.Do(req)
	if err != nil {
		log.Println("do distributed write request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("distributed write status:", resp.Status)
		return
	}

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

	flag.Parse()

	rand.Seed(*randSeed)

	tmpl, err := template.ParseGlob("*.tmpl")
	if err != nil {
		log.Fatal("parse templates: ", err)
	}

	// Spread starts over the interval to prevent thunering herd
	sleepTime := *startPeriod / time.Duration(*hostCount)
	var agents []*Agent
	for i := 0; i < *hostCount; i++ {
		a := NewAgent(*serverURL, *enrollSecret, tmpl, *configInterval, *queryInterval)
		agents = append(agents, a)
		go a.runLoop()
		time.Sleep(sleepTime)
	}

	fmt.Println("Agents running. Kill with C-c.")
	<-make(chan struct{})
}
