package db

import (
	"reflect"
	"testing"
	"time"

	"github.com/vulsio/goval-dictionary/config"
	"github.com/vulsio/goval-dictionary/models"
)

func Test_fileterPacksByArch(t *testing.T) {
	type args struct {
		packs []models.Package
		arch  string
	}
	tests := []struct {
		in       args
		expected []models.Package
	}{
		{
			in: args{
				packs: []models.Package{
					{
						Name: "name-x86_64",
						Arch: "x86_64",
					},
					{
						Name: "name-i386",
						Arch: "i386",
					},
				},
				arch: "x86_64",
			},
			expected: []models.Package{{
				Name: "name-x86_64",
				Arch: "x86_64",
			}},
		},
		{
			in: args{
				packs: []models.Package{
					{
						Name: "name-x86_64",
						Arch: "x86_64",
					},
					{
						Name: "name-i386",
						Arch: "i386",
					},
				},
				arch: "",
			},
			expected: []models.Package{
				{
					Name: "name-x86_64",
					Arch: "x86_64",
				},
				{
					Name: "name-i386",
					Arch: "i386",
				},
			},
		},
	}

	for i, tt := range tests {
		if aout := fileterPacksByArch(tt.in.packs, tt.in.arch); !reflect.DeepEqual(aout, tt.expected) {
			t.Errorf("[%d] fileterPacksByArch expected: %#v\n  actual: %#v\n", i, tt.expected, aout)
		}
	}
}

