## File carving

Fleet supports osquery's file carving functionality as of Fleet 3.3.0. This allows the Fleet server to request files (and sets of files) from Fleet's agent (fleetd) returning the full contents to Fleet.

File carving data can be either stored in Fleet's database or to an external S3 bucket. For information on how to configure the latter, consult the [configuration docs](https://fleetdm.com/docs/deploying/configuration#s-3-file-carving-backend).

### Configuration

Given a working flagfile for connecting fleetd to Fleet, add the following flags to enable carving:

```sh
--disable_carver=false
--carver_disable_function=false
--carver_start_endpoint=/api/v1/osquery/carve/begin
--carver_continue_endpoint=/api/v1/osquery/carve/block
--carver_block_size=8000000
```

The default flagfile provided in the "Add new host" dialog also includes this configuration.

#### Carver block size

The `carver_block_size` flag should be configured in osquery.

For the (default) MySQL Backend, the configured value must be less than the value of
`max_allowed_packet` in the MySQL connection, allowing for some overhead. The default for [MySQL 8](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_max_allowed_packet) it is 64MB.

For the S3/Minio backend, this value must be set to at least 5MiB (`5242880`) due to the
[constraints of S3's multipart
uploads](https://docs.aws.amazon.com/AmazonS3/latest/dev/qfacts.html).

#### Compression

Compression of the carve contents can be enabled with the `carver_compression` flag in osquery. When used, the carve results will be compressed with [Zstandard](https://facebook.github.io/zstd/) compression.

### Usage

File carves are initiated with osquery queries. Issue a query to the `carves` table, providing `carve = 1` along with the desired path(s) as constraints.

For example, to extract the `/etc/hosts` file on a host with hostname `mac-workstation`:

```sh
fleetctl query --hosts mac-workstation --query 'SELECT * FROM carves WHERE carve = 1 AND path = "/etc/hosts"'
```

The standard osquery file globbing syntax is also supported to carve entire directories or more:

```sh
fleetctl query --hosts mac-workstation --query 'SELECT * FROM carves WHERE carve = 1 AND path LIKE "/etc/%%"'
```

#### Retrieving carves

List the non-expired (see below) carves with `fleetctl get carves`. Note that carves will not be available through this command until osquery checks in to the Fleet server with the first of the carve contents. This can take some time from initiation of the carve.

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

#### Expiration

Carve contents remain available for 24 hours after the first data is provided from the osquery client. After this time, the carve contents are cleaned from the database and the carve is marked as "expired".

The same is not true if S3 is used as the storage backend. In that scenario, it is suggested to setup a [bucket lifecycle configuration](https://docs.aws.amazon.com/AmazonS3/latest/dev/object-lifecycle-mgmt.html) to avoid retaining data in excess. Fleet, in an "eventual consistent" manner (i.e. by periodically performing comparisons), will keep the metadata relative to the files carves in sync with what it is actually available in the bucket.

### Alternative carving backends

#### MinIO

Configure the following:
- `FLEET_S3_ENDPOINT_URL=minio_host:port`
- `FLEET_S3_BUCKET=minio_bucket_name`
- `FLEET_S3_SECRET_ACCESS_KEY=your_secret_access_key`
- `FLEET_S3_ACCESS_KEY_ID=acces_key_id`
- `FLEET_S3_FORCE_S3_PATH_STYLE=true`
- `FLEET_S3_REGION=minio` or any non-empty string otherwise Fleet will attempt to derive the region.

If you're testing file carving locally with the docker-compose environment, the `--dev` flag on Fleet server will
automatically point carves to the local MinIO container and write to the `carves-dev` bucket without needing to set
additional configuration. Note that this bucket is *not* created automatically when bringing MinIO up; you'll need to
log in via `http://localhost:9001` with credentials `minio` / `minio123!` to create the bucket.

### Troubleshooting

#### Check carve status in osquery

Osquery can report on the status of carves through queries to the `carves` table.

The details provided by

```sh
fleetctl query --labels 'All Hosts' --query 'SELECT * FROM carves'
```

can be helpful to debug carving problems.

#### Ensure `carver_block_size` is set appropriately

`carver_block_size` is an osquery flag that sets the size of each part of a file carve that osquery
sends to the Fleet server.

When using the MySQL backend (default), this value must be less than the `max_allowed_packet`
setting in MySQL. If it is too large, MySQL will reject the writes.

When using S3, the value must be at least 5MiB (5242880 bytes), as smaller multipart upload
sizes are rejected. Additionally, [S3
limits](https://docs.aws.amazon.com/AmazonS3/latest/userguide/qfacts.html) the maximum number of
parts to 10,000.

The value must be small enough that HTTP requests do not time out.

Start with a default of 2MiB for MySQL (2097152 bytes), and 5MiB for S3/Minio (5242880 bytes).

<meta name="pageOrderInSection" value="2100">
