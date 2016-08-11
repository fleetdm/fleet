package app

import (
	"testing"
)

func TestEnrollHost(t *testing.T) {
	db := openTestDB(t)

	expect := Host{
		UUID:      "uuid123",
		HostName:  "fakehostname",
		IPAddress: "192.168.1.1",
		Platform:  "Mac OSX",
	}

	host, err := EnrollHost(db, expect.UUID, expect.HostName, expect.IPAddress, expect.Platform)
	if err != nil {
		t.Fatal(err.Error())
	}

	if host.UUID != expect.UUID {
		t.Errorf("UUID not as expected: %s != %s", host.UUID, expect.UUID)
	}

	if host.HostName != expect.HostName {
		t.Errorf("HostName not as expected: %s != %s", host.HostName, expect.HostName)
	}

	if host.IPAddress != expect.IPAddress {
		t.Errorf("IPAddress not as expected: %s != %s", host.IPAddress, expect.IPAddress)
	}

	if host.Platform != expect.Platform {
		t.Errorf("Platform not as expected: %s != %s", host.Platform, expect.Platform)
	}

	if host.NodeKey == "" {
		t.Error("Node key was not set")
	}

}

func TestReEnrollHost(t *testing.T) {
	db := openTestDB(t)

	expect := Host{
		UUID:      "uuid123",
		HostName:  "fakehostname",
		IPAddress: "192.168.1.1",
		Platform:  "Mac OSX",
	}

	host, err := EnrollHost(db, expect.UUID, expect.HostName, expect.IPAddress, expect.Platform)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Save the node key to check that it changed
	oldNodeKey := host.NodeKey

	expect.HostName = "newhostname"

	host, err = EnrollHost(db, expect.UUID, expect.HostName, "", "")
	if err != nil {
		t.Fatal(err.Error())
	}

	if host.UUID != expect.UUID {
		t.Errorf("UUID not as expected: %s != %s", host.UUID, expect.UUID)
	}

	if host.HostName != expect.HostName {
		t.Errorf("HostName not as expected: %s != %s", host.HostName, expect.HostName)
	}

	if host.IPAddress != expect.IPAddress {
		t.Errorf("IPAddress not as expected: %s != %s", host.IPAddress, expect.IPAddress)
	}

	if host.Platform != expect.Platform {
		t.Errorf("Platform not as expected: %s != %s", host.Platform, expect.Platform)
	}

	if host.NodeKey == "" {
		t.Error("Node key was not set")
	}

	if host.NodeKey == oldNodeKey {
		t.Error("Node key should have changed")
	}

}
