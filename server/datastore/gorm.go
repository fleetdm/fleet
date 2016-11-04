package datastore

import (
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // db driver

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
	&kolide.AppConfig{},
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
	err := orm.DB.Model(&kolide.LabelQueryExecution{}).AddUniqueIndex("idx_lqe_label_host", "label_id", "host_id").Error
	if err != nil {
		return err
	}

	indexes := []interface{}{}
	err = orm.DB.Raw("SELECT * FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA = 'kolide' AND INDEX_NAME = 'hosts_search';").Scan(&indexes).Error
	if err != nil {
		return err
	}
	if len(indexes) == 0 {
		err = orm.DB.Exec("CREATE FULLTEXT INDEX hosts_search ON hosts(host_name, primary_ip);").Error
		if err != nil {
			return err
		}
	}

	indexes = []interface{}{}
	err = orm.DB.Raw("SELECT * FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA = 'kolide' AND INDEX_NAME = 'labels_search';").Scan(&indexes).Error
	if err != nil {
		return err
	}
	if len(indexes) == 0 {
		err = orm.DB.Exec("CREATE FULLTEXT INDEX labels_search ON labels(name);").Error
		if err != nil {
			return err
		}
	}

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
	db := orm.DB
	if opt.PerPage != 0 {
		// PerPage value of 0 indicates unlimited
		offset := opt.Page * opt.PerPage
		db = db.Limit(opt.PerPage).Offset(offset)
	}

	if opt.OrderKey != "" {
		var dir string
		if opt.OrderDirection == kolide.OrderDescending {
			dir = "DESC"
		} else {
			dir = "ASC"
		}

		db = db.Order(opt.OrderKey + " " + dir)
	}

	return db
}
