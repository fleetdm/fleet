package main

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"

	"github.com/gin-gonic/gin"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// Get the database connection from the context, or panic
func GetDB(c *gin.Context) *gorm.DB {
	return c.MustGet("DB").(*gorm.DB)
}

type BaseModel struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

type ScheduledQuery struct {
	BaseModel
	Name         string `gorm:"not null"`
	QueryID      int
	Query        Query
	Interval     uint `gorm:"not null"`
	Snapshot     bool
	Differential bool
	Platform     string
	PackID       uint
}

type Query struct {
	BaseModel
	Query   string   `gorm:"not null"`
	Targets []Target `gorm:"many2many:query_targets"`
}

type TargetType int

const (
	TargetLabel TargetType = iota
	TargetHost  TargetType = iota
)

type Target struct {
	BaseModel
	Type     TargetType
	QueryID  uint
	TargetID uint
}

type DistributedQueryStatus int

const (
	QueryRunning  DistributedQueryStatus = iota
	QueryComplete DistributedQueryStatus = iota
	QueryError    DistributedQueryStatus = iota
)

type DistributedQuery struct {
	BaseModel
	Query       Query
	MaxDuration time.Duration
	Status      DistributedQueryStatus
	UserID      uint
}

type DistributedQueryExecutionStatus int

const (
	ExecutionWaiting   DistributedQueryExecutionStatus = iota
	ExecutionRequested DistributedQueryExecutionStatus = iota
	ExecutionSucceeded DistributedQueryExecutionStatus = iota
	ExecutionFailed    DistributedQueryExecutionStatus = iota
)

type DistributedQueryExecution struct {
	HostID             uint
	DistributedQueryID uint
	Status             DistributedQueryExecutionStatus
	Error              string `gorm:"size:1024"`
	ExecutionDuration  time.Duration
}

type Pack struct {
	BaseModel
	Name             string `gorm:"not null;unique_index:idx_pack_unique_name"`
	Platform         string
	Queries          []ScheduledQuery
	DiscoveryQueries []DiscoveryQuery
}

type DiscoveryQuery struct {
	BaseModel
	Query string `gorm:"size:1024" gorm:"not null"`
}

type Host struct {
	BaseModel
	NodeKey   string `gorm:"unique_index:idx_host_unique_nodekey"`
	HostName  string
	UUID      string `gorm:"unique_index:idx_host_unique_uuid"`
	IPAddress string
	Platform  string
	Labels    []*Label `gorm:"many2many:host_labels;"`
}

type Label struct {
	BaseModel
	Name  string `gorm:"not null;unique_index:idx_label_unique_name"`
	Query string
	Hosts []Host
}

type Option struct {
	BaseModel
	Key      string `gorm:"not null;unique_index:idx_option_unique_key"`
	Value    string `gorm:"not null"`
	Platform string
}

type DecoratorType int

const (
	DecoratorLoad     DecoratorType = iota
	DecoratorAlways   DecoratorType = iota
	DecoratorInterval DecoratorType = iota
)

type Decorator struct {
	BaseModel
	Type     DecoratorType `gorm:"not null"`
	Interval int
	Query    string
}

var tables = [...]interface{}{
	&User{},
	&ScheduledQuery{},
	&Pack{},
	&DiscoveryQuery{},
	&Host{},
	&Label{},
	&Option{},
	&Decorator{},
	&Target{},
	&DistributedQuery{},
	&Query{},
	&DistributedQueryExecution{},
}

func setDBSettings(db *gorm.DB) {
	// Tell gorm to use the logrus logger
	db.SetLogger(logrus.StandardLogger())

	// If debug mode is enabled, tell gorm to turn on logmode (log each
	// query as it is executed)
	if debug != nil && *debug {
		db.LogMode(true)
	}
}

func openDB(user, password, address, dbName string) (*gorm.DB, error) {
	connectionString := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local", user, password, address, dbName)
	db, err := gorm.Open("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	setDBSettings(db)
	return db, nil
}

func openTestDB() *gorm.DB {
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		panic(fmt.Sprintf("Error opening test DB: %s", err.Error()))
	}

	setDBSettings(db)
	createTables(db)
	if db.Error != nil {
		panic(fmt.Sprintf("Error creating test DB tables: %s", db.Error.Error()))
	}
	return db
}

func dropTables(db *gorm.DB) {
	for _, table := range tables {
		db.DropTableIfExists(table)
	}
}

func createTables(db *gorm.DB) {
	for _, table := range tables {
		db.AutoMigrate(table)
	}
}
