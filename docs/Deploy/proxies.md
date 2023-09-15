# Using a proxy

If you are in an enterprise environment where Fleet is behind a proxy and you would like to be able to retrieve vulnerability data for [Vulnerability Processing](https://fleetdm.com/docs/using-fleet/vulnerability-processing#vulnerability-processing), it may be necessary to configure the proxy settings. Fleet automatically uses the `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables.

For example, to configure the proxy in a systemd service file:

```
[Service]
Environment="HTTP_PROXY=http(s)://PROXY_URL:PORT/"
Environment="HTTPS_PROXY=http(s)://PROXY_URL:PORT/"
Environment="NO_PROXY=localhost,127.0.0.1,::1"
```

After modifying the configuration you will need to reload and restart the Fleet service, as explained above.

<meta name="title" value="Proxies">
<meta name="pageOrderInSection" value="800">
<meta name="description" value="Learn how to configure proxy settings for Fleet.">
<meta name="navSection" value="TBD">
