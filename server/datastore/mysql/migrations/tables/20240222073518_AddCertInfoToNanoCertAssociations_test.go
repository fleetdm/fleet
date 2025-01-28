package tables

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20240222073518(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec("INSERT INTO scep_serials (serial) VALUES (1), (2)")
	require.NoError(t, err)

	threeDaysAgo := time.Now().UTC().Add(-72 * time.Hour).Truncate(time.Second)
	_, err = db.Exec(`
        INSERT INTO scep_certificates (serial, not_valid_before, not_valid_after, certificate_pem)
        VALUES (?, ?, ?, ?), (?, ?, ?, ?)`,
		// not_valid_* values don't really matter as the migration
		// takes the value from the parsed cert.
		1, threeDaysAgo, threeDaysAgo, dummyCert1,
		2, threeDaysAgo, threeDaysAgo, dummyCert2,
	)
	require.NoError(t, err)

	sha1, sha2 := "4c51d40f56f5c5e13448995d4d2fd0b6b7befef860e4e7341c355ab38031ee35", "53c2dc9ce116a1df4adfba0c556843625fd1e91f83fc89a47c3267dff9a4c4ba" // #nosec G101
	_, err = db.Exec(`
        INSERT INTO nano_cert_auth_associations (id, sha256, created_at, updated_at)
        VALUES (?, ?, ?, ?), (?, ?, ?, ?), (?, ?, ?, ?)`,
		"uuid-1", sha1, threeDaysAgo, threeDaysAgo,
		"uuid-2", sha2, threeDaysAgo, threeDaysAgo,
		// host with duplicate cert, should never happen, but we don't
		// have constraints in the db.
		"uuid-3", sha2, threeDaysAgo, threeDaysAgo,
	)
	require.NoError(t, err)

	applyNext(t, db)

	var assoc struct {
		HostUUID          string     `db:"id"`
		SHA256            string     `db:"sha256"`
		CreatedAt         time.Time  `db:"created_at"`
		UpdatedAt         time.Time  `db:"updated_at"`
		CertNotValidAfter *time.Time `db:"cert_not_valid_after"`
		RenewCommandUUID  *string    `db:"renew_command_uuid"`
	}

	selectStmt := "SELECT id, sha256, created_at, updated_at, cert_not_valid_after, renew_command_uuid FROM nano_cert_auth_associations WHERE id = ?"

	// new values are filled and timestamps preserved
	err = sqlx.Get(db, &assoc, selectStmt, "uuid-1")
	require.NoError(t, err)
	require.Equal(t, "uuid-1", assoc.HostUUID)
	require.Equal(t, sha1, assoc.SHA256)
	require.Equal(t, threeDaysAgo, assoc.CreatedAt)
	require.Equal(t, threeDaysAgo, assoc.UpdatedAt)
	require.Equal(t, "2025-02-20 19:57:24", assoc.CertNotValidAfter.Format("2006-01-02 15:04:05"))
	require.Nil(t, assoc.RenewCommandUUID)

	err = sqlx.Get(db, &assoc, selectStmt, "uuid-2")
	require.NoError(t, err)
	require.Equal(t, "uuid-2", assoc.HostUUID)
	require.Equal(t, sha2, assoc.SHA256)
	require.Equal(t, threeDaysAgo, assoc.CreatedAt)
	require.Equal(t, threeDaysAgo, assoc.UpdatedAt)
	require.Equal(t, "2025-02-20 19:57:25", assoc.CertNotValidAfter.Format("2006-01-02 15:04:05"))
	require.Nil(t, assoc.RenewCommandUUID)

	err = sqlx.Get(db, &assoc, selectStmt, "uuid-3")
	require.NoError(t, err)
	require.Equal(t, "uuid-3", assoc.HostUUID)
	require.Equal(t, sha2, assoc.SHA256)
	require.Equal(t, threeDaysAgo, assoc.CreatedAt)
	require.Equal(t, threeDaysAgo, assoc.UpdatedAt)
	require.Equal(t, "2025-02-20 19:57:25", assoc.CertNotValidAfter.Format("2006-01-02 15:04:05"))
	require.Nil(t, assoc.RenewCommandUUID)

	// creating a new association sets NULL as default values
	_, err = db.Exec(`
        INSERT INTO nano_cert_auth_associations (id, sha256)
        VALUES (?, ?)`, "uuid-4", sha1)
	require.NoError(t, err)

	err = sqlx.Get(db, &assoc, selectStmt, "uuid-4")
	require.NoError(t, err)
	require.Equal(t, "uuid-4", assoc.HostUUID)
	require.Equal(t, sha1, assoc.SHA256)
	require.Nil(t, assoc.CertNotValidAfter)
	require.Nil(t, assoc.RenewCommandUUID)
}

