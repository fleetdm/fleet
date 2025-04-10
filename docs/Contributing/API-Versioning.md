# API versioning

All of the [Fleet API routes]([url](https://fleetdm.com/docs/rest-api/rest-api)) currently include `v1` in the URL path to identify which iteration of the API is being used. This allows us to release new features, improvements, or breaking changes in the future under a different version (like `/v2/`) without disrupting existing clients that still rely on the previous version. By versioning our API, we can manage changes in a predictable, structured way. This approach provides a clear upgrade path for developers and prevents confusion by standardizing how changes are communicated, documented, and ultimately adopted by API consumers.

<meta name="pageOrderInSection" value="900">
<meta name="description" value="Learn about how and why the Fleet API is versioned.">
