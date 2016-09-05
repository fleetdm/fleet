package main

import (
	"math/rand"
	"time"

	"github.com/kolide/kolide-ose/cli"
	_ "github.com/kolide/kolide-ose/config"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	cli.Launch()
}
