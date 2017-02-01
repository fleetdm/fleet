package main

import (
	"math/rand"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kolide/kolide/cli"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	cli.Launch()
}
