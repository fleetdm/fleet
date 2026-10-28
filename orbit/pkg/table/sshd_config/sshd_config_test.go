package sshd_config

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	output := "port 22\n" +
		"listenaddress 0.0.0.0:22\n" +
		"listenaddress [::]:22\n" +
		"PermitRootLogin prohibit-password\n" +
		"\n" +
		"subsystem sftp /usr/libexec/sftp-server\r\n" +
		"persourcepenalties crash:90.000000 authfail:5.000000\n" +
		"bannerkeywordonly\n"

	assert.Equal(t, []configEntry{
		{"port", "22"},
		{"listenaddress", "0.0.0.0:22"},
		{"listenaddress", "[::]:22"},
		{"permitrootlogin", "prohibit-password"},
		{"subsystem", "sftp /usr/libexec/sftp-server"},
		{"persourcepenalties", "crash:90.000000 authfail:5.000000"},
		{"bannerkeywordonly", ""},
	}, parseConfig(output))

	assert.Empty(t, parseConfig(""))
}

func TestSanitizeDetail(t *testing.T) {
	assert.Equal(t, "a b c", sanitizeDetail(" a\nb\r\n\tc "))
	assert.Equal(t, "?[31mred ?bell", sanitizeDetail("\x1b[31mred \abell"))
	assert.Equal(t, "Invalid test mode specification usér", sanitizeDetail("Invalid test mode specification usér\n"))

	long := sanitizeDetail(strings.Repeat("é", maxErrorDetail+100))
	assert.Equal(t, strings.Repeat("é", maxErrorDetail)+"...", long)
	assert.True(t, utf8.ValidString(long))
}

func TestUnsupportedFlag(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{"bsd libc", "/usr/sbin/sshd: illegal option -- G\n", true},
		{"glibc", "sshd: invalid option -- 'G'\n", true},
		{"openssh getopt", "unknown option -- G\r\n", true},
		{
			"openssh getopt with usage (Debian 12)",
			"unknown option -- G\r\n" +
				"OpenSSH_9.2p1 Debian-2+deb12u10, OpenSSL 3.0.22 25 Aug 2026\n" +
				"usage: sshd [-46DdeiqTtV] [-C connection_spec] [-c host_cert_file]\n",
			true,
		},
		{"musl", "sshd: unrecognized option: G\n", true},
		{"usage only", "usage: sshd [-46DdeiqTt] [-C connection_spec]\n", true},
		{"bad config", "/etc/ssh/sshd_config: line 12: Bad configuration option: Foo\n", false},
		{"unsupported directive", "/etc/ssh/sshd_config line 3: Unsupported option UsePrivilegeSeparation\n", false},
		{"bad spec", "Invalid test mode specification foo\n", false},
		{"other flag", "sshd: illegal option -- Z\n", false},
		{"missing keys", "sshd: no hostkeys available -- exiting.\n", false},
		{"echoed spec with usage text", "Invalid test mode specification usage: sshd\n", false},
		{"echoed spec with getopt text", "Invalid test mode specification sshd: illegal option -- G\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, unsupportedFlag.MatchString(tt.stderr))
		})
	}
}

func TestConnectionSpecs(t *testing.T) {
	tests := []struct {
		name string
		ctx  table.QueryContext
		want []string
	}{
		{"no constraints", table.QueryContext{}, nil},
		{
			"equals and in, deduplicated",
			specQueryContext(
				table.Constraint{Operator: table.OperatorEquals, Expression: "user=root"},
				table.Constraint{Operator: table.OperatorEquals, Expression: "user=alice,addr=10.0.0.1"},
				table.Constraint{Operator: table.OperatorEquals, Expression: "user=root"},
			),
			[]string{"user=root", "user=alice,addr=10.0.0.1"},
		},
		{
			"empty spec is the global configuration",
			specQueryContext(table.Constraint{Operator: table.OperatorEquals, Expression: ""}),
			[]string{""},
		},
		{
			"non-equality ignored",
			specQueryContext(table.Constraint{Operator: table.OperatorLike, Expression: "user=%"}),
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, connectionSpecs(tt.ctx))
		})
	}
}

func specQueryContext(constraints ...table.Constraint) table.QueryContext {
	return table.QueryContext{Constraints: map[string]table.ConstraintList{
		"connection_spec": {Affinity: table.ColumnTypeText, Constraints: constraints},
	}}
}
