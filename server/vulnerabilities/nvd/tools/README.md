> The contents of this directory were copied (in April 2024) from  https://github.com/facebookincubator/nvdtools.git.
---

![Tests](https://github.com/facebookincubator/nvdtools/actions/workflows/tests.yaml/badge.svg)

# NVD Tools

A collection of tools for working with [National Vulnerability Database](https://nvd.nist.gov/) feeds.

The [HOWTO](HOWTO.md) provides a broader view on how to effectively use these tools.

---

* [Requirements](#requirements)
* [Installation](#installation)
* [How build](#How-build)
* [Command line tools](#command-line-tools)
  * [cpe2cve](#cpe2cve)
  * [csv2cpe](#cpe2cve)
  * [fireeye2nvd](#fireeye2nvd)
  * [flexera2nvd](#flexera2nvd)
  * [idefense2nvd](#idefense2nvd)
  * [nvdsync](#nvdsync)
  * [rpm2cpe](#rpm2cpe)
  * [rustsec2nvd](#rustsec2nvd)
  * [vfeed2nvd](#vfeed2nvd)
  * [vulndb](#vulndb)
* [Libraries](#libraries)
  * [cvss2](#cvss2)
  * [cvss3](#cvss3)
  * [wfn](#wfn)
* [License](#license)

---

## Requirements

* Go 1.13 or newer

## Installation

You need a properly setup Go environment.

#### Download and install NVD Tools:

For Go 1.13 - 1.14:
```bash
go get github.com/facebookincubator/nvdtools/...
cd "$GOPATH"/src/github.com/facebookincubator/nvdtools/cmd
go install ./...
```

From Go 1.15 onwards, modules are not downloaded to `GOPATH`, but to `GOMODCACHE`. It is recommended to clone the repo and run run go install from there instead:
```bash
git clone https://github.com/facebookincubator/nvdtools
cd nvdtools
go install ./...
```

From Go 1.17 onwards, `go get` is deprecated. `go install` is used instead to download the module to the cache and install it:
```bash
go install github.com/facebookincubator/nvdtools/...@latest
```

## How-build
```bash
go mod init github.com/facebookincubator/nvdtools
go mod tidy
make
cp build/bin/* ~/go/bin/

```

## Command line tools

### `cpe2cve`

*cpe2cve* is a command line tool for scanning an inventory of CPE names for vulnerabilities.

It expects a stream of lines of delimiter-separated fields, one of these fields being a delimiter-separated list of CPE names in the inventory.

Vulnerability feeds should be provided as arguments to the program in JSON format.

Output is a stream of delimiter-separated input value decorated with a vulnerability ID (CVE) and a delimiter-separated list of CPE names that match this vulnerability.

Unwanted input fields could be erased from the output with `-e` option.

Input and output delimiters can be configured with `-d`, `-d2`, `-o` an `-o2` options.

The column to which output the CVE and matches for that CVE can be configured with `-cve` and `-matches` options correspondingly.

### download data
```bash
curl -o- -s -k -v https://nvd.nist.gov/vuln/data-feeds >data-feeds.html
cat data-feeds.html|grep  -Eo '(/feeds\/[^"]*\.gz)'|xargs -I % wget -c https://nvd.nist.gov%
```

#### Example 1: scan a software for vulnerabilities

```bash
echo "cpe:/a:apache"|cpe2cve -cpe 1 -e 1 -cve 1  nvdcve-1.1-*.json.gz
echo "cpe:/a:gnu:glibc:2.28" | cpe2cve -cpe 1 -e 1 -cve 1 nvdcve-1.0-*.json.gz
CVE-2009-4881
CVE-2015-8985
CVE-2016-4429
CVE-2010-3192
CVE-2010-4756
```

#### Example 2: find vulnerabilities in software inventory per production host

```bash
./cpe2cve -d ' ' -d2 , -o ' ' -o2 , -cpe 2 -e 2 -matches 3 -cve 2 nvdcve-1.0-*.json.gz << EOF
host1.foo.bar cpe:/a:gnu:glibc:2.28,cpe:/a:gnu:zlib:1.2.8
host2.foo.bar cpe:/a:gnu:glibc:2.28,cpe:/a:haxx:curl:7.55.0
EOF
host1.foo.bar CVE-2009-4881 cpe:/a:gnu:glibc:2.28
host1.foo.bar CVE-2016-4429 cpe:/a:gnu:glibc:2.28
host2.foo.bar CVE-2014-5119 cpe:/a:gnu:glibc:2.28
host2.foo.bar CVE-2016-4429 cpe:/a:gnu:glibc:2.28
host2.foo.bar CVE-2018-1000120 cpe:/a:haxx:curl:7.55.0
host2.foo.bar CVE-2018-1000122 cpe:/a:haxx:curl:7.55.0
host2.foo.bar CVE-2010-4756 cpe:/a:gnu:glibc:2.28
host2.foo.bar CVE-2017-8817 cpe:/a:haxx:curl:7.55.0
```

### `csv2cpe`

*csv2cpe* is a tool that generates a URI-bound CPE from CSV input, flags configure the meaning of each input field:

* `-cpe_part` -- identifies the class of a product: h for hardware, a for application and o for OS
* `-cpe_vendor` -- identifies  the person or organisation that manufactured or created the product
* `-cpe_product` -- describes or identifies the most common and recognisable title or name of the product
* `-cpe_version` -- vendor-specific alphanumeric strings characterising the particular release version of the product
* `-cpe_update` -- vendor-specific alphanumeric strings characterising the particular update, service pack, or point release of the product
* `-cpe_edition` -- capture edition-related terms applied by the vendor to the product; this attribute is considered deprecated in CPE specification version 2.3 and it should be assigned the logical value ANY except where required for backward compatibility with version 2.2 of the CPE specification.
* `-cpe_swedition` -- characterises how the product is tailored to a particular market or class of end users
* `-cpe_targetsw` -- characterises the software computing environment within which the product operates
* `-cpe_targethw` -- characterises the software computing environment within which the product operates
* `-cpe_language` --  defines the language supported in the user interface of the product being described; must be valid language tags as defined by [RFC5646]
* `-cpe_other` -- any other general descriptive or identifying information which is vendor- or product-specific and which does not logically fit in any other attribute value

Omitted parts of the CPE name defaults to logical value ANY, as per [specification](https://nvlpubs.nist.gov/nistpubs/Legacy/IR/nistir7695.pdf)

Optional flag `-lower` brings the strings to lower case.

#### Example: generate URI-bound CPE name out of comma-separated list of attributes

```bash
$ echo 'a,Microsoft,Internet Explorer,8.1,SP1,-,*' | csv2cpe -x -lower -cpe_part=1 -cpe_vendor=2 -cpe_product=3 -cpe_version=4 -cpe_update=5 -cpe_edition=6 -cpe_language=7
cpe:/a:microsoft:internet_explorer:8.1:sp1:-
```

### `fireeye2nvd`

*fireeye2nvd* downloads the vulnerability data from [FireEye](https://www.fireeye.com/) and converts it into NVD format. The resulting file can be used as a feed in [`cpe2cve`](#cpe2cve) processor

### `flexera2nvd`

*flexera2nvd* downloads the vulnerability data from [Flexera](https://www.flexera.com/) and converts it into NVD format. The resulting file can be used as a feed in [`cpe2cve`](#cpe2cve) processor

### `idefense2nvd`

*idefense2nvd* downloads the vulnerability data from Idefense and converts it into NVD format. The resulting file can be used as a feed in [`cpe2cve`](#cpe2cve) processor

### `nvdsync`

*nvdsync* synchronizes NVD data feeds to local directory; it  checks the hashes of the files against the ones provided by NVD and only updates the changed files.

### `rpm2cpe`

*rpm2cpe* takes a delimiter-separated input with one of the fields containing RPM package name and produces delimiter-separated output consisting of the same fields plus CPE name parsed from RPM package name.

#### Example: generate URI-bound CPE name out of RPM package filename

```bash
echo openoffice-eu-writer-4.1.5-9789.i586.rpm | rpm2cpe -rpm=1 -cpe=2 -e=1
cpe:/a::openoffice-eu-writer:4.1.5:9789:~~~~i586~
```

### `rustsec2nvd`

*rustsec2nvd* converts the vulnerabilities from the [Rustsec Advisory-DB](https://github.com/RustSec/advisory-db) into NVD format. The resulting file can be used as a feed in [`cpe2cve`](#cpe2cve) processor

### `snyk2nvd`

*snyk2nvd* downloads the vulnerability data from [Snyk](https://snyk.io/) and converts it into NVD format. The resulting file can be used as a feed in [`cpe2cve`](#cpe2cve) processor

### `vfeed2nvd`

*vfeed2nvd* converts the vulnerability data from [vFeed](https://vfeed.io/) into NVD format. The resulting file can be used as a feed in [`cpe2cve`](#cpe2cve) processor

### `vulndb`

*vulndb* is a command line tool to manage NVD-like vulnerability databases, backed by MySQL.

Supports NVD CVE JSON 1.0 feeds. Data is versioned, organized by provider names and grouped by vendor, custom, and snoozes datasets:

* Vendor dataset: read-only CVE feeds we continuously import.
* Custom dataset: allows to overwrite CVEs from vendor data with custom data during exports
* Snooze dataset: user-defined CVE and metadata with deadline, used for remediation automation

See `vulndb help` for details.

## Libraries

### cvss2

Implementation of [CVSS v2 specification](https://www.first.org/cvss/v2/guide) which provides functions for serializing and deserializing vectors as well as score calculation.

### cvss3

Implementation of [CVSS v3 specification](https://www.first.org/cvss/specification-document) which provides functions for serializing and deserializing vectors as well as score calculation.

## License

nvdtools licensed under Apache License, Version 2.0, as found in the [LICENSE](LICENSE) file.
