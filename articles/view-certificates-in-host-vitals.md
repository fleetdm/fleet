# View certificates in host vitals

Fleet [v4.65.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.65.0) expands host vitals to include a list of certificates for macOS, iOS, and iPadOS hosts. This feature allows you to view the certificates installed on devices, helping you understand if a missing or expired certificate is the reason why an end user can't connect to the corporate network.

This guide introduces you to the certificates section in host vitals and explains how to access and interpret the certificate information.

## Prerequisites

* Fleet [v4.65.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.65.0) or greater.
* macOS, iOS, or iPadOS devices enrolled in Fleet.

## How does it work?

You'll find certificate information on the host vitals page under the **Details** tab, where they're displayed in **Certificates** at the bottom of the page. End users can also find this information for their device on their **My device** page accessible via Fleet Desktop.

The **Certificates** section displays the name of the certificate and its expiration date. There's also a status indicator for each certificate. The indicator is red if the certificate is expired, yellow if it expires within 30 days, and green if the certificate is valid for more than 30 days. Clicking on the certificate's row opens a modal with additional details about the certificate.

Fleet API users can access host certificate information via the "Get host's certificates" [endpoint](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-certificates).

For macOS hosts, Fleet retrieves certificate information using osquery's `certificates` [table](https://fleetdm.com/learn-more-about/certificates-query). For iOS and iPadOS hosts, Fleet retrieves certificates via MDM using the `CertificateList` [command](https://developer.apple.com/documentation/devicemanagement/certificate-list-command).

## Conclusion

The certificates section in host vitals provides you with a quick overview of the certificates installed on your macOS, iOS, and iPadOS devices. This feature helps you identify and troubleshoot certificate-related issues that may prevent your end users from connecting to the corporate network.

<meta name="articleTitle" value="View certificates in host vitals">
<meta name="authorFullName" value="Sarah Gillespie">
<meta name="authorGitHubUsername" value="gillespi314">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-03-04">
<meta name="description" value="Learn about certificates in host vitals">