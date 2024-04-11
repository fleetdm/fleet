# `flexera2nvd`

`flexera2nvd` downloads the vulnerability data from Flexera and converts it into NVD format. The resulting file can be used as a feed in [`cpe2cve`](https://github.com/facebookincubator/nvdtools/tree/master/cmd/cpe2cve) processor

## Example: download all vulnerabilities since 2h ago

```bash
export FLEXERA_TOKEN=token
./flexera2nvd -download -since 2h > vulns.json
```