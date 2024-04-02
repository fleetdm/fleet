# Testing CPE Translations

To improve accuracy when [mapping software to CVEs](../../../docs/Using%20Fleet/Vulnerability-Processing.md), we can add data to [cpe_translations.json](./cpe_translations.json) which
will get picked up by the NVD repo.

To test these changes locally, you can:

1. make the [appropriate](../../../docs/Using%20Fleet/Vulnerability-Processing.md#Improving-accuracy) changes to cpe_translations

2. host this file on a local web server

    ```bash
    go run ./tools/file-server/main.go 8082 ./server/vulnerabilities/nvd/
    ```

3. (re)launch your local fleet server with one of the following

    Config method
    ```yaml
    vulnerabilities:
    cpe_translations_url: "http://localhost:8082/cpe_translations.json"
    ```
    
    Environment method
    ```bash
    FLEET_VULNERABILITIES_CPE_TRANSLATIONS_URL="http://localhost:8082/cpe_translations.json" ./build/fleet serve --dev --dev_license --logging_debug
    ```

4. trigger a vulnerabilities scan
    ```bash
    fleetctl trigger --name vulnerabilities
    ```
