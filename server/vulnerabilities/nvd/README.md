# Testing CPE Translations

To improve accuracy when [mapping software to CVEs](../../../docs/Using%20Fleet/Vulnerability-Processing.md), we can add data to [cpe_translations.json](./cpe_translations.json) which
will get picked up by the NVD repo.

To test these changes locally, you can:

1. make the [appropriate](../../../docs/Using%20Fleet/Vulnerability-Processing.md#Improving-accuracy) changes to cpe_translations

2. host this file on a local web server

    ```bash
    ./tools/file-server 8082 ./server/vulnerabilities/nvd/cpe_translations.json
    ```

3. (re)launch your local fleet server with the following `--config`

    ```yaml
    vulnerabilities:
    cpe_translations_url: "http://localhost:8082/cpe_translations.json"
    ```

4. trigger the vulnerabilities scan

    ```bash
    fleetctl trigger --name vulnerabilities
    ```
