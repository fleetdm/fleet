# goval-dictionary

This is tool to build a local copy of the OVAL. The local copy is generated in sqlite format, and the tool has a server mode for easy querying.

## Installation

### Requirements

goval-dictionary requires the following packages.

- SQLite3, MySQL (or MariaDB), PostgreSQL or Redis
- git
- gcc
- lastest version of go
    - https://golang.org/doc/install

### Install

```bash
$ mkdir -p $GOPATH/src/github.com/vulsio
$ cd $GOPATH/src/github.com/vulsio
$ git clone https://github.com/vulsio/goval-dictionary.git
$ cd goval-dictionary
$ make install
```

----

## Usage

```bash
$ goval-dictionary --help
OVAL(Open Vulnerability and Assessment Language) dictionary

Usage:
  goval-dictionary [command]

Available Commands:
  completion  generate the autocompletion script for the specified shell
  fetch       Fetch Vulnerability dictionary
  help        Help about any command
  select      Select from DB
  server      Start OVAL dictionary HTTP server
  version     Show version

Flags:
      --config string       config file (default is $HOME/.oval.yaml)
      --dbpath string       /path/to/sqlite3 or SQL connection string (default "$PWD/oval.sqlite3")
      --dbtype string       Database type to store data in (sqlite3, mysql, postgres or redis supported) (default "sqlite3")
      --debug               debug mode (default: false)
      --debug-sql           SQL debug mode
  -h, --help                help for goval-dictionary
      --http-proxy string   http://proxy-url:port (default: empty)
      --log-dir string      /path/to/log (default "/var/log/goval-dictionary")
      --log-json            output log as JSON

Use "goval-dictionary [command] --help" for more information about a command.
```

### Usage: Fetch OVAL data
```bash
$ goval-dictionary fetch --help
Fetch Vulnerability dictionary

Usage:
  goval-dictionary fetch [command]

Available Commands:
  alpine      Fetch Vulnerability dictionary from Alpine secdb
  amazon      Fetch Vulnerability dictionary from Amazon ALAS
  debian      Fetch Vulnerability dictionary from Debian
  fedora      Fetch Vulnerability dictionary from Fedora
  oracle      Fetch Vulnerability dictionary from Oracle
  redhat      Fetch Vulnerability dictionary from RedHat
  suse        Fetch Vulnerability dictionary from SUSE
  ubuntu      Fetch Vulnerability dictionary from Ubuntu

Flags:
  -h, --help          help for fetch
      --no-details    without vulnerability details

Global Flags:
      --config string       config file (default is $HOME/.oval.yaml)
      --dbpath string       /path/to/sqlite3 or SQL connection string (default "$PWD/oval.sqlite3")
      --dbtype string       Database type to store data in (sqlite3, mysql, postgres or redis supported) (default "sqlite3")
      --debug               debug mode (default: false)
      --debug-sql           SQL debug mode
      --http-proxy string   http://proxy-url:port (default: empty)
      --log-dir string      /path/to/log (default "/var/log/goval-dictionary")
      --log-json            output log as JSON

Use "goval-dictionary fetch [command] --help" for more information about a command.
```

#### Usage: Fetch OVAL data from RedHat

