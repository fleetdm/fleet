# File carving

Fleet supports file carving, which allows you to request files (and sets of files) and their full contents from hosts.

File carving data can be either stored in Fleet's database or to an external S3 bucket. For information on how to configure the latter, consult the [configuration docs](https://fleetdm.com/docs/deploying/configuration#s-3-file-carving-backend).

## Setup

In your agent configuration, add the following [command line flags](https://fleetdm.com/docs/configuration/agent-configuration#options-and-command-line-flags) to enable carving:

```yaml
  disable_carver: false
  carver_disable_function: false
  carver_start_endpoint: /api/v1/osquery/carve/begin
  carver_continue_endpoint: /api/v1/osquery/carve/block
  carver_block_size: 8000000
```

The configured `carver_block_size` must be less than the value of `max_allowed_packet` in the MySQL connection, allowing for some overhead. The default for [MySQL 8](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_max_allowed_packet) is 64MB (`67108864`).

For the S3-compatible backend, `carver_block_size` must be set to at least 5MiB (`5242880`) due to the [constraints of S3's multipart
uploads](https://docs.aws.amazon.com/AmazonS3/latest/dev/qfacts.html).

Compression of the carve contents can be enabled with the `carver_compression` flag. When used, the carve results will be compressed with [Zstandard](https://facebook.github.io/zstd/) compression.

## Create carves

> File carving can cause significant performance impact if multiple factors are scaled up simultaneously. To avoid overloading your Fleet instance:
> - Target a narrow host set. Avoid running carves against all hosts.
> - Use specific paths. Avoid wildcard paths (e.g. /tmp/* or user home directories) that may match many or large files.
> - Mind the limits. Individual files must be under 8 GB
> - Avoid scheduled carves on broad targets. Automations that repeat carves against large host sets compound the load over time.
> The total load scales as the product of: number of hosts × number of paths × number of matching files × average file size. Any one of these can be large in isolation, but all four at once can result in millions of database writes and terabytes of S3 data simultaneously.

File carves are initiated with live reports. Run live report using the `carves` table, providing `carve = 1` along with the desired path(s) as constraints.

For example, to extract the `/etc/hosts` file on a host with hostname `mac-workstation`:

```sh
fleetctl report --hosts mac-workstation --query 'SELECT * FROM carves WHERE carve = 1 AND path = "/etc/hosts"'
```

Glob syntax is also supported to carve entire directories or more:

```sh
fleetctl report --hosts mac-workstation --query 'SELECT * FROM carves WHERE carve = 1 AND path LIKE "/etc/%%"'
```

## Retrieve carves

List the non-expired (see below) carves with `fleetctl get carves`. Note that carves will not be available through this command until Fleet's agent (fleetd) checks in to the Fleet server with the first of the carve contents. This can take some time from initiation of the carve.

To also retrieve expired carves, use `fleetctl get carves --expired`.

Contents of carves are returned as .tar archives, and compressed if that option is configured.

To download the contents of a carve with ID 3, use

```sh
fleetctl get carve --outfile carve.tar 3
```

It can also be useful to pipe the results directly into the tar command for unarchiving:

```sh
fleetctl get carve --stdout 3 | tar -x
```

## Expiration

Carve contents remain available for 24 hours after the first data is provided from Fleet's agent (fleetd). After this time, the carve contents are cleaned from the database, and the carve is marked as "expired".

The same is not true if S3 is used as the storage backend. In that scenario, it is suggested to set up a [bucket lifecycle configuration](https://docs.aws.amazon.com/AmazonS3/latest/dev/object-lifecycle-mgmt.html) to avoid retaining data in excess. Fleet, in an "eventual consistent" manner (i.e., by periodically performing comparisons), will keep the metadata relative to the files carves in sync with what is actually available in the bucket.

## Alternative carving backends

#### RustFS

Configure the following:
- `FLEET_S3_ENDPOINT_URL=rustfs_host:port`
- `FLEET_S3_BUCKET=bucket_name`
- `FLEET_S3_SECRET_ACCESS_KEY=your_secret_access_key`
- `FLEET_S3_ACCESS_KEY_ID=access_key_id`
- `FLEET_S3_FORCE_S3_PATH_STYLE=true`
- `FLEET_S3_REGION=localhost` or any non-empty string otherwise Fleet will attempt to derive the region.

If you're testing file carving locally, the `--dev` flag on Fleet server will automatically point carves to the local RustFS container and write to the `carves-dev` bucket (created automatically) without needing to set additional configuration.

## Troubleshooting

### Check carve status

You can report on the status of carves through queries to the `carves` table.

You can debug carving problems with:

```sh
fleetctl report --labels 'All Hosts' --query 'SELECT * FROM carves'
```


### Ensure `carver_block_size` is set appropriately

`carver_block_size` is an option that sets the size of each part of a file carve that Fleet's agent (fleetd) sends to the Fleet server.

When using the MySQL backend (default), this value must be less than the `max_allowed_packet` setting in MySQL. If it is too large, MySQL will reject the writes.

When using S3, the value must be at least 5MiB (5242880 bytes), as smaller multipart upload
sizes are rejected. Additionally, [S3 limits](https://docs.aws.amazon.com/AmazonS3/latest/userguide/qfacts.html) the maximum number of
parts to 10,000.

The value must be small enough that HTTP requests do not time out.

Start with a default of 2MiB for MySQL (2097152 bytes), and 5MiB for S3 (5242880 bytes).

<meta name="articleTitle" value="File carving in Fleet"> 
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-06-10">
<meta name="description" value="Learn how file carving allows you to request files and their contents from hosts">
<meta name="category" value="guides">