func Test_restoreDefinition(t *testing.T) {
	type args struct {
		defstr  string
		family  string
		version string
		arch    string
	}
	tests := []struct {
		in       args
		expected models.Definition
	}{
		{
			in: args{
				defstr:  "{\"DefinitionID\":\"oval:com.ubuntu.focal:def:201606340000000\",\"Title\":\"CVE-2016-0634 on Ubuntu 20.04 (focal) - low.\",\"Description\":\"The expansion of '\\\\h' in the prompt string in bash 4.3 allows remote authenticated users to execute arbitrary code via shell metacharacters placed in 'hostname' of a machine.\",\"Advisory\":{\"Severity\":\"Low\",\"Cves\":[{\"CveID\":\"CVE-2016-0634\",\"Cvss2\":\"\",\"Cvss3\":\"\",\"Cwe\":\"\",\"Impact\":\"\",\"Href\":\"https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-0634\",\"Public\":\"\"}],\"Bugzillas\":[],\"AffectedCPEList\":[],\"Issued\":\"1000-01-01T00:00:00Z\",\"Updated\":\"1000-01-01T00:00:00Z\"},\"Debian\":null,\"AffectedPacks\":[{\"Name\":\"bash\",\"Version\":\"4.4-2ubuntu1\",\"Arch\":\"\",\"NotFixedYet\":false,\"ModularityLabel\":\"\"}],\"References\":[{\"Source\":\"CVE\",\"RefID\":\"CVE-2016-0634\",\"RefURL\":\"https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-0634\"},{\"Source\":\"Ref\",\"RefID\":\"\",\"RefURL\":\"http://people.canonical.com/~ubuntu-security/cve/2016/CVE-2016-0634.html\"},{\"Source\":\"Ref\",\"RefID\":\"\",\"RefURL\":\"http://www.openwall.com/lists/oss-security/2016/09/16/8\"},{\"Source\":\"Ref\",\"RefID\":\"\",\"RefURL\":\"https://ubuntu.com/security/notices/USN-3294-1\"},{\"Source\":\"Bug\",\"RefID\":\"\",\"RefURL\":\"https://bugs.launchpad.net/ubuntu/+source/bash/+bug/1507025\"}]}",
				family:  config.Ubuntu,
				version: "20",
				arch:    "",
			},
			expected: models.Definition{
				DefinitionID: "oval:com.ubuntu.focal:def:201606340000000",
				Title:        "CVE-2016-0634 on Ubuntu 20.04 (focal) - low.",
				Description:  "The expansion of '\\h' in the prompt string in bash 4.3 allows remote authenticated users to execute arbitrary code via shell metacharacters placed in 'hostname' of a machine.",
				Advisory: models.Advisory{
					Severity: "Low",
					Cves: []models.Cve{
						{
							CveID:  "CVE-2016-0634",
							Cvss2:  "",
							Cvss3:  "",
							Cwe:    "",
							Impact: "",
							Href:   "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-0634",
							Public: "",
						},
					},
					Bugzillas:       []models.Bugzilla{},
					AffectedCPEList: []models.Cpe{},
					Issued:          time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC),
					Updated:         time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
				Debian: nil,
				AffectedPacks: []models.Package{
					{
						Name:            "bash",
						Version:         "4.4-2ubuntu1",
						Arch:            "",
						NotFixedYet:     false,
						ModularityLabel: "",
					},
				},
				References: []models.Reference{
					{
						Source: "CVE",
						RefID:  "CVE-2016-0634",
						RefURL: "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-0634",
					},
					{
						Source: "Ref",
						RefID:  "",
						RefURL: "http://people.canonical.com/~ubuntu-security/cve/2016/CVE-2016-0634.html",
					},
					{
						Source: "Ref",
						RefID:  "",
						RefURL: "http://www.openwall.com/lists/oss-security/2016/09/16/8",
					},
					{
						Source: "Ref",
						RefID:  "",
						RefURL: "https://ubuntu.com/security/notices/USN-3294-1",
					},
					{
						Source: "Bug",
						RefID:  "",
						RefURL: "https://bugs.launchpad.net/ubuntu/+source/bash/+bug/1507025",
					},
				},
			},
		},
		{
			in: args{
				defstr:  "{\"DefinitionID\":\"oval:com.redhat.rhsa:def:20201113\",\"Title\":\"RHSA-2020:1113: bash security update (Moderate)\",\"Description\":\"The bash packages provide Bash (Bourne-again shell), which is the default shell for Red Hat Enterprise Linux.\\n\\nSecurity Fix(es):\\n\\n* bash: BASH_CMD is writable in restricted bash shells (CVE-2019-9924)\\n\\nFor more details about the security issue(s), including the impact, a CVSS score, acknowledgments, and other related information, refer to the CVE page(s) listed in the References section.\\n\\nAdditional Changes:\\n\\nFor detailed information on changes in this release, see the Red Hat Enterprise Linux 7.8 Release Notes linked from the References section.\",\"Advisory\":{\"Severity\":\"Moderate\",\"Cves\":[{\"CveID\":\"CVE-2019-9924\",\"Cvss2\":\"\",\"Cvss3\":\"7.8/CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H\",\"Cwe\":\"CWE-138\",\"Impact\":\"moderate\",\"Href\":\"https://access.redhat.com/security/cve/CVE-2019-9924\",\"Public\":\"20190307\"}],\"Bugzillas\":[{\"BugzillaID\":\"1691774\",\"URL\":\"https://bugzilla.redhat.com/1691774\",\"Title\":\"CVE-2019-9924 bash: BASH_CMD is writable in restricted bash shells\"}],\"AffectedCPEList\":[{\"Cpe\":\"cpe:/o:redhat:enterprise_linux:7\"},{\"Cpe\":\"cpe:/o:redhat:enterprise_linux:7::client\"},{\"Cpe\":\"cpe:/o:redhat:enterprise_linux:7::computenode\"},{\"Cpe\":\"cpe:/o:redhat:enterprise_linux:7::server\"},{\"Cpe\":\"cpe:/o:redhat:enterprise_linux:7::workstation\"}],\"Issued\":\"2020-03-31T00:00:00Z\",\"Updated\":\"2020-03-31T00:00:00Z\"},\"Debian\":null,\"AffectedPacks\":[{\"Name\":\"bash\",\"Version\":\"0:4.2.46-34.el7\",\"Arch\":\"\",\"NotFixedYet\":false,\"ModularityLabel\":\"\"},{\"Name\":\"bash-doc\",\"Version\":\"0:4.2.46-34.el7\",\"Arch\":\"\",\"NotFixedYet\":false,\"ModularityLabel\":\"\"}],\"References\":[{\"Source\":\"RHSA\",\"RefID\":\"RHSA-2020:1113\",\"RefURL\":\"https://access.redhat.com/errata/RHSA-2020:1113\"},{\"Source\":\"CVE\",\"RefID\":\"CVE-2019-9924\",\"RefURL\":\"https://access.redhat.com/security/cve/CVE-2019-9924\"}]}",
				family:  config.RedHat,
				version: "7",
				arch:    "",
			},
			expected: models.Definition{
				DefinitionID: "oval:com.redhat.rhsa:def:20201113",
				Title:        "RHSA-2020:1113: bash security update (Moderate)",
				Description:  "The bash packages provide Bash (Bourne-again shell), which is the default shell for Red Hat Enterprise Linux.\n\nSecurity Fix(es):\n\n* bash: BASH_CMD is writable in restricted bash shells (CVE-2019-9924)\n\nFor more details about the security issue(s), including the impact, a CVSS score, acknowledgments, and other related information, refer to the CVE page(s) listed in the References section.\n\nAdditional Changes:\n\nFor detailed information on changes in this release, see the Red Hat Enterprise Linux 7.8 Release Notes linked from the References section.",
				Advisory: models.Advisory{
					Severity: "Moderate",
					Cves: []models.Cve{
						{
							CveID:  "CVE-2019-9924",
							Cvss2:  "",
							Cvss3:  "7.8/CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H",
							Cwe:    "CWE-138",
							Impact: "moderate",
							Href:   "https://access.redhat.com/security/cve/CVE-2019-9924",
							Public: "20190307",
						},
					},
					Bugzillas: []models.Bugzilla{
						{
							BugzillaID: "1691774",
							URL:        "https://bugzilla.redhat.com/1691774",
							Title:      "CVE-2019-9924 bash: BASH_CMD is writable in restricted bash shells",
						},
					},
					AffectedCPEList: []models.Cpe{
						{
							Cpe: "cpe:/o:redhat:enterprise_linux:7",
						},
						{
							Cpe: "cpe:/o:redhat:enterprise_linux:7::client",
						},
						{
							Cpe: "cpe:/o:redhat:enterprise_linux:7::computenode",
						},
						{
							Cpe: "cpe:/o:redhat:enterprise_linux:7::server",
						},
						{
							Cpe: "cpe:/o:redhat:enterprise_linux:7::workstation",
						},
					},
					Issued:  time.Date(2020, time.March, 31, 0, 0, 0, 0, time.UTC),
					Updated: time.Date(2020, time.March, 31, 0, 0, 0, 0, time.UTC),
				},
				Debian: nil,
				AffectedPacks: []models.Package{
					{
						Name:            "bash",
						Version:         "0:4.2.46-34.el7",
						Arch:            "",
						NotFixedYet:     false,
						ModularityLabel: "",
					},
					{
						Name:            "bash-doc",
						Version:         "0:4.2.46-34.el7",
						Arch:            "",
						NotFixedYet:     false,
						ModularityLabel: "",
					},
				},
				References: []models.Reference{
					{
						Source: "RHSA",
						RefID:  "RHSA-2020:1113",
						RefURL: "https://access.redhat.com/errata/RHSA-2020:1113",
					},
					{
						Source: "CVE",
						RefID:  "CVE-2019-9924",
						RefURL: "https://access.redhat.com/security/cve/CVE-2019-9924",
					},
				},
			},
		},
		{in: args{
			defstr:  "{\"DefinitionID\":\"def-ALAS2-2020-1503\",\"Title\":\"ALAS2-2020-1503\",\"Description\":\"Package updates are available for Amazon Linux 2 that fix the following vulnerabilities:\\nCVE-2019-9924:\\n\\trbash in Bash before 4.4-beta2 did not prevent the shell user from modifying BASH_CMDS, thus allowing the user to execute any command with the permissions of the shell.\\n1691774: CVE-2019-9924 bash: BASH_CMD is writable in restricted bash shells\\n\",\"Advisory\":{\"Severity\":\"medium\",\"Cves\":[{\"CveID\":\"CVE-2019-9924\",\"Cvss2\":\"\",\"Cvss3\":\"\",\"Cwe\":\"\",\"Impact\":\"\",\"Href\":\"https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2019-9924\",\"Public\":\"\"}],\"Bugzillas\":[],\"AffectedCPEList\":[],\"Issued\":\"1000-01-01T00:00:00Z\",\"Updated\":\"2020-10-22T22:39:00Z\"},\"Debian\":null,\"AffectedPacks\":[{\"Name\":\"bash\",\"Version\":\"0:4.2.46-34.amzn2\",\"Arch\":\"aarch64\",\"NotFixedYet\":false,\"ModularityLabel\":\"\"},{\"Name\":\"bash-doc\",\"Version\":\"0:4.2.46-34.amzn2\",\"Arch\":\"aarch64\",\"NotFixedYet\":false,\"ModularityLabel\":\"\"},{\"Name\":\"bash-debuginfo\",\"Version\":\"0:4.2.46-34.amzn2\",\"Arch\":\"aarch64\",\"NotFixedYet\":false,\"ModularityLabel\":\"\"},{\"Name\":\"bash\",\"Version\":\"0:4.2.46-34.amzn2\",\"Arch\":\"x86_64\",\"NotFixedYet\":false,\"ModularityLabel\":\"\"},{\"Name\":\"bash-doc\",\"Version\":\"0:4.2.46-34.amzn2\",\"Arch\":\"x86_64\",\"NotFixedYet\":false,\"ModularityLabel\":\"\"},{\"Name\":\"bash-debuginfo\",\"Version\":\"0:4.2.46-34.amzn2\",\"Arch\":\"x86_64\",\"NotFixedYet\":false,\"ModularityLabel\":\"\"},{\"Name\":\"bash\",\"Version\":\"0:4.2.46-34.amzn2\",\"Arch\":\"i686\",\"NotFixedYet\":false,\"ModularityLabel\":\"\"},{\"Name\":\"bash-doc\",\"Version\":\"0:4.2.46-34.amzn2\",\"Arch\":\"i686\",\"NotFixedYet\":false,\"ModularityLabel\":\"\"},{\"Name\":\"bash-debuginfo\",\"Version\":\"0:4.2.46-34.amzn2\",\"Arch\":\"i686\",\"NotFixedYet\":false,\"ModularityLabel\":\"\"}],\"References\":[{\"Source\":\"cve\",\"RefID\":\"CVE-2019-9924\",\"RefURL\":\"http://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2019-9924\"}]}",
			family:  config.Amazon,
			version: "2",
			arch:    "x86_64",
		},
			expected: models.Definition{
				DefinitionID: "def-ALAS2-2020-1503",
				Title:        "ALAS2-2020-1503",
				Description:  "Package updates are available for Amazon Linux 2 that fix the following vulnerabilities:\nCVE-2019-9924:\n\trbash in Bash before 4.4-beta2 did not prevent the shell user from modifying BASH_CMDS, thus allowing the user to execute any command with the permissions of the shell.\n1691774: CVE-2019-9924 bash: BASH_CMD is writable in restricted bash shells\n",
				Advisory: models.Advisory{
					Severity: "medium",
					Cves: []models.Cve{
						{
							CveID:  "CVE-2019-9924",
							Cvss2:  "",
							Cvss3:  "",
							Cwe:    "",
							Impact: "",
							Href:   "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2019-9924",
							Public: "",
						},
					},
					Bugzillas:       []models.Bugzilla{},
					AffectedCPEList: []models.Cpe{},
					Issued:          time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC),
					Updated:         time.Date(2020, time.October, 22, 22, 39, 0, 0, time.UTC),
				},
				Debian: nil,
				AffectedPacks: []models.Package{
					{
						Name:            "bash",
						Version:         "0:4.2.46-34.amzn2",
						Arch:            "x86_64",
						NotFixedYet:     false,
						ModularityLabel: "",
					},
					{
						Name:            "bash-doc",
						Version:         "0:4.2.46-34.amzn2",
						Arch:            "x86_64",
						NotFixedYet:     false,
						ModularityLabel: "",
					},
					{
						Name:            "bash-debuginfo",
						Version:         "0:4.2.46-34.amzn2",
						Arch:            "x86_64",
						NotFixedYet:     false,
						ModularityLabel: "",
					},
				},
				References: []models.Reference{
					{
						Source: "cve",
						RefID:  "CVE-2019-9924",
						RefURL: "http://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2019-9924",
					},
				},
			},
		},
	}

	for i, tt := range tests {
		if aout, _ := restoreDefinition(tt.in.defstr, tt.in.family, tt.in.version, tt.in.arch); !reflect.DeepEqual(aout, tt.expected) {
			t.Errorf("[%d] restoreDefinition expected: %#v\n  actual: %#v\n", i, tt.expected, aout)
		}
	}
}
