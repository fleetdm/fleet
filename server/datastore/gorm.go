package datastore

import (
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // db driver
	_ "github.com/mattn/go-sqlite3"    // db driver

	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/server/config"
	"github.com/kolide/kolide-ose/server/kolide"
)

var tables = [...]interface{}{
	&kolide.User{},
	&kolide.PasswordResetRequest{},
	&kolide.Session{},
	&kolide.Pack{},
	&kolide.PackQuery{},
	&kolide.PackTarget{},
	&kolide.Host{},
	&kolide.Label{},
	&kolide.LabelQueryExecution{},
	&kolide.Option{},
	&kolide.DistributedQueryCampaign{},
	&kolide.DistributedQueryCampaignTarget{},
	&kolide.Query{},
	&kolide.DistributedQueryExecution{},
	&kolide.OrgInfo{},
	&kolide.Invite{},
}

type gormDB struct {
	DB     *gorm.DB
	Driver string
	config config.KolideConfig
}

func (orm gormDB) Name() string {
	return "gorm"
}

func (orm gormDB) Migrate() error {
	for _, table := range tables {
		if err := orm.DB.AutoMigrate(table).Error; err != nil {
			return err
		}
	}

	// Have to manually add indexes. Yuck!
	orm.DB.Model(&kolide.LabelQueryExecution{}).AddUniqueIndex("idx_lqe_label_host", "label_id", "host_id")

	return nil
}

func (orm gormDB) Drop() error {
	var err error
	for _, table := range tables {
		err = orm.DB.DropTableIfExists(table).Error
	}
	return err
}

// create connection with mysql backend, using a backoff timer and maxAttempts
func openGORM(driver, conn string, maxAttempts int) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	for attempts := 1; attempts <= maxAttempts; attempts++ {
		db, err = gorm.Open(driver, conn)
		if err == nil {
			break
		} else {
			if err.Error() == "invalid database source" {
				return nil, err
			}
			// TODO: use a logger
			fmt.Printf("could not connect to mysql: %v\n", err)
			time.Sleep(time.Duration(attempts) * time.Second)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mysql backend, err = %v", err)
	}
	return db, nil
}

// applyLimitOffset applies the appropriate limit and offset parameters to the
// gorm.DB instance, returning a DB that can be chained as usual with *gorm.DB.
func (orm *gormDB) applyListOptions(opt kolide.ListOptions) *gorm.DB {
	if opt.PerPage == 0 {
		// PerPage value of 0 indicates unlimited
		return orm.DB
	}

	offset := opt.Page * opt.PerPage
	return orm.DB.Limit(opt.PerPage).Offset(offset)
}
