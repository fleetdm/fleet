package inmem

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/kolide/kolide-ose/server/config"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/patrickmn/sortutil"
)

type Datastore struct {
	Driver  string
	mtx     sync.RWMutex
	nextIDs map[interface{}]uint

	users                           map[uint]*kolide.User
	sessions                        map[uint]*kolide.Session
	passwordResets                  map[uint]*kolide.PasswordResetRequest
	invites                         map[uint]*kolide.Invite
	labels                          map[uint]*kolide.Label
	labelQueryExecutions            map[uint]*kolide.LabelQueryExecution
	queries                         map[uint]*kolide.Query
	packs                           map[uint]*kolide.Pack
	hosts                           map[uint]*kolide.Host
	scheduledQueries                map[uint]*kolide.ScheduledQuery
	packTargets                     map[uint]*kolide.PackTarget
	distributedQueryExecutions      map[uint]kolide.DistributedQueryExecution
	distributedQueryCampaigns       map[uint]kolide.DistributedQueryCampaign
	distributedQueryCampaignTargets map[uint]kolide.DistributedQueryCampaignTarget

	orginfo *kolide.AppConfig
	config  *config.KolideConfig
}

func New(config config.KolideConfig) (*Datastore, error) {
	ds := &Datastore{
		Driver: "inmem",
		config: &config,
	}

	if err := ds.Migrate(); err != nil {
		return nil, err
	}

	return ds, nil
}

func (d *Datastore) Name() string {
	return "inmem"
}

func sortResults(slice interface{}, opt kolide.ListOptions, fields map[string]string) error {
	field, ok := fields[opt.OrderKey]
	if !ok {
		return errors.New("cannot sort on unknown key: " + opt.OrderKey)
	}

	if opt.OrderDirection == kolide.OrderDescending {
		sortutil.DescByField(slice, field)
	} else {
		sortutil.AscByField(slice, field)
	}

	return nil
}

func (d *Datastore) Migrate() error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.nextIDs = make(map[interface{}]uint)
	d.users = make(map[uint]*kolide.User)
	d.sessions = make(map[uint]*kolide.Session)
	d.passwordResets = make(map[uint]*kolide.PasswordResetRequest)
	d.invites = make(map[uint]*kolide.Invite)
	d.labels = make(map[uint]*kolide.Label)
	d.labelQueryExecutions = make(map[uint]*kolide.LabelQueryExecution)
	d.queries = make(map[uint]*kolide.Query)
	d.packs = make(map[uint]*kolide.Pack)
	d.hosts = make(map[uint]*kolide.Host)
	d.scheduledQueries = make(map[uint]*kolide.ScheduledQuery)
	d.packTargets = make(map[uint]*kolide.PackTarget)
	d.distributedQueryExecutions = make(map[uint]kolide.DistributedQueryExecution)
	d.distributedQueryCampaigns = make(map[uint]kolide.DistributedQueryCampaign)
	d.distributedQueryCampaignTargets = make(map[uint]kolide.DistributedQueryCampaignTarget)
	return nil
}

func (d *Datastore) Drop() error {
	return d.Migrate()
}

func (d *Datastore) Initialize() error {
	if err := d.createBuiltinLabels(); err != nil {
		return err
	}

	if err := d.createDevUsers(); err != nil {
		return err
	}

	if err := d.createDevHosts(); err != nil {
		return err
	}

	if err := d.createDevQueries(); err != nil {
		return err
	}

	if err := d.createDevLabels(); err != nil {
		return err
	}

	if err := d.createDevOrgInfo(); err != nil {
		return err
	}

	if err := d.createDevPacksAndQueries(); err != nil {
		return err
	}

	return nil
}

// getLimitOffsetSliceBounds returns the bounds that should be used for
// re-slicing the results to comply with the requested ListOptions. Lack of
// generics forces us to do this rather than reslicing in this method.
func (d *Datastore) getLimitOffsetSliceBounds(opt kolide.ListOptions, length int) (low uint, high uint) {
	if opt.PerPage == 0 {
		// PerPage value of 0 indicates unlimited
		return 0, uint(length)
	}

	offset := opt.Page * opt.PerPage
	max := offset + opt.PerPage
	if offset > uint(length) {
		offset = uint(length)
	}
	if max > uint(length) {
		max = uint(length)
	}
	return offset, max
}