var dummyCert1 = []byte(`-----BEGIN CERTIFICATE-----
MIIDgDCCAmigAwIBAgIBAjANBgkqhkiG9w0BAQsFADBBMQkwBwYDVQQGEwAxEDAO
BgNVBAoTB3NjZXAtY2ExEDAOBgNVBAsTB1NDRVAgQ0ExEDAOBgNVBAMTB0ZsZWV0
RE0wHhcNMjQwMjIxMTk0NzI0WhcNMjUwMjIwMTk1NzI0WjBdMRswGQYDVQQKExJm
bGVldC1vcmdhbml6YXRpb24xPjA8BgNVBAMTNWZsZWV0LXRlc3RkZXZpY2UtN0U0
RUJENTQtNjVDNy00RkU2LThFODUtOUUyNDEwMTQ1RENEMIIBIjANBgkqhkiG9w0B
AQEFAAOCAQ8AMIIBCgKCAQEAtgu75XAA5B2iys8DIZwdf2pdzFk157vyZZTnLI4r
7whAtLV556c6hjstyXhOmkut+kfiWHWoKQgbrtBj5LTfXwDbu11FapJvYPiI/GwD
vAQ+KbV9JcoGX70vL5Qmh+M2P+Ky//cE/zDc2YvPpEk4lcR+BNMJ1SnpRqZQ7ggC
0mw62TWbnOuQM4o+1ykvDpJBJhrLxdsEVNaGZVRb0W/GRLzMZNbkQtcBxhpi0yqy
iAScF75A0uy7pRSg1Fkr612qqA2bUcPMY901t264Hn7/YyAorVQS7iEvX9DVQbVu
T4GNtU5VaDrFsWBlDjVyj2+KUUU2g4klYJLEbfIjSa62WQIDAQABo2cwZTAOBgNV
HQ8BAf8EBAMCB4AwEwYDVR0lBAwwCgYIKwYBBQUHAwIwHQYDVR0OBBYEFK9+cymm
EOOnA6EYicjk/OrJQI74MB8GA1UdIwQYMBaAFGmjhvWYNxfl4+HKLj8gWPCbXnqc
MA0GCSqGSIb3DQEBCwUAA4IBAQClD0xOhWS7Pqmiz6t0cC91sL2nAHgFtFhSQKNY
bQFGb0GIJQe0YVV1fJbDDqgHdaYXz+QwJWKfCui0ixYEPgho4SqdeNWsRgDs5EqU
chV6P/+yksXdKiu5f2wmf1T3oqgnrBxTm9bXe2ZQFR77FeeeA1AHUAOCESI3d6QF
ClvMWXXA/cutC3Wp/34M540trLGiM914whaf0Pb6Rx8HjldEn/dThWOKZDYK4MSK
4W2h3vw2aouSe46i86VtYTaDfTP5H4As+N6NunT7lK6sc3UWeWo7k3dliiywnQ4Y
AiCWE3wJMLpNwPaxdxz/grg8MLw9uPvfznZ9K9G/i5IDvI9O
-----END CERTIFICATE-----`)

var dummyCert2 = []byte(`-----BEGIN CERTIFICATE-----
MIIDgDCCAmigAwIBAgIBAzANBgkqhkiG9w0BAQsFADBBMQkwBwYDVQQGEwAxEDAO
BgNVBAoTB3NjZXAtY2ExEDAOBgNVBAsTB1NDRVAgQ0ExEDAOBgNVBAMTB0ZsZWV0
RE0wHhcNMjQwMjIxMTk0NzI1WhcNMjUwMjIwMTk1NzI1WjBdMRswGQYDVQQKExJm
bGVldC1vcmdhbml6YXRpb24xPjA8BgNVBAMTNWZsZWV0LXRlc3RkZXZpY2UtMjQw
RkI4NEQtQzFBOS00ODhCLUEzNDItQkFBOTI2NTkwOEJBMIIBIjANBgkqhkiG9w0B
AQEFAAOCAQ8AMIIBCgKCAQEA0H/BTmCHrLrYHn0CWC+V0qMVvvOjE9fE178DOU8W
x/W5FGw9Vm+kYE2Tt/dQVLDYUnEg8u1v6JCN2YErGc3eLjyUPVz28778sVQCTc7s
Ax1QTxoRjxss7KDhSArdPyEu2YzbKfefEcVqPymDxQTeTKrscgN9XTIe6uvb6qCM
3HHKQJsUb8me8Sat8RyR1q+ahR7vrj9pHCXC/nyeK2l1xmnTgz2++C47zMVzjJ7g
VduG3SV440spcd/0TbCjYvu2qe4KcK1TypAbjyo/XOBI75ZV/S8uLmFR9C1XxDvQ
1rngNjyHa24LiweOYd3MIVe+g8htsOCOB8S9hWhN8Xn1OwIDAQABo2cwZTAOBgNV
HQ8BAf8EBAMCB4AwEwYDVR0lBAwwCgYIKwYBBQUHAwIwHQYDVR0OBBYEFL3wSLk7
LWXNnzzNM4ZrIhPEL/0OMB8GA1UdIwQYMBaAFGmjhvWYNxfl4+HKLj8gWPCbXnqc
MA0GCSqGSIb3DQEBCwUAA4IBAQBXEbOh4hCbOfnRbtBUtd5s1aNd0N+E11eFJM6k
hwgOzHCgGrfG7eh/8QQ+4fYAnpyBEEz863EEqfmPY++MifLI7AI8b82EqxNVT8UK
YeFIvbtOwgKiq+YDLIzXPzRzOS6lgGB68nFNRyni4TeTCx5aaKBKfWlDNwOCdI7c
F97od8YqLp1wDG5caCKVvzLXbOvMZmdmjztKZoI+/SjPDpVsNKZrixYmijDVhZNf
Hd2ktxwNgxBx6TDAbCjwXhim2vPAg7ZoklxLHN4KS2F+ZtKDUbdR2WZyolxJh5QC
KuY7qFtlQZQFIcXnSpgXTC6tpG+oldTkz9exA4Zm5eXqTBfU
-----END CERTIFICATE-----`)
