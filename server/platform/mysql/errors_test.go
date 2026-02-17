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
