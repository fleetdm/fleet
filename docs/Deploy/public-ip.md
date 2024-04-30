# Public IPs of devices

Fleet attempts to deduce the public IP of devices from well-known HTTP headers received on requests made by Fleet's agent (fleetd).

The HTTP request headers are checked in the following order:
1. If `True-Client-IP` header is set, then Fleet will extract its value.
2. If `X-Real-IP` header is set, then Fleet will extract its value.
3. If `X-Forwarded-For` header is set, then Fleet will extract the first comma-separated value.
4. If none of the above headers are present in the HTTP request then Fleet will attempt to use the remote address of the TCP connection (note that on deployments with ingress proxies the remote address seen by Fleet is the IP of the ingress proxy).

<meta name="title" value="Public IPs">
<meta name="pageOrderInSection" value="800">
<meta name="description" value="Learn how to configure proxy settings for Fleet.">
<meta name="navSection" value="TBD">