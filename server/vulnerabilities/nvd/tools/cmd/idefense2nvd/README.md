# `idefense2nvd`

`idefense2nvd` downloads the vulnerability data from Idefense and converts it into NVD format. The resulting file can be used as a feed in [`cpe2cve`](https://github.com/facebookincubator/nvdtools/tree/master/cmd/cpe2cve) processor

## Example: download all vulnerabilities since 2h ago

```bash
export IDEFENSE_TOKEN=token
./idefense2nvd -download -since 2h > vulns.json 
```