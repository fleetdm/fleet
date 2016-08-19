package datastore

import (
	"fmt"
	"testing"

	"github.com/kolide/kolide-ose/kolide"
)

func TestEnrollHost(t *testing.T) {
	db := setup(t)
	defer teardown(t, db)

	testEnrollHost(t, db)

}

func TestAuthenticateHost(t *testing.T) {
	db := setup(t)
	defer teardown(t, db)

	testAuthenticateHost(t, db)

}

func testAuthenticateHost(t *testing.T, db kolide.HostStore) {
	for i, tt := range enrollTests {
		h, err := db.EnrollHost(tt.uuid, tt.hostname, tt.ip, tt.platform, tt.nodeKeySize)
		if err != nil {
			t.Fatalf("failed to enroll host. test # %v, err=%v", i, err)
		}

		returned, err := db.AuthenticateHost(h.NodeKey)
		if err != nil {
			t.Fatal(err)
		}
		if returned.NodeKey != h.NodeKey {
			t.Errorf("expected nodekey: %v, got %v", h.NodeKey, returned.NodeKey)
		}
	}

	_, err := db.AuthenticateHost("7B1A9DC9-B042-489F-8D5A-EEC2412C95AA")
	if err == nil {
		t.Errorf("expected an error for missing host, but got nil")
	}
}

func testEnrollHost(t *testing.T, db kolide.HostStore) {
	var hosts []*kolide.Host
	for i, tt := range enrollTests {
		h, err := db.EnrollHost(tt.uuid, tt.hostname, tt.ip, tt.platform, tt.nodeKeySize)
		if err != nil {
			t.Fatalf("failed to enroll host. test # %v, err=%v", i, err)
		}

		hosts = append(hosts, h)

		if h.UUID != tt.uuid {
			t.Errorf("expected %s, got %s, test # %v", tt.uuid, h.UUID, i)
		}

		if h.HostName != tt.hostname {
			t.Errorf("expected %s, got %s", tt.hostname, h.HostName)
		}

		if h.IPAddress != tt.ip {
			t.Errorf("expected %s, got %s", tt.ip, h.IPAddress)
		}

		if h.Platform != tt.platform {
			t.Errorf("expected %s, got %s", tt.platform, h.Platform)
		}

		if h.NodeKey == "" {
			t.Errorf("node key was not set, test # %v", i)
		}
	}

	// test re-enrollment
	for i, enrolled := range hosts {
		oldNodeKey := enrolled.NodeKey
		newhostname := fmt.Sprintf("changed.%s", enrolled.HostName)

		h, err := db.EnrollHost(enrolled.UUID, newhostname, enrolled.IPAddress, enrolled.Platform, 15)
		if err != nil {
			t.Fatalf("failed to re-enroll host. test # %v, err=%v", i, err)
		}
		if h.UUID != enrolled.UUID {
			t.Errorf("expected %s, got %s, test # %v", enrolled.UUID, h.UUID, i)
		}

		if h.NodeKey == "" {
			t.Errorf("node key was not set, test # %v", i)
		}

		if h.NodeKey == oldNodeKey {
			t.Errorf("node key should have changed, test # %v", i)
		}

	}
}

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
		platform:    "Mac OSX",
		nodeKeySize: 25,
	},
	3: {uuid: "uuid123",
		hostname:    "fakehostname",
		ip:          "192.168.1.1",
		platform:    "Mac OSX",
		nodeKeySize: 1,
	},
}
