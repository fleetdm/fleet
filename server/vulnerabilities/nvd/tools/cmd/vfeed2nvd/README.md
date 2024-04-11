# `vfeed2nvd`

`vfeed2nvd` loads the vulnerability data from a local vfeed repo and converts it
into NVD format. The resulting file can be used as a feed in the
[`cpe2cve`](https://github.com/facebookincubator/nvdtools/tree/master/cmd/cpe2cve)
processor.

## Example: download all vulnerabilities

```bash
export VFEED_REPO_PATH=/usr/local/vfeed/data-json
./vfeed2nvd > vulns.json
```
