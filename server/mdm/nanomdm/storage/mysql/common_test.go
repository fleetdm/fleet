//go:build integration
// +build integration

package mysql

import "flag"

var flDSN = flag.String("dsn", "", "DSN of test MySQL instance")
