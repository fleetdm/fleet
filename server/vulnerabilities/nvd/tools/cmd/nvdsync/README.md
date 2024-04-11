# nvdsync

`nvdsync` is a command line tool for synchronizing vulnerability [data feeds from NVD](https://nvd.nist.gov/vuln/data-feeds) to a local directory.

Currently supports CVE and CPE feeds.

## How it works

For CVE feeds, nvdsync downloads the .meta files provided by NVD and compare them to a local copy of the same file. If the local file does not exist or the contents are different, then it stores the remote .meta file locally and downloads the corresponding feed file. When new files are downloaded, nvdsync validates their SHA256 of the uncompressed data against what's in the .meta file.

CPE feeds do not offer a .meta file thus nvdsync relies on the web server's etag http response header to know it's time to sync the local feeds. If a .etag file does not exist in the local directory it creates one and downloads the CPE feed then subsequent runs use the .etag file.

By default, nvdsync does not print any information out, except errors. In order to get more information please us -v=1 flags in the command line.

## Proxy

nvdsync uses a standard http client that assumes it can access NVD (or the configured upstream host) directory. In order to use proxies please set the http_proxy or https_proxy environment variables.

## Example: download NVD CVE feed in JSON to ~/feeds/json

```bash
./nvdsync -v 1 -cve_feed=cve-1.0.json.gz ~/feeds/json
I0820 09:15:56.270696 1197925 cve.go:217] checking meta file "nvdcve-1.0-2002.meta" for updates to "nvdcve-1.0-2002.json.gz"
I0820 09:15:56.270713 1197925 cve.go:252] downloading meta file "https://static.nvd.nist.gov/feeds/json/cve/1.0/nvdcve-1.0-2002.meta"
I0820 09:16:01.847147 1197925 cve.go:217] checking meta file "nvdcve-1.0-2003.meta" for updates to "nvdcve-1.0-2003.json.gz"
I0820 09:16:01.847168 1197925 cve.go:252] downloading meta file "https://static.nvd.nist.gov/feeds/json/cve/1.0/nvdcve-1.0-2003.meta"

... 14 lines skipped ...

I0820 09:16:26.833321 1197925 cve.go:217] checking meta file "nvdcve-1.0-2011.meta" for updates to "nvdcve-1.0-2011.json.gz"
I0820 09:16:26.833346 1197925 cve.go:252] downloading meta file "https://static.nvd.nist.gov/feeds/json/cve/1.0/nvdcve-1.0-2011.meta"
I0820 09:16:29.316286 1197925 cve.go:267] data file "nvdcve-1.0-2011.json.gz" needs update in "/home/dvl/feeds/json": local{LastModifiedDate:2018-07-28 03:33:26 -0400
-0400 Size:201819657 ZipSize:9353214 GzSize:9353078 SHA256:AAEE78FB567FA96CC4A654C432414D98B741014A8A410E980F200127FD90F430} != remote{LastModifiedDate:2018-08-15 03:4
8:06 -0400 -0400 Size:202676227 ZipSize:9409647 GzSize:9409511 SHA256:585251B440C894CAC1C96C45800D00488AC9EE82A46998797627E4937839FE03}
I0820 09:16:29.316352 1197925 cve.go:311] downloading data file "https://static.nvd.nist.gov/feeds/json/cve/1.0/nvdcve-1.0-2011.json.gz"

... more lines skipped ...
```