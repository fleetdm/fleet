# How to use nvdtools

The command line tools provided by nvdtools were designed for processing inventory data in pipelines.

To start, you will need a vulnerability database. In this toolkit you'll find the [nvdsync](https://github.com/facebookincubator/nvdtools/tree/master/cmd/nvdsync) command, which can download the public NVD database to local disk:

```bash
nvdsync -v=1 -cve_feed=cve-1.0.json.gz /tmp/nvd
```

Next up, you need a data collector to create a CPE inventory. Collectors are domain-specific programs capable of acquiring asset information (e.g. a list of hardware, or packages in a repo or system) and printing this information to standard output.

Think of the simplest data collector as an execution of rpm (or repoquery):

```bash
rpm -qa | rpm2cpe -rpm=1 -cpe=2
```

This collector [rpm2cpe](https://github.com/facebookincubator/nvdtools/tree/master/cmd/rpm2cpe) will use the name of the rpm files in column 1 of the input, produce a CPE in column 2, and print both to standard output.

Finally, use the [cpe2cve](https://github.com/facebookincubator/nvdtools/tree/master/cmd/cpe2cve) processor to consume the CPE inventory from standard input and print CVEs affecting which CPEs to standard output:

```bash
rpm -qa | \
rpm2cpe -rpm=1 -cpe=2 | \
cpe2cve -cpe=2 -cve=3 -cwe=4 /tmp/nvd/*.json.gz
```

The command above process each CPE individually and prints their respective CVEs. However, it's not uncommon in the NVD database to have more elaborate CVEs which affect a combination of CPEs, e.g. if A and B and not C. For this case, you could group your CPEs per host, for example, and process them in a single batch:

```bash
set -o pipefail
(hostname
rpm -qa | rpm2cpe -rpm=1 -cpe=2 -e=1 | sort -u | paste -s -d, | \
cpe2cve -cpe=1 -cve=2 -e=1 /tmp/nvd/*.json.gz | paste -s -d,) | paste -s -d'\t'
```

The command above prints a single line containing `<hostname> <tab> <comma-separated list of CVEs>` for your machine. Great, but is unrealistic to use in each machine in production systems. That's when things start to get more interesting. See the next section for how to decouple this pipeline from collection to processing and reporting.

# Using nvdtools in production

In order to effectively use nvdtools, you will likely want to decouple data collection from processing and reporting.

The idea is to use nvdtools as the building blocks of a much larger system that orchestrates data collection separately from processing, leaving the processing and reporting (heavy lifting) to be executed in a data warehouse.

Starting from the data collection, think of the different inventory classes that may exist in the environment:

* Hosted software: packages sitting in software repositories, available to your fleet (source and binary, first-party and third-party)
* Installed software: packages installed on machines or containers, ideally from your managed repositories
* Running software: processes executing on machines or containers, ideally from a known package
* Hardware: a list of hardware parts that can be used to create CPEs, e.g. `cpe:/h:dell:inspiron:8500`

The collecting stage have different requirements for each class. The processing stage consume inventories from these collector classes and process them with specialized vulnerability databases.

The public NVD database covers a great deal of open source software and common hardware. However, there are several ecosystems that may be present in your infrastructure (php, python, nodeJS, go) but not well covered by the NVD database alone.

To maximize vulnerability matching and coverage (and data quality, later user experience on reports), consider using multiple database providers. You will need to convert their databases to the [NVD CVE JSON 1.0](https://csrc.nist.gov/schema/nvd/feed/1.0/nvd_cve_feed_json_1.0.schema) format to use them with the cpe2cve processor.

Once the data from collectors is decoupled from processing, the nvdtools can be used to process large inventories with millions of assets more efficiently.

The following sections cover collectors and processors in a bit more depth.

## Collectors

This section covers some of the inventory classes mentioned above.

Collectors are all about retrieving asset information and providing enough data to build CPEs for late processing.

### Hosted software collectors

These are domain-specific programs that scrape software repositories (or logs) and report packages available to the fleet.

Examples of hosted software collectors are programs to report packages hosted in yum, maven, munki, chocolatey, docker registry.

Data provided by hosted software collectors must contain enough information to create CPEs, comprising at least the asset type (part; a=sw, h=hw, o=os), product and version. Other fields like vendor and target hardware can improve vulnerability matching later, but are not blockers to get started.

Orchestrating the execution of collectors is platform dependent. At the very least, a cron-like system could periodically run collectors and store their data in files and/or a database.

Here's an example of cron-like job to scrape all yum repositories configured on the machine running the collector:

```bash
Q=('{"vendor":"%{VENDOR}","product":"%{NAME}","version":"%{VERSION}","update":"%{RELEASE}","target_hw":"%{ARCH}","metadata":{"product_group":"%{REPO}","package_name":"%{NAME}-%{VERSION}-%{RELEASE}.%{ARCH}.rpm","package_source":"%{SOURCERPM}"}}')

set -o pipefail
repoquery -C --all --queryformat "${Q[@]}" | \
jq -r '[ "a", .vendor, .product, .version // "-", .update, .sw_edition, .target_sw, .target_hw, ( .metadata | tojson ) ] | @csv' | \
csv2cpe \
    -cpe_part=1 \
    -cpe_vendor=2 \
    -cpe_product=3 \
    -cpe_version=4 \
    -cpe_update=5 \
    -cpe_swedition=6 \
    -cpe_targetsw=7 \
    -cpe_targethw=8 \
    -e=1 \
    -i=1 \
    -lower \
    -o=$'\t'
```

The `-e=1` flag erases the injected "a" part from jq, and the `-i=1` flag tells [csv2cpe](https://github.com/facebookincubator/nvdtools/tree/master/cmd/csv2cpe) to add the cpe in column 1 of its output.

The tab-separated output contains the following columns:

```
cpe, vendor, product, version, update, sw_edition, target_sw, target_hw, metadata_json
```

This type of output can be stored in a database such as MySQL by simply adding `mysqlimport` at the end of the pipeline; or write the output to a message queue in similar fashion.

If executing jq and csv2cpe along with the collector is not an option, you can always store the raw JSON inventory and later execute jq and csv2cpe in the processing stage of the pipeline.

Notice the metadata field: that information may be helpful much later on the processing and reporting stages, allowing your system to report packages in such a way that your users understand them, avoiding people having to learn the CPE format and details of your system.

### Installed software collectors

There are several ways of collecting information about packages installed on a system. We mostly use [osquery](https://osquery.io/) for this, taking periodic snapshots of what is installed on a machine and shipping the data to the data warehouse.

The main advantage of using osquery is to support all major operating systems with a SQL-like interface for collecting information.

The osquery results are used to build CPEs which are later processed in batches.

Here's an example of a query to collect the macOS operating system version and all apps installed:

```bash
Q=("
SELECT
    'o' AS part,
    'apple' AS vendor,
    os.name AS product,
    os.version
FROM
    os_version AS os
;
SELECT
    'a' AS part,
    '' AS vendor,
    bundle_name AS product,
    bundle_version AS version
FROM
    apps
WHERE
    bundle_name IS NOT NULL AND bundle_name <> ''
;
")

osqueryi --json "${Q[@]}"
```

Although osquery supports a `--csv` flag, the JSON output gives flexibility (e.g. handling NULL values) and we can use jq to re-format to CSV, then use csv2cpe to produce the installed software inventory:

```bash
set -o pipefail
osqueryi --json "${Q[@]}" | \
jq -r '.[] | [.part, .vendor, .product, .version // "-"] | @csv' | \
csv2cpe \
    -cpe_part=1 \
    -cpe_vendor=2 \
    -cpe_product=3 \
    -cpe_version=4 \
    -e=1 \
    -i=1 \
    -lower \
    -o=$'\t'
```

Converting NULL versions to '-' tells the processor (much later, when cpe2cve is run) to handle dash as "Not Available" during CVE matching, instead of "Any" for empty space.

RPM packages have richer information, and provide extra metadata for matching CPEs against data from the hosted software collector. Following is a more complex query returning host RPM inventory with metadata:

```bash
Q=("
SELECT
    'o' AS part,
    'centos' AS vendor,
    'centos' AS product,
    (os.major || '.' || os.minor || '.' || os.patch) AS version,
    '' AS release,
    sys.cpu_type AS target_hw,
    NULL as metadata
FROM
    os_version AS os, system_info AS sys
;
SELECT
    'a' AS part,
    '' AS vendor,
    name AS product,
    version,
    release,
    arch AS target_hw,
    JSON_OBJECT(
        'package_name', (name || '-' || version || '-' || release || '.' || arch || '.rpm'),
        'package_source', source,
        'package_sha1', sha1,
        'package_size', size
    ) AS metadata
FROM
    rpm_packages
;
")

set -o pipefail
osqueryi --json "${Q[@]}" | \
jq -r '.[] | [.part, .vendor, .product, .version // "-", .release, .target_hw, .metadata] | @csv' | \
csv2cpe \
    -cpe_part=1 \
    -cpe_vendor=2 \
    -cpe_product=3 \
    -cpe_version=4 \
    -cpe_update=5 \
    -cpe_targethw=6 \
    -e=1 \
    -i=1 \
    -lower \
    -o=$'\t'
```

Similarly to the hosted software collectors, it's up to you to ship raw osquery JSON to a database or message queue, and execute jq and csv2cpe in the processing stage of the pipeline. Also, you'll likely want to record the hostname where the query was executed. Check out the system_info osquery table for details.

### Running software collectors

Process information alone is not very useful for vulnerability scanning. Moreover, you have to choose between collecting samples (a snapshot of ps) or hook up into the OS to track all process executions.

This data is expensive to collect, decorate (enrich with useful information), and move around - can be massive in size. Ask yourself whether this is really needed in your environment.

Nonetheless, following query is an example for osquery that can capture process information, reporting the RPM package where the binary comes from, along with process-related metadata.

Note: this query can take a few minutes to run depending on how many processes and packages your system have.

```bash
Q=("
SELECT
    'a' AS part,
    '' AS vendor,
    pkg.name AS product,
    pkg.version,
    pkg.release,
    pkg.arch AS target_hw,
    JSON_OBJECT(
        'package_name', (pkg.name || '-' || pkg.version || '-' || pkg.release || '.' || pkg.arch || '.rpm'),
        'package_source', pkg.source,
        'package_sha1', pkg.sha1,
        'package_size', pkg.size,
        'process_name', proc.name,
        'process_parent', proc.parent,
        'process_cwd', proc.cwd,
        'process_cmd', proc.cmdline,
        'process_pid', proc.pid,
        'process_start_time', proc.start_time
    ) AS metadata
FROM (
    SELECT * FROM processes WHERE path <> ''
) AS proc
JOIN (
    SELECT * FROM rpm_package_files
        WHERE package <> '' AND path <> ''
) AS pkg_files
ON
    proc.path = pkg_files.path
JOIN (
    SELECT * FROM rpm_packages WHERE name <> ''
) AS pkg
ON
    pkg_files.package = pkg.name
")

set -o pipefail
osqueryi --json "${Q[@]}" | \
jq -r '.[] | [.part, .vendor, .product, .version // "-", .release, .target_hw, .metadata] | @csv' | \
csv2cpe \
    -cpe_part=1 \
    -cpe_vendor=2 \
    -cpe_product=3 \
    -cpe_version=4 \
    -cpe_update=5 \
    -cpe_targethw=6 \
    -e=1 \
    -i=1 \
    -lower \
    -o=$'\t'
```

osquery also supports collecting information from docker containers, their networks, and images. This can be useful if you have an inventory of images in a managed registry.

Metadata can be used later to join against data from the hosted and/or installed software collectors.

## Processors

The main processor covered in this section is cpe2cve, the vulnerability matching processor.

Once collectors are producing data and CPEs are available (or can be built), the cpe2cve processor can perform CVEs matching and produce reports. The output of cpe2cve is always one CVE per line, regardless of whether the input was a single CPE or a group or CPEs.

Given the different inventories, you may want different vulnerability databases to process them. As previously mentioned, the public NVD database alone is generally not enough for good coverage. Specialized ecosystems (e.g. nodejs, ruby, python, php, go) require specialized databases.

### Vulnerability Databases

It is recommended to use multi-vendor databases. The cpe2cve processor require databases in the NVD CVE JSON 1.0 format, as files on disk. XML is also supported but discouraged, and likely to be deprecated - XML databases don't support the concept of version ranges, resulting in lower quality CVE matching and reporting.

On a system with multi-vendor databases, the maintainers of collectors should be able to define which database(s) to use to process their inventory. For example, the yum collector maintainer would pick the NVD database, but the nodejs collector maintainer would prefer a specialized database, e.g. snyk.

### Vulnerability Database: patches, snoozes, edits

It's not uncommon for processors to report false positives due to the quality of the inventory and databases, lack of normalization (missing vendors, wrong product names, bad versions).

Curating the data is the hardest part of maintaining a large system with multiple inventories and databases. Reporting high quality data is generally what makes the system successful.

Following are some methods that can help improve the data quality and end-user experience:

* Patch reports: allow the collectors to report patches applied to their source code; this can avoid reporting false positives by effectively `grep -v`'ing a list of patched CVEs from the processor output
* Snoozes: let users snooze certain CVEs, in the sense of not reporting them for a period of time or indefinitely; this can avoid reporting false positives consecutively
* Edits: some times the quality of the vulnerability database is subpar, missing information, or containing incorrect information; allowing edits to existing CVEs or creating new CVEs can improve the quality of matching and reports

### The [cpe2cve](https://github.com/facebookincubator/nvdtools/blob/master/cmd/cpe2cve) vulnerability processor

Using the cpe2cve vulnerability processor is pretty straightforward, but it's worth highlighting a few things:

* The quality of the vulnerability matching (CVE) results depend entirely on the quality of the CPE inventory and the vulnerability database being used (garbage in -> garbage out)
* Using specialized vulnerability databases for specific inventories can increase the quality of the results
* Processing one CPE alone may not yield all vulnerabilities; CVE databases use conditional logic (expressions) to match CPEs: if A and B or C
* Processing inventories from hosted software collectors generally process each CPE individually and does not account for their dependencies; putting that data together is not part of nvdtools
* Processing inventories from installed or running software collectors yield better results when all CPEs are grouped and processed in one batch; ideally with an operating system CPE (cpe:/o) in addition to all packages (cpe:/a)
* Consider whether you really need to process inventories from installed and running software using cpe2cve: this can be expensive depending on the size of your fleet; you may want to start by simply matching CPEs back to your hosted software inventories
* Use patching information, snoozes, and edits on top of the CVE database - in pre or post processing stages - to avoid false positives and consecutive false positives resulting in anger and disappointment from your users
* Use the metadata from collectors to build high quality reports for their maintainers, present data that know about (their own package names, not CPEs)

All the collector examples in previous sections put their generated CPE (or comma-separated list of CPEs) in the first column of their output. Their output has tab-separated columns. Those are also the default delimiters for the cpe2cve input (check --help).

With the examples above, an execution of the CVE processor could take CPE(s) from column 1 of the input, and insert CVE in the same column, pushing the original input one column forward:

```bash
cat inventory.csv | cpe2cve -cpe=1 -cve=1 /tmp/nvd/*.json.gz
```

Check out the [--help](https://github.com/facebookincubator/nvdtools/blob/master/cmd/cpe2cve/cpe2cve.go#L51) flag for all options related to input and output delimiters, lists, caching, and extra columns you may want to add to the output, such as CVSS score and CWE of each CVE.
