# leet 3.6.0

We’re excited to announce Fleet 3.6.0, which includes easier first-time setup, more flexible configuration, Amazon S3 integration for file carving, and more!

Let’s jump into the highlights…

- S3 buckets as file carving storage
- Build a Docker container with Fleet running as a non-root user
- Read in the MySQL password and JSON Web Token from a file

For the complete summary of changes check out the [release notes](https://github.com/fleetdm/fleet/releases/tag/3.6.0) on GitHub.

## Amazon S3 buckets as file carving storage

Thank you Matteo Piano from [Yelp](https://medium.com/u/469a0b267942?source=post_page-----53abbb94df6a--------------------------------)! This [awesome contribution](https://github.com/fleetdm/fleet/pull/126) adds the ability to set up Amazon S3 at the storage backend for file carving.

Prior to these changes, file carving in Fleet could only be saved to the Fleet database. A couple of concerns with this limitation [were surfaced by the [Fleet community](https://github.com/fleetdm/fleet/issues/111) and Matteo’s contribution kicks off the ability to add more backends in the future.

Check out the [new documentation](https://github.com/fleetdm/fleet/blob/master/docs/3-Deployment/2-Configuration.md#s3-file-carving-backend) on how to configure Fleet so file carving data is stored in an S3 bucket.

Build a Docker container with Fleet running as a non-root user
Shoutout to Ben Bornholm! The author of the sweet [holdmybeersecurity.com](https://holdmybeersecurity.com/) packed in two contributions for this release. [The first](https://holdmybeersecurity.com/) allows Fleet users to build the Docker container with Fleet running as a non-root user, an upgrade that aligns Fleet with Docker best practices.

Read in the MySQL password and JSON Web Token from a file
[The second of Ben’s contributions](https://github.com/fleetdm/fleet/pull/141) adds support to read in your MySQL password and JWT from a file. With this addition, Fleet users can avoid storing secrets in a static configuration file or in environment variables.

Using Docker secrets for supplying the above credentials is an example use case of reading in such credentials.

```
mysql:
  address: mysql:3306  
  database: fleet
  username: fleet
  password_path: /run/secrets/mysql-fleetdm-password
redis:
  address: redis:6379
server:
  address: 0.0.0.0:8080
  cert: /run/secrets/fleetdm-tls-cert
  key: /run/secrets/fleetdm-tls-key
auth:
  jwt_key_path: /run/secrets/fleetdm-jwt-key
filesystem:
  status_log_file: /var/log/osquery/status.log
  result_log_file: /var/log/osquery/result.log
  enable_log_rotation: true
logging:
  json: true
```
---

## Ready to update?

Visit our [update guide](https://fleetdm.com/docs/using-fleet/updating-fleet) in the Fleet docs for instructions on updating to Fleet 3.6.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2021-01-09">
<meta name="articleTitle" value="Fleet 3.6.0">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-3.6.0-cover-1600x900@2x.jpg">