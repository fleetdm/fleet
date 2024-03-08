# nvdvuln

This tool can be used to reproduce false positive/negative vulnerabilities found by Fleet.

The tool has two modes of operation:
1. Run vulnerability processing using the NVD dataset on a specific software item. Such software item should be specified to the tool with the fields as stored in Fleet's `software` MySQL table.
2. Fetch software from a Fleet instance (and their found vulnerabilities), then, run vulnerability processing on such software and report any differences in CVEs against the Fleet instance. This mode of operation is useful to test new changes to the vulnerability processing.

PS: This tool is only useful on systems and software where the NVD dataset is used to detect vulnerabilities. For instance, this tool should not be used with Microsoft Office applications for macOS because Fleet uses a different dataset to detect vulnerabilities on such applications.

## Example Mode 1

```sh
go run -tags fts5 ./tools/nvdvuln \
    -software_name Python.app \
    -software_version 3.7.3 \
    -software_source apps \
    -software_bundle_identifier com.apple.python3 \
    -sync \
    -db_dir /tmp/vulndbtest
[...]
CVEs found for Python.app (3.7.3): CVE-2007-4559, CVE-2019-10160, CVE-2019-15903, CVE-2022-0391,
CVE-2020-14422, CVE-2020-10735, CVE-2023-40217, CVE-2015-20107, CVE-2016-3189, CVE-2018-25032,
CVE-2019-20907, CVE-2019-9740, CVE-2020-8315, CVE-2019-16056, CVE-2021-3177, CVE-2021-23336,
CVE-2022-48560, CVE-2022-45061, CVE-2019-18348, CVE-2019-16935, CVE-2019-9947, CVE-2021-4189,
CVE-2021-3426, CVE-2022-48566, CVE-2021-3733, CVE-2022-48564, CVE-2023-24329, CVE-2023-27043,
CVE-2019-12900, CVE-2021-28861, CVE-2023-36632, CVE-2022-48565, CVE-2019-9948, CVE-2020-8492,
CVE-2020-27619, CVE-2020-26116, CVE-2021-3737, CVE-2022-37454
```

## Example Mode 2

```sh
go run -tags fts5 ./tools/nvd/nvdvuln/nvdvuln.go \
    -debug \
    -sync \
    -db_dir /tmp/vulndbtest \
    -software_from_url https://fleet.example.com \
    -software_from_api_token <...>
```

## CPU and memory usage

> Requirement: gnuplot (`brew install gnuplot`)

If set to `-debug` mode, the `nvdvuln` tool will sample its CPU and memory usage and store them on a file under the `-db_dir`.
Such data can be visualized with the following command:
```sh
./tools/nvd/nvdvuln/gnuplot.sh /path/to/db/directory
```