// nextID returns the next ID value that should be used for a struct of the
// given type
func (d *Datastore) nextID(val interface{}) uint {
	valType := reflect.TypeOf(reflect.Indirect(reflect.ValueOf(val)).Interface())
	d.nextIDs[valType]++
	return d.nextIDs[valType]
}

func (d *Datastore) createDevPacksAndQueries() error {
	query1 := &kolide.Query{
		Name:  "Osquery Info",
		Query: "select * from osquery_info",
	}
	query1, err := d.NewQuery(query1)
	if err != nil {
		return err
	}

	query2 := &kolide.Query{
		Name:  "Launchd",
		Query: "select * from launchd",
	}
	query2, err = d.NewQuery(query2)
	if err != nil {
		return err
	}

	query3 := &kolide.Query{
		Name:  "registry",
		Query: "select * from osquery_registry",
	}
	query3, err = d.NewQuery(query3)
	if err != nil {
		return err
	}

	pack1 := &kolide.Pack{
		Name: "Osquery Internal Info",
	}
	pack1, err = d.NewPack(pack1)
	if err != nil {
		return err
	}

	pack2 := &kolide.Pack{
		Name: "macOS Attacks",
	}
	pack2, err = d.NewPack(pack2)
	if err != nil {
		return err
	}

	_, err = d.NewScheduledQuery(&kolide.ScheduledQuery{
		QueryID:  query1.ID,
		PackID:   pack1.ID,
		Interval: 60,
	})
	if err != nil {
		return err
	}

	t := true
	_, err = d.NewScheduledQuery(&kolide.ScheduledQuery{
		QueryID:  query3.ID,
		PackID:   pack1.ID,
		Interval: 60,
		Snapshot: &t,
	})
	if err != nil {
		return err
	}

	_, err = d.NewScheduledQuery(&kolide.ScheduledQuery{
		QueryID:  query2.ID,
		PackID:   pack2.ID,
		Interval: 60,
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *Datastore) createBuiltinLabels() error {
	labels := []kolide.Label{
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Now().UTC(),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Now().UTC(),
				},
			},
			Platform:  "darwin",
			Name:      "Mac OS X",
			Query:     "select 1 from osquery_info where build_platform = 'darwin';",
			LabelType: kolide.LabelTypeBuiltIn,
		},
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Now().UTC(),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Now().UTC(),
				},
			},
			Platform:  "ubuntu",
			Name:      "Ubuntu Linux",
			Query:     "select 1 from osquery_info where build_platform = 'ubuntu';",
			LabelType: kolide.LabelTypeBuiltIn,
		},
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Now().UTC(),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Now().UTC(),
				},
			},
			Platform:  "centos",
			Name:      "CentOS Linux",
			Query:     "select 1 from osquery_info where build_platform = 'centos';",
			LabelType: kolide.LabelTypeBuiltIn,
		},
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Now().UTC(),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Now().UTC(),
				},
			},
			Platform:  "windows",
			Name:      "MS Windows",
			Query:     "select 1 from osquery_info where build_platform = 'windows';",
			LabelType: kolide.LabelTypeBuiltIn,
		},
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Now().UTC(),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Now().UTC(),
				},
			},
			Platform:  "all",
			Name:      "All Hosts",
			Query:     "select 1;",
			LabelType: kolide.LabelTypeBuiltIn,
		},
	}

	for _, label := range labels {
		label := label
		_, err := d.NewLabel(&label)
		if err != nil {
			return err
		}
	}

	return nil
}

