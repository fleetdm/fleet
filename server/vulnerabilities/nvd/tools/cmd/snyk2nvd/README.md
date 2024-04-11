# `snyk2nvd`

`snyk2nvd` downloads the vulnerability data from Snyk and converts it into NVD format. The resulting file can be used as a feed in [`cpe2cve`](https://github.com/facebookincubator/nvdtools/tree/master/cmd/cpe2cve) processor

## Example: download all vulnerabilities and convert them

```bash
SNYK_ID=id SNYK_READONLY_KEY=key ./snyk2nvd -download > snyk.json
./snyk2nvd -convert -language=golang snyk.json > snyk_golang.json
./snyk2nvd -convert -language=python snyk.json > snyk_python.json
...
```