package main

import (
	"math/rand"
	"time"

	"github.com/kolide/kolide-ose/cli"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	cli.Launch()
}
