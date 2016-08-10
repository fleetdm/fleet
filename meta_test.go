package main

import (
	"fmt"
	"os/exec"
	"testing"
)

// goSourceDirectories declares all Go subpackages of this package
var goSourceDirectories = []string{
	"app",
	"config",
	"errors",
	"sessions",
}

var goSourceDirectoriesCount int

func init() {
	goSourceDirectoriesCount = len(goSourceDirectories)
}

// Run go vet on all Go subpackages in parallel
func TestGoVet(t *testing.T) {
	t.Parallel()
	done := make(chan bool, goSourceDirectoriesCount)
	for _, dir := range goSourceDirectories {
		go func(dir string) {
			output, err := exec.Command(
				"go",
				"vet",
				fmt.Sprintf("./%s", dir),
			).Output()
			if err != nil {
				t.Log(string(output))
				t.Error(err.Error())
			}
			done <- true
		}(dir)
	}
	for i := 1; i <= goSourceDirectoriesCount; i++ {
		<-done
	}
}

// Run go test and generate test coverage reports on all Go subpackages in parallel
func TestGoTest(t *testing.T) {
	t.Parallel()
	done := make(chan bool, goSourceDirectoriesCount)
	for _, dir := range goSourceDirectories {
		go func(dir string) {
			output, err := exec.Command(
				"go",
				"test",
				fmt.Sprintf("-coverprofile=./%s/%s.cover", dir, dir),
				"-v",
				fmt.Sprintf("./%s", dir),
			).Output()
			t.Log(string(output))
			if err != nil {
				t.Error(err.Error())
			}
			done <- true
		}(dir)
	}
	for i := 1; i <= goSourceDirectoriesCount; i++ {
		<-done
	}
}
