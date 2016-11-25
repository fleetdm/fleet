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
	packQueries                     map[uint]*kolide.PackQuery
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

func (orm *Datastore) Name() string {
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

func (orm *Datastore) Migrate() error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()
	orm.nextIDs = make(map[interface{}]uint)
	orm.users = make(map[uint]*kolide.User)
	orm.sessions = make(map[uint]*kolide.Session)
	orm.passwordResets = make(map[uint]*kolide.PasswordResetRequest)
	orm.invites = make(map[uint]*kolide.Invite)
	orm.labels = make(map[uint]*kolide.Label)
	orm.labelQueryExecutions = make(map[uint]*kolide.LabelQueryExecution)
	orm.queries = make(map[uint]*kolide.Query)
	orm.packs = make(map[uint]*kolide.Pack)
	orm.hosts = make(map[uint]*kolide.Host)
	orm.packQueries = make(map[uint]*kolide.PackQuery)
	orm.packTargets = make(map[uint]*kolide.PackTarget)
	orm.distributedQueryExecutions = make(map[uint]kolide.DistributedQueryExecution)
	orm.distributedQueryCampaigns = make(map[uint]kolide.DistributedQueryCampaign)
	orm.distributedQueryCampaignTargets = make(map[uint]kolide.DistributedQueryCampaignTarget)
	return nil
}

func (orm *Datastore) Drop() error {
	return orm.Migrate()
}

func (orm *Datastore) Initialize() error {
	if err := orm.createBuiltinLabels(); err != nil {
		return err
	}

	if err := orm.createDevUsers(); err != nil {
		return err
	}

	if err := orm.createDevHosts(); err != nil {
		return err
	}

	if err := orm.createDevQueries(); err != nil {
		return err
	}

	if err := orm.createDevLabels(); err != nil {
		return err
	}

	if err := orm.createDevOrgInfo(); err != nil {
		return err
	}

	if err := orm.createDevPacksAndQueries(); err != nil {
		return err
	}

	return nil
}

// getLimitOffsetSliceBounds returns the bounds that should be used for
// re-slicing the results to comply with the requested ListOptions. Lack of
// generics forces us to do this rather than reslicing in this method.
func (orm *Datastore) getLimitOffsetSliceBounds(opt kolide.ListOptions, length int) (low uint, high uint) {
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
func (orm *Datastore) nextID(val interface{}) uint {
	valType := reflect.TypeOf(reflect.Indirect(reflect.ValueOf(val)).Interface())
	orm.nextIDs[valType]++
	return orm.nextIDs[valType]
}

func (orm *Datastore) createDevPacksAndQueries() error {
	query1 := &kolide.Query{
		Name:  "Osquery Info",
		Query: "select * from osquery_info",
	}
	query1, err := orm.NewQuery(query1)
	if err != nil {
		return err
	}

	query2 := &kolide.Query{
		Name:     "Launchd",
		Query:    "select * from launchd",
		Platform: "darwin",
	}
	query2, err = orm.NewQuery(query2)
	if err != nil {
		return err
	}

	query3 := &kolide.Query{
		Name:  "registry",
		Query: "select * from osquery_registry",
	}
	query3, err = orm.NewQuery(query3)
	if err != nil {
		return err
	}

	pack1 := &kolide.Pack{
		Name: "Osquery Internal Info",
	}
	pack1, err = orm.NewPack(pack1)
	if err != nil {
		return err
	}

	pack2 := &kolide.Pack{
		Name: "macOS Attacks",
	}
	pack2, err = orm.NewPack(pack2)
	if err != nil {
		return err
	}

	err = orm.AddQueryToPack(query1.ID, pack1.ID)
	if err != nil {
		return err
	}

	err = orm.AddQueryToPack(query3.ID, pack1.ID)
	if err != nil {
		return err
	}

	err = orm.AddQueryToPack(query2.ID, pack2.ID)
	if err != nil {
		return err
	}

	return nil
}

func (orm *Datastore) createBuiltinLabels() error {
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
		_, err := orm.NewLabel(&label)
		if err != nil {
			return err
		}
	}

	return nil
}

// Bootstrap a few users when using the in-memory database.
// Each user's default password will just be their username.
func (orm *Datastore) createDevUsers() error {
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
		err := user.SetPassword(user.Username, orm.config.Auth.SaltKeySize, orm.config.Auth.BcryptCost)
		if err != nil {
			return nil
		}
		_, err = orm.NewUser(&user)
		if err != nil {
			return err
		}
	}

	return nil
}

func (orm *Datastore) createDevQueries() error {
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
		_, err := orm.NewQuery(&query)
		if err != nil {
			return err
		}
	}

	return nil
}

// Bootstrap a few hosts when using the in-memory database.
func (orm *Datastore) createDevHosts() error {
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
			PrimaryMAC:       "C0:11:1B:13:3E:15",
			PrimaryIP:        "192.168.1.10",
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
			PrimaryMAC:       "7e:5c:be:ef:b4:df",
			PrimaryIP:        "192.168.1.11",
			DetailUpdateTime: time.Now().Add(-10 * time.Second),
		},
	}

	for _, host := range hosts {
		host := host
		_, err := orm.NewHost(&host)
		if err != nil {
			return err
		}
	}

	return nil
}

func (orm *Datastore) createDevOrgInfo() error {
	devOrgInfo := &kolide.AppConfig{
		OrgName:    "Kolide",
		OrgLogoURL: fmt.Sprintf("%s/logo.png", orm.config.Server.Address),
	}
	_, err := orm.NewAppConfig(devOrgInfo)
	if err != nil {
		return err
	}
	return nil
}

func (orm *Datastore) createDevLabels() error {
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
		_, err := orm.NewLabel(&label)
		if err != nil {
			return err
		}
	}

	return nil
}
