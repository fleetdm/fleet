package main

import "testing"

func TestPackage(t *testing.T) {

	// --type is required
	runAppCheckErr(t, []string{"package", "deb"}, "Required flag \"type\" not set")

	// if you provide -fleet-url & --enroll-secret are required together
	runAppCheckErr(t, []string{"package", "--type=deb", "--fleet-url=https://localhost:8080"}, "--enroll-secret and --fleet-url must be provided together")
	runAppCheckErr(t, []string{"package", "--type=deb", "--enroll-secret=foobar"}, "--enroll-secret and --fleet-url must be provided together")

	// --insecure and --fleet-certificate are mutually exclusive
	runAppCheckErr(t, []string{"package", "--type=deb", "--insecure", "--fleet-certificate=test123"}, "--insecure and --fleet-certificate may not be provided together")

	// run package tests, each should output their respective package type
	runAppForTest(t, []string{"package", "--type=deb", "--insecure"})
	runAppForTest(t, []string{"package", "--type=rpm", "--insecure"})
	runAppForTest(t, []string{"package", "--type=msi", "--insecure"})
	//runAppForTest(t, []string{"package", "--type=pkg", "--insecure"}) TODO: had a hard time getting xar installed on Ubuntu

}
