package main

import (
	"math/rand"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kolide/kolide-ose/cli"
	_ "github.com/kolide/kolide-ose/server/datastore/mysql/migrations"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	cli.Launch()
}