- [Redhat OVAL](https://www.redhat.com/security/data/oval/)

```bash
$ goval-dictionary fetch redhat 5 6 7 8 9
```

#### Usage: Fetch OVAL data from Debian

- [Debian OVAL](https://www.debian.org/security/oval/)

```bash
$ goval-dictionary fetch debian 7 8 9 10 11 12 13
```

#### Usage: Fetch OVAL data from Ubuntu

- [Ubuntu](https://security-metadata.canonical.com/oval/)
```bash
$ goval-dictionary fetch ubuntu 14.04 16.04 18.04 20.04 22.04 24.04 24.10 25.04
```

#### Usage: Fetch OVAL data from SUSE

- [SUSE](http://ftp.suse.com/pub/projects/security/oval/)

```bash
$ goval-dictionary fetch suse --help
Fetch Vulnerability dictionary from SUSE

Usage:
  goval-dictionary fetch suse [flags]

Flags:
  -h, --help               help for suse
      --suse-type string   Fetch SUSE Type (default "opensuse-leap")

Global Flags:
      --config string       config file (default is $HOME/.oval.yaml)
      --dbpath string       /path/to/sqlite3 or SQL connection string (default "/$PWD/oval.sqlite3")
      --dbtype string       Database type to store data in (sqlite3, mysql, postgres or redis supported) (default "sqlite3")
      --debug               debug mode (default: false)
      --debug-sql           SQL debug mode
      --http-proxy string   http://proxy-url:port (default: empty)
      --log-dir string      /path/to/log (default "/var/log/goval-dictionary")
      --log-json            output log as JSON
      --no-details          without vulnerability details
```

```bash
$ goval-dictionary fetch suse --suse-type opensuse 10.2 10.3 11.0 11.1 11.2 11.3 11.4 12.1 12.2 12.3 13.1 13.2 tumbleweed
$ goval-dictionary fetch suse --suse-type opensuse-leap 42.1 42.2 42.3 15.0 15.1 15.2 15.3 15.4 15.5 15.6
$ goval-dictionary fetch suse --suse-type suse-enterprise-server 9 10 11 12 15
$ goval-dictionary fetch suse --suse-type suse-enterprise-desktop 10 11 12 15
```

#### Usage: Fetch OVAL data from Oracle

- [Oracle Linux](https://linux.oracle.com/security/oval/)

```bash
 $ goval-dictionary fetch oracle 5 6 7 8 9
```

### Usage: Fetch alpine-secdb as OVAL data type

- [Alpine Linux](https://secdb.alpinelinux.org/)
alpine-secdb is provided in YAML format and not OVAL, but it is supported by goval-dictionary to make alpine-secdb easier to handle from Vuls.
See [here](https://secdb.alpinelinux.org/) for a list of supported alpines.

```bash
 $ goval-dictionary fetch alpine 3.2 3.3 3.4 3.5 3.6 3.7 3.8 3.9 3.10 3.11 3.12 3.13 3.14 3.15 3.16 3.17 3.18 3.19 3.20
```

#### Usage: Fetch Amazon ALAS as OVAL data type

Amazon ALAS provideis Vulnerability data as `no-OVAL-format`, but it is supported by goval-dictionary to make Amazon ALAS easier to handle from Vuls.

```bash
 $ goval-dictionary fetch amazon 1 2 2022 2023
```

#### Usage: Fetch Security Updates from Fedora

- [Fedora Updates](https://dl.fedoraproject.org/pub/fedora/linux/updates/)

```bash
$ goval-dictionary fetch fedora 32 33 34 35 36 37 38 39 40
```

### Usage: select oval by package name

Select from DB where package name is golang.

<details>
<summary>
`$ goval-dictionary select package redhat 7 golang`
</summary>

```bash
$ goval-dictionary select package redhat 7 golang
[Apr 10 10:22:43]  INFO Opening DB (sqlite3).
CVE-2015-5739
    {3399 319 golang 0:1.6.3-1.el7_2.1}
    {3400 319 golang-bin 0:1.6.3-1.el7_2.1}
    {3401 319 golang-docs 0:1.6.3-1.el7_2.1}
    {3402 319 golang-misc 0:1.6.3-1.el7_2.1}
    {3403 319 golang-src 0:1.6.3-1.el7_2.1}
    {3404 319 golang-tests 0:1.6.3-1.el7_2.1}
CVE-2015-5740
    {3399 319 golang 0:1.6.3-1.el7_2.1}
    {3400 319 golang-bin 0:1.6.3-1.el7_2.1}
    {3401 319 golang-docs 0:1.6.3-1.el7_2.1}
    {3402 319 golang-misc 0:1.6.3-1.el7_2.1}
    {3403 319 golang-src 0:1.6.3-1.el7_2.1}
    {3404 319 golang-tests 0:1.6.3-1.el7_2.1}
CVE-2015-5741
    {3399 319 golang 0:1.6.3-1.el7_2.1}
    {3400 319 golang-bin 0:1.6.3-1.el7_2.1}
    {3401 319 golang-docs 0:1.6.3-1.el7_2.1}
    {3402 319 golang-misc 0:1.6.3-1.el7_2.1}
    {3403 319 golang-src 0:1.6.3-1.el7_2.1}
    {3404 319 golang-tests 0:1.6.3-1.el7_2.1}
CVE-2016-3959
    {3399 319 golang 0:1.6.3-1.el7_2.1}
    {3400 319 golang-bin 0:1.6.3-1.el7_2.1}
    {3401 319 golang-docs 0:1.6.3-1.el7_2.1}
    {3402 319 golang-misc 0:1.6.3-1.el7_2.1}
    {3403 319 golang-src 0:1.6.3-1.el7_2.1}
    {3404 319 golang-tests 0:1.6.3-1.el7_2.1}
CVE-2016-5386
    {3399 319 golang 0:1.6.3-1.el7_2.1}
    {3400 319 golang-bin 0:1.6.3-1.el7_2.1}
    {3401 319 golang-docs 0:1.6.3-1.el7_2.1}
    {3402 319 golang-misc 0:1.6.3-1.el7_2.1}
    {3403 319 golang-src 0:1.6.3-1.el7_2.1}
    {3404 319 golang-tests 0:1.6.3-1.el7_2.1}
------------------
[]models.Definition{
  models.Definition{
    ID:          0x13f,
    MetaID:      0x1,
    Title:       "RHSA-2016:1538: golang security, bug fix, and enhancement update (Moderate)",
    Description: "The golang packages provide the Go programming language compiler.\n\nThe following packages have been upgraded to a newer upstream version: golang (1.6.3). (BZ#1346331)\n\nSecurity Fix(es):\n\n* An input-validation flaw was discovered in the Go programming language built in CGI implementation, which set the environment variable \"HTTP_PROXY\" using the incoming \"Proxy\" HTTP-request header. The environment variable \"HTTP_PROXY\" is used by numerous web clients, including Go's net/http package, to specify a proxy server to use for HTTP and, in some cases, HTTPS requests. This meant that when a CGI-based web application ran, an attacker could specify a proxy server which the application then used for subsequent outgoing requests, allowing a man-in-the-middle attack. (CVE-2016-5386)\n\nRed Hat would like to thank Scott Geary (VendHQ) for reporting this issue.",
    Advisory:    models.Advisory{
      ID:           0x13f,
      DefinitionID: 0x13f,
      Severity:     "Moderate",
      Cves:         []models.Cve{
        models.Cve{
          ID:         0x54f,
          AdvisoryID: 0x13f,
          CveID:      "CVE-2015-5739",
          Cvss2:      "6.8/AV:N/AC:M/Au:N/C:P/I:P/A:P",
          Cvss3:      "",
          Cwe:        "CWE-444",
          Href:       "https://access.redhat.com/security/cve/CVE-2015-5739",
          Public:     "20150729",
        },
        models.Cve{
          ID:         0x550,
          AdvisoryID: 0x13f,
          CveID:      "CVE-2015-5740",
          Cvss2:      "6.8/AV:N/AC:M/Au:N/C:P/I:P/A:P",
          Cvss3:      "",
          Cwe:        "CWE-444",
          Href:       "https://access.redhat.com/security/cve/CVE-2015-5740",
          Public:     "20150729",
        },
        models.Cve{
          ID:         0x551,
          AdvisoryID: 0x13f,
          CveID:      "CVE-2015-5741",
          Cvss2:      "6.8/AV:N/AC:M/Au:N/C:P/I:P/A:P",
          Cvss3:      "",
          Cwe:        "CWE-444",
          Href:       "https://access.redhat.com/security/cve/CVE-2015-5741",
          Public:     "20150729",
        },
        models.Cve{
          ID:         0x552,
          AdvisoryID: 0x13f,
          CveID:      "CVE-2016-3959",
          Cvss2:      "4.3/AV:N/AC:M/Au:N/C:N/I:N/A:P",
          Cvss3:      "5.3/CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L",
          Cwe:        "CWE-835",
          Href:       "https://access.redhat.com/security/cve/CVE-2016-3959",
          Public:     "20160405",
        },
        models.Cve{
          ID:         0x553,
          AdvisoryID: 0x13f,
          CveID:      "CVE-2016-5386",
          Cvss2:      "5.0/AV:N/AC:L/Au:N/C:N/I:P/A:N",
          Cvss3:      "5.0/CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:C/C:N/I:L/A:N",
          Cwe:        "CWE-20",
          Href:       "https://access.redhat.com/security/cve/CVE-2016-5386",
          Public:     "20160718",
        },
      },
      Bugzillas: []models.Bugzilla{
        models.Bugzilla{
          ID:         0x93f,
          AdvisoryID: 0x13f,
          BugzillaID: "1346331",
          URL:        "https://bugzilla.redhat.com/1346331",
          Title:      "REBASE to golang 1.6",
        },
        models.Bugzilla{
          ID:         0x940,
          AdvisoryID: 0x13f,
          BugzillaID: "1353798",
          URL:        "https://bugzilla.redhat.com/1353798",
          Title:      "CVE-2016-5386 Go: sets environmental variable  based on user supplied Proxy request header",
        },
      },
      AffectedCPEList: []models.Cpe{
        models.Cpe{
          ID:         0x204,
          AdvisoryID: 0x13f,
          Cpe:        "cpe:/o:redhat:enterprise_linux:7",
        },
      },
    },
    AffectedPacks: []models.Package{
      models.Package{
        ID:           0xd47,
        DefinitionID: 0x13f,
        Name:         "golang",
        Version:      "0:1.6.3-1.el7_2.1",
      },
      models.Package{
        ID:           0xd48,
        DefinitionID: 0x13f,
        Name:         "golang-bin",
        Version:      "0:1.6.3-1.el7_2.1",
      },
      models.Package{
        ID:           0xd49,
        DefinitionID: 0x13f,
        Name:         "golang-docs",
        Version:      "0:1.6.3-1.el7_2.1",
      },
      models.Package{
        ID:           0xd4a,
        DefinitionID: 0x13f,
        Name:         "golang-misc",
        Version:      "0:1.6.3-1.el7_2.1",
      },
      models.Package{
        ID:           0xd4b,
        DefinitionID: 0x13f,
        Name:         "golang-src",
        Version:      "0:1.6.3-1.el7_2.1",
      },
      models.Package{
        ID:           0xd4c,
        DefinitionID: 0x13f,
        Name:         "golang-tests",
        Version:      "0:1.6.3-1.el7_2.1",
      },
    },
    References: []models.Reference{
      models.Reference{
        ID:           0x68d,
        DefinitionID: 0x13f,
        Source:       "RHSA",
        RefID:        "RHSA-2016:1538-01",
        RefURL:       "https://rhn.redhat.com/errata/RHSA-2016-1538.html",
      },
      models.Reference{
        ID:           0x68e,
        DefinitionID: 0x13f,
        Source:       "CVE",
        RefID:        "CVE-2015-5739",
        RefURL:       "https://access.redhat.com/security/cve/CVE-2015-5739",
      },
      models.Reference{
        ID:           0x68f,
        DefinitionID: 0x13f,
        Source:       "CVE",
        RefID:        "CVE-2015-5740",
        RefURL:       "https://access.redhat.com/security/cve/CVE-2015-5740",
      },
      models.Reference{
        ID:           0x690,
        DefinitionID: 0x13f,
        Source:       "CVE",
        RefID:        "CVE-2015-5741",
        RefURL:       "https://access.redhat.com/security/cve/CVE-2015-5741",
      },
      models.Reference{
        ID:           0x691,
        DefinitionID: 0x13f,
        Source:       "CVE",
        RefID:        "CVE-2016-3959",
        RefURL:       "https://access.redhat.com/security/cve/CVE-2016-3959",
      },
      models.Reference{
        ID:           0x692,
        DefinitionID: 0x13f,
        Source:       "CVE",
        RefID:        "CVE-2016-5386",
        RefURL:       "https://access.redhat.com/security/cve/CVE-2016-5386",
      },
    },
  },
}

```

Upper part format:
```
CVE-YYYY-NNNN
    {ID DefinitionID PackageName PackageVersion NotFixedYet}
```


</details>

### Usage: select oval by CVE-ID

<details>
<summary>
`Select from DB where CVE-ID CVE-2017-6009`
</summary>

```bash
$ goval-dictionary select cve-id redhat 7 CVE-2017-6009
[Apr 12 12:12:36]  INFO Opening DB (sqlite3).
RHSA-2017:0837: icoutils security update (Important)
Important
[{1822 430 CVE-2017-5208  8.1/CVSS:3.0/AV:L/AC:L/PR:L/UI:R/S:C/C:H/I:H/A:L CWE-190 CWE-122 https://access.redhat.com/security/cve/CVE-2017-5208 20170108} {1823 430 CVE-2017-5332  2.8/CVSS:3.0/AV:L/AC:L/PR:L/UI:R/S:U/C:N/I:N/A:L CWE-190 CWE-125 https://access.redhat.com/security/cve/CVE-2017-5332 20170108} {1824 430 CVE-2017-5333  8.1/CVSS:3.0/AV:L/AC:L/PR:L/UI:R/S:C/C:H/I:H/A:L CWE-190 CWE-122 https://access.redhat.com/security/cve/CVE-2017-5333 20170108} {1825 430 CVE-2017-6009  8.1/CVSS:3.0/AV:L/AC:L/PR:L/UI:R/S:C/C:H/I:H/A:L CWE-190 CWE-122 https://access.redhat.com/security/cve/CVE-2017-6009 20170203} {1826 430 CVE-2017-6010  8.1/CVSS:3.0/AV:L/AC:L/PR:L/UI:R/S:C/C:H/I:H/A:L CWE-190 CWE-122 https://access.redhat.com/security/cve/CVE-2017-6010 20170203} {1827 430 CVE-2017-6011  8.1/CVSS:3.0/AV:L/AC:L/PR:L/UI:R/S:C/C:H/I:H/A:L CWE-122 https://access.redhat.com/security/cve/CVE-2017-6011 20170203}]
------------------
[]models.Definition{
  models.Definition{
    ID:          0x1ae,
    MetaID:      0x1,
    Title:       "RHSA-2017:0837: icoutils security update (Important)",
    Description: "The icoutils are a set of programs for extracting and converting images in Microsoft Windows icon and cursor files. These files usually have the extension .ico or .cur, but they can also be embedded in executables or libraries.\n\nSecurity Fix(es):\n\n* Multiple vulnerabilities were found in icoutils, in the wrestool program. An attacker could create a crafted executable that, when read by wrestool, could result in memory corruption leading to a crash or potential code execution. (CVE-2017-5208, CVE-2017-5333, CVE-2017-6009)\n\n* A vulnerability was found in icoutils, in the wrestool program. An attacker could create a crafted executable that, when read by wrestool, could result in failure to allocate memory or an over-large memcpy operation, leading to a crash. (CVE-2017-5332)\n\n* Multiple vulnerabilities were found in icoutils, in the icotool program. An attacker could create a crafted ICO or CUR file that, when read by icotool, could result in memory corruption leading to a crash or potential code execution. (CVE-2017-6010, CVE-2017-6011)",
    Advisory:    models.Advisory{
      ID:           0x1ae,
      DefinitionID: 0x1ae,
      Severity:     "Important",
      Cves:         []models.Cve{
        models.Cve{
          ID:         0x71e,
          AdvisoryID: 0x1ae,
          CveID:      "CVE-2017-5208",
          Cvss2:      "",
          Cvss3:      "8.1/CVSS:3.0/AV:L/AC:L/PR:L/UI:R/S:C/C:H/I:H/A:L",
          Cwe:        "CWE-190 CWE-122",
          Href:       "https://access.redhat.com/security/cve/CVE-2017-5208",
          Public:     "20170108",
        },
        models.Cve{
          ID:         0x71f,
          AdvisoryID: 0x1ae,
          CveID:      "CVE-2017-5332",
          Cvss2:      "",
          Cvss3:      "2.8/CVSS:3.0/AV:L/AC:L/PR:L/UI:R/S:U/C:N/I:N/A:L",
          Cwe:        "CWE-190 CWE-125",
          Href:       "https://access.redhat.com/security/cve/CVE-2017-5332",
          Public:     "20170108",
        },
        models.Cve{
          ID:         0x720,
          AdvisoryID: 0x1ae,
          CveID:      "CVE-2017-5333",
          Cvss2:      "",
          Cvss3:      "8.1/CVSS:3.0/AV:L/AC:L/PR:L/UI:R/S:C/C:H/I:H/A:L",
          Cwe:        "CWE-190 CWE-122",
          Href:       "https://access.redhat.com/security/cve/CVE-2017-5333",
          Public:     "20170108",
        },
        models.Cve{
          ID:         0x721,
          AdvisoryID: 0x1ae,
          CveID:      "CVE-2017-6009",
          Cvss2:      "",
          Cvss3:      "8.1/CVSS:3.0/AV:L/AC:L/PR:L/UI:R/S:C/C:H/I:H/A:L",
          Cwe:        "CWE-190 CWE-122",
          Href:       "https://access.redhat.com/security/cve/CVE-2017-6009",
          Public:     "20170203",
        },
        models.Cve{
          ID:         0x722,
          AdvisoryID: 0x1ae,
          CveID:      "CVE-2017-6010",
          Cvss2:      "",
          Cvss3:      "8.1/CVSS:3.0/AV:L/AC:L/PR:L/UI:R/S:C/C:H/I:H/A:L",
          Cwe:        "CWE-190 CWE-122",
          Href:       "https://access.redhat.com/security/cve/CVE-2017-6010",
          Public:     "20170203",
        },
        models.Cve{
          ID:         0x723,
          AdvisoryID: 0x1ae,
          CveID:      "CVE-2017-6011",
          Cvss2:      "",
          Cvss3:      "8.1/CVSS:3.0/AV:L/AC:L/PR:L/UI:R/S:C/C:H/I:H/A:L",
          Cwe:        "CWE-122",
          Href:       "https://access.redhat.com/security/cve/CVE-2017-6011",
          Public:     "20170203",
        },
      },
      Bugzillas: []models.Bugzilla{
        models.Bugzilla{
          ID:         0xe4a,
          AdvisoryID: 0x1ae,
          BugzillaID: "1411251",
          URL:        "https://bugzilla.redhat.com/1411251",
          Title:      "CVE-2017-5208 icoutils: Check_offset overflow on 64-bit systems",
        },
        models.Bugzilla{
          ID:         0xe4b,
          AdvisoryID: 0x1ae,
          BugzillaID: "1412259",
          URL:        "https://bugzilla.redhat.com/1412259",
          Title:      "CVE-2017-5333 icoutils: Integer overflow vulnerability in extract.c",
        },
        models.Bugzilla{
          ID:         0xe4c,
          AdvisoryID: 0x1ae,
          BugzillaID: "1412263",
          URL:        "https://bugzilla.redhat.com/1412263",
          Title:      "CVE-2017-5332 icoutils: Access to unallocated memory possible in extract.c",
        },
        models.Bugzilla{
          ID:         0xe4d,
          AdvisoryID: 0x1ae,
          BugzillaID: "1422906",
          URL:        "https://bugzilla.redhat.com/1422906",
          Title:      "CVE-2017-6009 icoutils: Buffer overflow in the decode_ne_resource_id function",
        },
        models.Bugzilla{
          ID:         0xe4e,
          AdvisoryID: 0x1ae,
          BugzillaID: "1422907",
          URL:        "https://bugzilla.redhat.com/1422907",
          Title:      "CVE-2017-6010 icoutils: Buffer overflow in the extract_icons function",
        },
        models.Bugzilla{
          ID:         0xe4f,
          AdvisoryID: 0x1ae,
          BugzillaID: "1422908",
          URL:        "https://bugzilla.redhat.com/1422908",
          Title:      "CVE-2017-6011 icoutils: Buffer overflow in the simple_vec function",
        },
      },
      AffectedCPEList: []models.Cpe{
        models.Cpe{
          ID:         0x2ae,
          AdvisoryID: 0x1ae,
          Cpe:        "cpe:/o:redhat:enterprise_linux:7",
        },
      },
    },
    AffectedPacks: []models.Package{
      models.Package{
        ID:           0x11b1,
        DefinitionID: 0x1ae,
        Name:         "icoutils",
        Version:      "0:0.31.3-1.el7_3",
      },
    },
    References: []models.Reference{
      models.Reference{
        ID:           0x8cb,
        DefinitionID: 0x1ae,
        Source:       "RHSA",
        RefID:        "RHSA-2017:0837-01",
        RefURL:       "https://access.redhat.com/errata/RHSA-2017:0837",
      },
      models.Reference{
        ID:           0x8cc,
        DefinitionID: 0x1ae,
        Source:       "CVE",
        RefID:        "CVE-2017-5208",
        RefURL:       "https://access.redhat.com/security/cve/CVE-2017-5208",
      },
      models.Reference{
        ID:           0x8cd,
        DefinitionID: 0x1ae,
        Source:       "CVE",
        RefID:        "CVE-2017-5332",
        RefURL:       "https://access.redhat.com/security/cve/CVE-2017-5332",
      },
      models.Reference{
        ID:           0x8ce,
        DefinitionID: 0x1ae,
        Source:       "CVE",
        RefID:        "CVE-2017-5333",
        RefURL:       "https://access.redhat.com/security/cve/CVE-2017-5333",
      },
      models.Reference{
        ID:           0x8cf,
        DefinitionID: 0x1ae,
        Source:       "CVE",
        RefID:        "CVE-2017-6009",
        RefURL:       "https://access.redhat.com/security/cve/CVE-2017-6009",
      },
      models.Reference{
        ID:           0x8d0,
        DefinitionID: 0x1ae,
        Source:       "CVE",
        RefID:        "CVE-2017-6010",
        RefURL:       "https://access.redhat.com/security/cve/CVE-2017-6010",
      },
      models.Reference{
        ID:           0x8d1,
        DefinitionID: 0x1ae,
        Source:       "CVE",
        RefID:        "CVE-2017-6011",
        RefURL:       "https://access.redhat.com/security/cve/CVE-2017-6011",
      },
    },
  },
}

```

Upper part format:
```
[
  {ID AdvisoryID CveID Cvss2 Cvss3 CWE Impact ReferenceURL PublishedDate}
  ...
]
```
</details>

### Usage: select advisories

<details>
<summary>
`Select Advisories from DB`
</summary>

```bash
$ goval-dictionary select advisories redhat 9
map[string][]string{
  "RHSA-2023:6482": []string{
    "CVE-2023-35789",
  },
  "RHSA-2022:8418": []string{
    "CVE-2021-28153",
  },
  "RHSA-2024:0811": []string{
    "CVE-2023-28486",
    "CVE-2023-28487",
    "CVE-2023-42465",
    "CVE-2023-28486",
    "CVE-2023-28487",
    "CVE-2023-42465",
    "CVE-2023-28486",
    "CVE-2023-28487",
    "CVE-2023-42465",
  },
  ...
}
```

### Usage: Start goval-dictionary as server mode

```bash
$ goval-dictionary server --help
Start OVAL dictionary HTTP server

Usage:
  goval-dictionary server [flags]

Flags:
      --bind string   HTTP server bind to IP address (default "127.0.0.1")
  -h, --help          help for server
      --port string   HTTP server port number (default "1324")

Global Flags:
      --config string       config file (default is $HOME/.oval.yaml)
      --dbpath string       /path/to/sqlite3 or SQL connection string (default "/$PWD/oval.sqlite3")
      --dbtype string       Database type to store data in (sqlite3, mysql, postgres or redis supported) (default "sqlite3")
      --debug               debug mode (default: false)
      --debug-sql           SQL debug mode
      --http-proxy string   http://proxy-url:port (default: empty)
      --log-dir string      /path/to/log (default "/var/log/goval-dictionary")
      --log-json            output log as JSON
```

#### cURL

```
$ curl http://127.0.0.1:1324/cves/ubuntu/16/CVE-2017-15400 | jq
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100  1237  100  1237    0     0  81365      0 --:--:-- --:--:-- --:--:-- 82466
[
  {
    "ID": 10582,
    "DefinitionID": "oval:com.ubuntu.xenial:def:201715400000",
    "Title": "CVE-2017-15400 on Ubuntu 16.04 LTS (xenial) - medium.",
    "Description": "Insufficient restriction of IPP filters in CUPS in Google Chrome OS prior to 62.0.3202.74 allowed a remote attacker to execute a command with the same privileges as the cups daemon via a crafted PPD file, aka a printer zeroconfig CRLF issue.",
    "Advisory": {
      "ID": 10575,
      "Severity": "Medium",
      "Cves": null,
      "Bugzillas": null,
      "AffectedCPEList": null,
      "Issued": "0001-01-01T00:00:00Z",
      "Updated": "0001-01-01T00:00:00Z"
    },
    "Debian": {
      "ID": 9330,
      "CveID": "CVE-2017-15400",
      "MoreInfo": "",
      "Date": "0001-01-01T00:00:00Z"
    },
    "AffectedPacks": [
      {
        "ID": 16117,
        "Name": "cups",
        "Version": "",
        "NotFixedYet": true
      }
    ],
    "References": [
      {
        "ID": 48602,
        "Source": "CVE",
        "RefID": "CVE-2017-15400",
        "RefURL": "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-15400"
      },
      {
        "ID": 48603,
        "Source": "Ref",
        "RefID": "",
        "RefURL": "http://people.canonical.com/~ubuntu-security/cve/2017/CVE-2017-15400.html"
      },
      {
        "ID": 48604,
        "Source": "Ref",
        "RefID": "",
        "RefURL": "https://chromereleases.googleblog.com/2017/10/stable-channel-update-for-chrome-os_27.html"
      },
      {
        "ID": 48605,
        "Source": "Bug",
        "RefID": "",
        "RefURL": "https://bugs.chromium.org/p/chromium/issues/detail?id=777215"
      }
    ]
  }
]
```

For details, see https://github.com/vulsio/goval-dictionary/blob/master/server/server.go#L44

----

## Tips

- How to use Redis as DB backend
see [#7](https://github.com/vulsio/goval-dictionary/pull/7)

----

## Data Source

- [RedHat](https://www.redhat.com/security/data/oval/)
- [Debian](https://www.debian.org/security/oval/)
- [Ubuntu(main)](https://security-metadata.canonical.com/oval/)
- [Ubuntu(sub)](https://people.canonical.com/~ubuntu-security/oval/)
- [SUSE](http://ftp.suse.com/pub/projects/security/oval/)
- [Oracle Linux](https://linux.oracle.com/security/oval/)
- [Alpine-secdb](https://secdb.alpinelinux.org/)
- [Amazon](https://alas.aws.amazon.com/alas.rss)

----

## Authors

kotakanbe ([@kotakanbe](https://twitter.com/kotakanbe)) created goval-dictionary and [these fine people](https://github.com/vulsio/goval-dictionary/graphs/contributors) have contributed.

----

## Change Log

Please see [CHANGELOG](https://github.com/vulsio/goval-dictionary/blob/master/CHANGELOG.md).

----

## License

Please see [LICENSE](https://github.com/vulsio/goval-dictionary/blob/master/LICENSE).