// Bootstrap a few users when using the in-memory database.
// Each user's default password will just be their username.
func (d *Datastore) createDevUsers() error {
	users := []kolide.User{
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Date(2016, time.October, 27, 10, 0, 0, 0, time.UTC),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Date(2016, time.October, 27, 10, 0, 0, 0, time.UTC),
				},
			},

			Name:     "Admin User",
			Username: "admin",
			Email:    "admin@kolide.co",
			Position: "Director of Security",
			Admin:    true,
			Enabled:  true,
		},
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Now().Add(-3 * time.Hour),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Now().Add(-1 * time.Hour),
				},
			},

			Name:     "Normal User",
			Username: "user",
			Email:    "user@kolide.co",
			Position: "Security Engineer",
			Admin:    false,
			Enabled:  true,
		},
	}
	for _, user := range users {
		user := user
		err := user.SetPassword(user.Username, d.config.Auth.SaltKeySize, d.config.Auth.BcryptCost)
		if err != nil {
			return nil
		}
		_, err = d.NewUser(&user)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Datastore) createDevQueries() error {
	queries := []kolide.Query{
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Date(2016, time.October, 17, 7, 6, 0, 0, time.UTC),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Date(2016, time.October, 17, 7, 6, 0, 0, time.UTC),
				},
			},

			Name:  "dev_query_1",
			Query: "select * from processes",
		},
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Date(2016, time.October, 27, 4, 3, 10, 0, time.UTC),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Date(2016, time.October, 27, 4, 3, 10, 0, time.UTC),
				},
			},
			Name:  "dev_query_2",
			Query: "select * from time",
		},
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Now().Add(-24 * time.Hour),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Now().Add(-17 * time.Hour),
				},
			},

			Name:  "dev_query_3",
			Query: "select * from cpuid",
		},
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Now().Add(-1 * time.Hour),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Now().Add(-30 * time.Hour),
				},
			},

			Name:  "dev_query_4",
			Query: "select 1 from processes where name like '%Apache%'",
		},
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Now(),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Now(),
				},
			},
			Name:  "dev_query_5",
			Query: "select 1 from osquery_info where build_platform='darwin'",
		},
	}

	for _, query := range queries {
		query := query
		_, err := d.NewQuery(&query)
		if err != nil {
			return err
		}
	}

	return nil
}

// Bootstrap a few hosts when using the in-memory database.
func (d *Datastore) createDevHosts() error {
	hosts := []kolide.Host{
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Date(2016, time.October, 27, 10, 0, 0, 0, time.UTC),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Now().Add(-20 * time.Minute),
				},
			},
			NodeKey:          "totally-legit",
			HostName:         "jmeller-mbp.local",
			UUID:             "1234-5678-9101",
			Platform:         "darwin",
			OsqueryVersion:   "2.0.0",
			OSVersion:        "Mac OS X 10.11.6",
			Uptime:           60 * time.Minute,
			PhysicalMemory:   4145483776,
			DetailUpdateTime: time.Now().Add(-20 * time.Minute),
		},
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Date(2016, time.October, 27, 4, 3, 10, 0, time.UTC),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Date(2016, time.October, 27, 4, 3, 10, 0, time.UTC),
				},
			},

			NodeKey:          "definitely-legit",
			HostName:         "marpaia.local",
			UUID:             "1234-5678-9102",
			Platform:         "windows",
			OsqueryVersion:   "2.0.0",
			OSVersion:        "Windows 10.0.0",
			Uptime:           60 * time.Minute,
			PhysicalMemory:   17179869184,
			DetailUpdateTime: time.Now().Add(-10 * time.Second),
		},
	}

	for _, host := range hosts {
		host := host
		_, err := d.NewHost(&host)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Datastore) createDevOrgInfo() error {
	devOrgInfo := &kolide.AppConfig{
		OrgName:    "Kolide",
		OrgLogoURL: fmt.Sprintf("%s/logo.png", d.config.Server.Address),
	}
	_, err := d.NewAppConfig(devOrgInfo)
	if err != nil {
		return err
	}
	return nil
}

func (d *Datastore) createDevLabels() error {
	labels := []kolide.Label{
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Date(2016, time.October, 27, 8, 31, 16, 0, time.UTC),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Date(2016, time.October, 27, 8, 31, 16, 0, time.UTC),
				},
			},
			Name:  "dev_label_apache",
			Query: "select * from processes where name like '%Apache%'",
		},
		{
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Now().Add(-1 * time.Hour),
				},
				UpdateTimestamp: kolide.UpdateTimestamp{
					UpdatedAt: time.Now(),
				},
			},

			Name:  "dev_label_darwin",
			Query: "select * from osquery_info where build_platform='darwin'",
		},
	}

	for _, label := range labels {
		label := label
		_, err := d.NewLabel(&label)
		if err != nil {
			return err
		}
	}

	return nil
}
