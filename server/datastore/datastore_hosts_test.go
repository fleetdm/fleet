package datastore

import (
	"fmt"
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
)

var enrollTests = []struct {
	uuid, hostname, ip, platform string
	nodeKeySize                  int
}{
	0: {uuid: "6D14C88F-8ECF-48D5-9197-777647BF6B26",
		hostname:    "web.kolide.co",
		ip:          "172.0.0.1",
		platform:    "linux",
		nodeKeySize: 12,
	},
	1: {uuid: "B998C0EB-38CE-43B1-A743-FBD7A5C9513B",
		hostname:    "mail.kolide.co",
		ip:          "172.0.0.2",
		platform:    "linux",
		nodeKeySize: 10,
	},
	2: {uuid: "008F0688-5311-4C59-86EE-00C2D6FC3EC2",
		hostname:    "home.kolide.co",
		ip:          "127.0.0.1",
		platform:    "darwin",
		nodeKeySize: 25,
	},
	3: {uuid: "uuid123",
		hostname:    "fakehostname",
		ip:          "192.168.1.1",
		platform:    "darwin",
		nodeKeySize: 1,
	},
}

func testEnrollHost(t *testing.T, db kolide.Datastore) {
	var hosts []*kolide.Host
	for _, tt := range enrollTests {
		h, err := db.EnrollHost(tt.uuid, tt.hostname, tt.ip, tt.platform, tt.nodeKeySize)
		assert.Nil(t, err)

		hosts = append(hosts, h)
		assert.Equal(t, tt.uuid, h.UUID)
		assert.Equal(t, tt.hostname, h.HostName)
		assert.Equal(t, tt.ip, h.IPAddress)
		assert.Equal(t, tt.platform, h.Platform)
		assert.NotEmpty(t, h.NodeKey)
	}

	for _, enrolled := range hosts {
		oldNodeKey := enrolled.NodeKey
		newhostname := fmt.Sprintf("changed.%s", enrolled.HostName)

		h, err := db.EnrollHost(enrolled.UUID, newhostname, enrolled.IPAddress, enrolled.Platform, 15)
		assert.Nil(t, err)
		assert.Equal(t, enrolled.UUID, h.UUID)
		assert.NotEmpty(t, h.NodeKey)
		assert.NotEqual(t, oldNodeKey, h.NodeKey)
	}

}

func testAuthenticateHost(t *testing.T, db kolide.Datastore) {
	for _, tt := range enrollTests {
		h, err := db.EnrollHost(tt.uuid, tt.hostname, tt.ip, tt.platform, tt.nodeKeySize)
		assert.Nil(t, err)

		returned, err := db.AuthenticateHost(h.NodeKey)
		assert.Nil(t, err)
		assert.Equal(t, h.NodeKey, returned.NodeKey)
	}

	_, err := db.AuthenticateHost("7B1A9DC9-B042-489F-8D5A-EEC2412C95AA")
	assert.NotNil(t, err)

	_, err = db.AuthenticateHost("")
	assert.NotNil(t, err)
}
