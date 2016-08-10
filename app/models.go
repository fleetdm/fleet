package app

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"

	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/kolide/kolide-ose/config"
	"github.com/kolide/kolide-ose/sessions"
)

var tables = [...]interface{}{
	&User{},
	&sessions.Session{},
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
	if config.App.Debug {
		db.LogMode(true)
	}
}

func OpenDB(user, password, address, dbName string) (*gorm.DB, error) {
	connectionString := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local", user, password, address, dbName)
	db, err := gorm.Open("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	setDBSettings(db)
	return db, nil
}

func DropTables(db *gorm.DB) {
	for _, table := range tables {
		db.DropTableIfExists(table)
	}
}

func CreateTables(db *gorm.DB) {
	for _, table := range tables {
		db.AutoMigrate(table)
	}
}
