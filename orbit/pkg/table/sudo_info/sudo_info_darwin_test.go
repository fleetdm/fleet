//go:build darwin
// +build darwin

package sudo_info

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSudoVOutput(t *testing.T) {
	const sample = `Sudo version 1.9.5p2
Configure options: --with-password-timeout=0 --disable-setreuid --with-env-editor --with-pam --with-libraries=bsm --with-noexec=no --sysconfdir=/private/etc --without-lecture --enable-static-sudoers --with-rundir=/var/db/sudo
Sudoers policy plugin version 1.9.5p2
Sudoers file grammar version 48

Sudoers path: /etc/sudoers
Authentication methods: 'pam'
Syslog facility if syslog is being used for logging: authpriv
Syslog priority to use when user authenticates successfully: notice
Syslog priority to use when user authenticates unsuccessfully: alert
Send mail if the user is not in sudoers
Lecture user the first time they run sudo
File containing the sudo lecture: /etc/sudo_lecture
Require users to authenticate by default
Root may run sudo
Allow some information gathering to give useful error messages
Visudo will honor the EDITOR environment variable
Set the LOGNAME and USER environment variables
Length at which to wrap log file lines (0 for no wrap): 80
Authentication timestamp timeout: 5.0 minutes
Password prompt timeout: 0.0 minutes
Number of tries to enter a password: 3
Umask to use or 0777 to use user's: 022
Path to mail program: /usr/sbin/sendmail
Flags for mail program: -t
Address to send mail to: root
Subject line for mail messages: *** SECURITY information for %h ***
Incorrect password message: Sorry, try again.
Path to lecture status dir: /var/db/sudo/lectured
Path to authentication timestamp dir: /var/db/sudo/ts
Default password prompt: Password: 
Default user to run commands as: root
Path to the editor for use by visudo: /usr/bin/vi
When to require a password for 'list' pseudocommand: any
When to require a password for 'verify' pseudocommand: all
File descriptors >= 3 will be closed before executing a command
Reset the environment to a default set of variables
Environment variables to check for safety:
        TZ
        TERM
        LINGUAS
        LC_*
        LANGUAGE
        LANG
        COLORTERM
Environment variables to remove:
        *=()*
        RUBYOPT
        RUBYLIB
        PYTHONUSERBASE
        PYTHONINSPECT
        PYTHONPATH
        PYTHONHOME
        TMPPREFIX
        ZDOTDIR
        READNULLCMD
        NULLCMD
        FPATH
        PERL5DB
        PERL5OPT
        PERL5LIB
        PERLLIB
        PERLIO_DEBUG
        JAVA_TOOL_OPTIONS
        SHELLOPTS
        BASHOPTS
        GLOBIGNORE
        PS4
        BASH_ENV
        ENV
        TERMCAP
        TERMPATH
        TERMINFO_DIRS
        TERMINFO
        DYLD_*
        _RLD*
        LD_*
        PATH_LOCALE
        NLSPATH
        HOSTALIASES
        RES_OPTIONS
        LOCALDOMAIN
        CDPATH
        IFS
Environment variables to preserve:
        MAIL
        HOME
        VISUAL
        EDITOR
        TZ
        SSH_AUTH_SOCK
        LSCOLORS
        COLUMNS
        LINES
        LC_TIME
        LC_NUMERIC
        LC_MONETARY
        LC_MESSAGES
        LC_CTYPE
        LC_COLLATE
        LC_ALL
        LANGUAGE
        LANG
        CHARSET
        __CF_USER_TEXT_ENCODING
        COLORTERM
        COLORFGBG
        BLOCKSIZE
        XAUTHORIZATION
        XAUTHORITY
        PS2
        PS1
        PATH
        LS_COLORS
        KRB5CCNAME
        HOSTNAME
        DISPLAY
        COLORS
Locale to use while parsing sudoers: C
Compress I/O logs using zlib
Directory in which to store input/output logs: /var/log/sudo-io
File in which to store the input/output log: %{seq}
Add an entry to the utmp/utmpx file when allocating a pty
PAM service name to use: sudo
PAM service name to use for login shells: sudo
Attempt to establish PAM credentials for the target user
Create a new PAM session for the command to run in
Perform PAM account validation management
Enable sudoers netgroup support
Check parent directories for writability when editing files with sudoedit
Allow commands to be run even if sudo cannot write to the audit log
Allow commands to be run even if sudo cannot write to the log file
Log entries larger than this value will be split into multiple syslog messages: 960
File mode to use for the I/O log files: 0600
Execute commands by file descriptor instead of by path: digest_only
Type of authentication timestamp record: tty
Ignore case when matching user names
Ignore case when matching group names
Log when a command is allowed by sudoers
Log when a command is denied by sudoers
Sudo log server timeout in seconds: 30
Enable SO_KEEPALIVE socket option on the socket connected to the logserver
Verify that the log server's certificate is valid
Set the pam remote user to the user running sudo
The format of logs to produce: sudo

Local IP address and netmask pairs:
        fe80::aede:48ff:fe00:1122/ffff:ffff:ffff:ffff::
        fe80::142f:6d87:1e52:591d/ffff:ffff:ffff:ffff::
        192.168.0.230/255.255.255.0
        fe80::70f9:67ff:fe50:983f/ffff:ffff:ffff:ffff::
        fe80::70f9:67ff:fe50:983f/ffff:ffff:ffff:ffff::
        fe80::8372:c8a:ecf8:40b5/ffff:ffff:ffff:ffff::
        fe80::612f:3c9d:f33a:9e7d/ffff:ffff:ffff:ffff::
        fe80::ce81:b1c:bd2c:69e/ffff:ffff:ffff:ffff::
        192.168.103.1/255.255.255.0
        fe80::682f:67ff:fee8:7464/ffff:ffff:ffff:ffff::
        172.16.132.1/255.255.255.0
        fe80::682f:67ff:fee8:7465/ffff:ffff:ffff:ffff::

Sudoers I/O plugin version 1.9.5p2
Sudoers audit plugin version 1.9.5p2`
	result := parseSudoVOutput([]byte(sample))
	_, err := json.Marshal(result)
	require.NoError(t, err)

	// First line.
	require.Contains(t, result, "Sudo version 1.9.5p2")
	require.Nil(t, result["Sudo version 1.9.5p2"])

	// Key without value.
	require.Contains(t, result, "Compress I/O logs using zlib")
	require.Nil(t, result["Compress I/O logs using zlib"])

	// Key and value pairs defined in one line.
	require.Equal(t, "C", result["Locale to use while parsing sudoers"])
	require.Equal(t, "tty", result["Type of authentication timestamp record"])
	require.Equal(t, "5.0 minutes", result["Authentication timestamp timeout"])

	// Last line.
	require.Contains(t, result, "Sudoers audit plugin version 1.9.5p2")
	require.Nil(t, result["Sudoers audit plugin version 1.9.5p2"])

	// Key and value paris defined in multiple lines
	v := result["Local IP address and netmask pairs"]
	require.NotNil(t, v)
	localIPAddrAndNetmaskPairs, ok := v.([]string)
	require.True(t, ok)
	require.Len(t, localIPAddrAndNetmaskPairs, 12)
	require.Contains(t, localIPAddrAndNetmaskPairs, "fe80::aede:48ff:fe00:1122/ffff:ffff:ffff:ffff::")
	require.Contains(t, localIPAddrAndNetmaskPairs, "192.168.103.1/255.255.255.0")
	require.Contains(t, localIPAddrAndNetmaskPairs, "fe80::682f:67ff:fee8:7465/ffff:ffff:ffff:ffff::")
	v = result["Environment variables to preserve"]
	require.NotNil(t, v)
	envVariablesToPreserve, ok := v.([]string)
	require.True(t, ok)
	require.Len(t, envVariablesToPreserve, 33)
	require.Contains(t, envVariablesToPreserve, "MAIL")
	require.Contains(t, envVariablesToPreserve, "COLORS")

	// Line with multiple colons:
	require.Equal(t, "Password: ", result["Default password prompt"])
}
