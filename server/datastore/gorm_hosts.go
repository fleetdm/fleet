package datastore

import (
	"net/http"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm gormDB) EnrollHost(uuid, hostname, ip, platform string, nodeKeySize int) (*kolide.Host, error) {
	if uuid == "" {
		return nil, errors.New("missing uuid for host enrollment", "programmer error?")
	}
	host := kolide.Host{UUID: uuid}
	err := orm.DB.Where(&host).First(&host).Error
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			// Create new Host
			host = kolide.Host{
				UUID:      uuid,
				HostName:  hostname,
				IPAddress: ip,
				Platform:  platform,
			}

		default:
			return nil, err
		}
	}

	// Generate a new key each enrollment
	host.NodeKey, err = generateRandomText(nodeKeySize)
	if err != nil {
		return nil, err
	}

	// Update these fields if provided
	if hostname != "" {
		host.HostName = hostname
	}
	if ip != "" {
		host.IPAddress = ip
	}
	if platform != "" {
		host.Platform = platform
	}

	if err := orm.DB.Save(&host).Error; err != nil {
		return nil, err
	}

	return &host, nil
}

func (orm gormDB) AuthenticateHost(nodeKey string) (*kolide.Host, error) {
	host := kolide.Host{NodeKey: nodeKey}
	err := orm.DB.Where("node_key = ?", host.NodeKey).First(&host).Error
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			e := errors.NewFromError(
				err,
				http.StatusUnauthorized,
				"invalid node key",
			)
			// osqueryd expects the literal string "true" here
			e.Extra = map[string]interface{}{"node_invalid": "true"}
			return nil, e
		default:
			return nil, errors.DatabaseError(err)
		}
	}

	return &host, nil
}

func (orm gormDB) SaveHost(host *kolide.Host) error {
	if err := orm.DB.Save(host).Error; err != nil {
		return errors.DatabaseError(err)
	}
	return nil
}

func (orm gormDB) DeleteHost(host *kolide.Host) error {
	return orm.DB.Delete(host).Error
}

func (orm gormDB) Host(id uint) (*kolide.Host, error) {
	host := &kolide.Host{
		ID: id,
	}
	err := orm.DB.Where(host).First(host).Error
	if err != nil {
		return nil, err
	}
	return host, nil
}

func (orm gormDB) Hosts() ([]*kolide.Host, error) {
	var hosts []*kolide.Host
	err := orm.DB.Find(&hosts).Error
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

func (orm gormDB) NewHost(host *kolide.Host) (*kolide.Host, error) {
	if host == nil {
		return nil, errors.New(
			"error creating host",
			"nil pointer passed to NewHost",
		)
	}
	err := orm.DB.Create(host).Error
	if err != nil {
		return nil, err
	}
	return host, err
}

func (orm gormDB) MarkHostSeen(host *kolide.Host, t time.Time) error {
	err := orm.DB.Exec("UPDATE hosts SET updated_at=? WHERE node_key=?", t, host.NodeKey).Error
	if err != nil {
		return errors.DatabaseError(err)
	}
	host.UpdatedAt = t
	return nil
}
