package mysql

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateMysqlConnectionString_IAMDefaultsToRDSMysqlTLS(t *testing.T) {
	t.Parallel()

	dsn := generateMysqlConnectionString(MysqlConfig{
		Protocol: "tcp",
		Address:  "db.example.com:3306",
		Username: "fleet_iam",
		Database: "fleet",
		Region:   "us-east-2",
	})

	params := dsnParams(t, dsn)
	require.Equal(t, "true", params.Get("allowCleartextPasswords"))
	require.Equal(t, "rdsmysql", params.Get("tls"))
}

func TestGenerateMysqlConnectionString_IAMWithCustomTLSConfig(t *testing.T) {
	t.Parallel()

	dsn := generateMysqlConnectionString(MysqlConfig{
		Protocol:  "tcp",
		Address:   "db.example.com:3306",
		Username:  "fleet_iam",
		Database:  "fleet",
		Region:    "us-east-2",
		TLSConfig: "custom",
	})

	params := dsnParams(t, dsn)
	require.Equal(t, "true", params.Get("allowCleartextPasswords"))
	require.Equal(t, "custom", params.Get("tls"))
}

func TestGenerateMysqlConnectionString_NonIAMWithCustomTLSConfig(t *testing.T) {
	t.Parallel()

	dsn := generateMysqlConnectionString(MysqlConfig{
		Protocol:  "tcp",
		Address:   "db.example.com:3306",
		Username:  "fleet",
		Password:  "some-password",
		Database:  "fleet",
		TLSConfig: "custom",
	})

	params := dsnParams(t, dsn)
	require.Empty(t, params.Get("allowCleartextPasswords"))
	require.Equal(t, "custom", params.Get("tls"))
}

func dsnParams(t *testing.T, dsn string) url.Values {
	t.Helper()

	parts := strings.SplitN(dsn, "?", 2)
	require.Len(t, parts, 2, "dsn has no query string: %s", dsn)

	params, err := url.ParseQuery(parts[1])
	require.NoError(t, err)

	return params
}
