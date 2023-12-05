# Catch missed authorization checks during software development

<div class="video-container" style="position: relative; width: 100%; padding-bottom: 56.25%; margin-top: 24px; margin-bottom: 40px;">
	<iframe class="video" style="position: absolute; top: 0; left: 0; width: 100%; height: 100%; border: 0;" src="https://www.youtube.com/embed/jbkPLQpzPtc?si=k1BUb98QWRT1V8fZ" allowfullscreen></iframe>
</div>

Authorization is giving permission to a user to do an action on the server. As developers, we must ensure that users are only allowed to do what they are authorized.

One way to ensure that authorization has happened is to loudly flag when it hasn’t. This is how we do it at [Fleet Device Management](https://www.linkedin.com/company/fleetdm/?lipi=urn%3Ali%3Apage%3Ad_flagship3_pulse_read%3BCaXkx0wxSNeQ8WfF5SZ17g%3D%3D).

In our code base, we use the [go-kit library](https://github.com/go-kit/kit). Most of the general endpoints are created in the handler.go file. For example:
```
// user-authenticated endpoints
ue := newUserAuthenticatedEndpointer(svc, opts, r, apiVersions...)

ue.POST("/api/_version_/fleet/trigger", triggerEndpoint, triggerRequest{})
```

Every endpoint calls **kithttp.NewServer** and wraps the endpoint with our **AuthzCheck**. From [handler.go](https://github.com/fleetdm/fleet/blob/36421bd5055d37a4c39a04e0f9bd96ad47951131/server/service/handler.go#L729):
```
e = authzcheck.NewMiddleware().AuthzCheck()(e)
return kithttp.NewServer(e, decodeFn, encodeResponse, opts...)
```
![Example check](../website/assets/images/articles/catch-missed-authorization-checks-during-software-development-720x179@2x.jpg
"Example check")

This means that after the business logic is processed, the AuthzCheck is called. This check ensures that authorization was checked. Otherwise, an error is returned. From [authzcheck.go](https://github.com/fleetdm/fleet/blob/36421bd5055d37a4c39a04e0f9bd96ad47951131/server/service/middleware/authzcheck/authzcheck.go#L51):
```
// If authorization was not checked, return a response that will
// marshal to a generic error and log that the check was missed.
if !authzctx.Checked() {
	// Getting to here means there is an authorization-related bug in our code.
	return nil, authz.CheckMissingWithResponse(response)
}
```

This additional check is useful during our development and QA process, to ensure that authorization always happens in our business logic.


<meta name="articleTitle" value="Catch missed authorization checks during software development">
<meta name="authorFullName" value="Victor Lyuboslavsky">
<meta name="authorGitHubUsername" value="getvictor">
<meta name="category" value="guides">
<meta name="publishedOn" value="2023-12-04">
<meta name="description" value="How to perform authorization checks in a golang codebase for cybersecurity">
