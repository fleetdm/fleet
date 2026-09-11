package mysql

import (
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

func TestIsReadOnlyError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "unrelated MySQL error",
			err:  &mysql.MySQLError{Number: 1045, Message: "Access denied"},
			want: false,
		},
		{
			name: "error 1792 read-only transaction",
			err:  &mysql.MySQLError{Number: 1792, Message: "Cannot execute statement in a READ ONLY transaction."},
			want: true,
		},
		{
			name: "error 1290 option prevents statement",
			err:  &mysql.MySQLError{Number: 1290, Message: "The MySQL server is running with the --read-only option"},
			want: true,
		},
		{
			name: "error 1836 read-only mode",
			err:  &mysql.MySQLError{Number: 1836, Message: "Running in read-only mode"},
			want: true,
		},
		{
			name: "wrapped read-only error",
			err:  fmt.Errorf("transaction failed: %w", &mysql.MySQLError{Number: 1792, Message: "read only"}),
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, IsReadOnlyError(tc.err))
		})
	}
}

func TestNotFoundErrorMessage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  *NotFoundError
		want string
	}{
		{
			name: "resource type only",
			err:  NotFound("CertificateTemplate"),
			want: "CertificateTemplate was not found in the datastore",
		},
		{
			name: "id only",
			err:  NotFound("CertificateTemplate").WithID(4),
			want: "CertificateTemplate 4 was not found in the datastore",
		},
		{
			name: "fleet id only",
			err:  NotFound("BootstrapPackage").WithFleetID(2),
			want: "BootstrapPackage for fleet 2 was not found in the datastore",
		},
		{
			name: "id and fleet id",
			err:  NotFound("CertificateTemplate").WithID(4).WithFleetID(2),
			want: "CertificateTemplate 4 for fleet 2 was not found in the datastore",
		},
		{
			name: "id and no team fleet id",
			err:  NotFound("CertificateTemplate").WithID(4).WithFleetID(0),
			want: "CertificateTemplate 4 was not found in the datastore",
		},
		{
			name: "name only",
			err:  NotFound("Team").WithName("Yellow jackets"),
			want: "Team Yellow jackets was not found in the datastore",
		},
		{
			name: "message only",
			err:  NotFound("Host").WithMessage("uuid abc"),
			want: "Host uuid abc was not found in the datastore",
		},
		{
			name: "id wins over name and message",
			err:  NotFound("Host").WithName("foo").WithMessage("bar").WithID(7),
			want: "Host 7 was not found in the datastore",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.err.Error())
			assert.True(t, tc.err.IsNotFound())
		})
	}
}
