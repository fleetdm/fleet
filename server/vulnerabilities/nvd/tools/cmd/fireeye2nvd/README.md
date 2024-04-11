# `fireeye2nvd`

`fireeye2nvd` downloads the vulnerability data from FireEye and converts it into NVD format. The resulting file can be used as a feed in [`cpe2cve`](https://github.com/facebookincubator/nvdtools/tree/master/cmd/cpe2cve) processor

## Example: download all vulnerabilities since 2h ago

```bash
export FIREEYE_PUBLIC_KEY=public_key
export FIREEYE_PRIVATE_KEY=private_key
./fireeye2nvd -download -since 2h > vulns.json
```