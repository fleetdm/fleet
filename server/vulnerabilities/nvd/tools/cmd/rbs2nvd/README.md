# `rbs2nvd`

`rbs2nvd` downloads the vulnerability data from Risk Based Security and converts it into NVD format. The resulting file can be used as a feed in [`cpe2cve`](https://github.com/facebookincubator/nvdtools/tree/master/cmd/cpe2cve) processor

## Example: download all vulnerabilities since 2h ago

```bash
export RBS_CLIENT_ID=client_id
export RBS_CLIENT_SECRET=client_secret
./rbs2nvd -since 2h > vulns.json
```
