package main

import "testing"

func TestPackage(t *testing.T) {

	runAppCheckErr(t, []string{"package", "deb"}, "Required flag \"type\" not set")
	runAppCheckErr(t, []string{"package", "--type=deb", "--fleet-url=https://localhost:8080"}, "--enroll-secret and --fleet-url must be provided together")
	runAppCheckErr(t, []string{"package", "--type=deb", "--enroll-secret=foobar"}, "--enroll-secret and --fleet-url must be provided together")
	runAppCheckErr(t, []string{"package", "--type=deb", "--insecure", "--fleet-certificate=test123"}, "--insecure and --fleet-certificate may not be provided together")

	runAppForTest(t, []string{"package", "--type=deb", "--insecure"})

}
