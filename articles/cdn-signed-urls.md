# How to use CloudFront signed URLs with Fleet

Fleet [v4.63.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.63.0) allows you to use CloudFront signed URLs for downloading MDM bootstrap packages and software installation packages to your hosts. CloudFront signed URLs grant access to a specific CloudFront distribution resource and are valid for a specified duration. Using CloudFront CDN (content delivery network) should speed up downloads and reduce the load on your Fleet server, especially for globally distributed device fleets.

## Prerequisites

- Fleet v4.63.0
- Orbit v1.39.0 agent installed on hosts
- S3 bucket with CloudFront distribution and a signing key pair

To add a CloudFront distribution with a signer to your S3 bucket, follow the instructions in the [AWS documentation](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-trusted-signers.html) or the [How to securely serve private CDN content using CloudFront](https://victoronsoftware.com/posts/cloudfront-signed-urls/) guide written by one of our engineers.

## Configure Fleet server for S3 and CloudFront

To configure S3 and CloudFront in Fleet, use the [S3 server configuration options](https://fleetdm.com/docs/configuration/fleet-server-configuration#s-3). Set these options via the command line, environment variables, or a configuration file.

To enable CloudFront signed URLs, set the following options in your Fleet server configuration:

- `s3_software_installers_cloudfront_url`: The base URL of your CloudFront distribution, such as `https://d1234567890.cloudfront.net`.
- `s3_software_installers_cloudfront_key_pair_id`: The CloudFront signer's key pair ID, such as `K1HFGXOMBB6TFF`.
- `s3_software_installers_cloudfront_private_key`: The CloudFront signer's private key, such as `-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAz...`.

The `FLEET_S3_SOFTWARE_INSTALLERS_CLOUDFRONT_URL_SIGNING_PRIVATE_KEY` environment variable can be set from a file. On macOS, it requires [gnu-sed](https://formulae.brew.sh/formula/gnu-sed) (`gsed`) to replace newlines with `\n` characters.

```bash
export FLEET_S3_SOFTWARE_INSTALLERS_CLOUDFRONT_URL_SIGNING_PRIVATE_KEY=$(cat ./private_key.pem | gsed -z 's/\n/\\n/g')
```

Non-signed CDN URLs are not secure and are not supported.

## Use CloudFront signed URLs in Fleet

Once configured, Fleet will automatically use CloudFront signed URLs to install MDM bootstrap packages and software packages on your hosts. The signed URLs are generated on the fly and are valid for six hours.

If the Fleet server encounters an error while generating a signed URL for the bootstrap package, it will fall back to using the Fleet server's URL.

If the Orbit agent encounters an error while downloading a software package using a signed URL, it will retry the download using the Fleet server's URL.

To make sure that the signed URLs are working correctly, you can check the CloudFront logs (if enabled) as well as [APM](https://aws.amazon.com/what-is/application-performance-monitoring/) or Fleet server debug logs. In APM or Fleet server logs, you should NOT see devices downloading packages from the Fleet server's non-CDN API paths, such as:

- `GET /api/v1/fleet/bootstrap`
- `POST /api/fleet/orbit/software_install/package`

## Conclusion

Using CloudFront signed URLs with Fleet can help speed up downloads and reduce the load on your Fleet server. If you have any questions or need help configuring CloudFront signed URLs, please contact our [support team](https://fleetdm.com/contact).

<meta name="articleTitle" value="How to use CloudFront signed URLs with Fleet">
<meta name="authorFullName" value="Victor Lyuboslavsky">
<meta name="authorGitHubUsername" value="getvictor">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-01-24">
<meta name="description" value="A guide on using signed URLs with MDM bootstrap packages and software installers.">
