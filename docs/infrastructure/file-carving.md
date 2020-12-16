# File Carving with Fleet

Fleet supports osquery's file carving functionality as of Fleet 3.3.0. This allows the Fleet server to request files (and sets of files) from osquery agents, returning the full contents to Fleet.

File carving data can be either stored in Fleet's database or to an external S3 bucket. For information on how to configure the latter, consult the [configuration docs](https://github.com/fleetdm/fleet/blob/master/docs/infrastructure/configuring-the-fleet-binary.md#s3-file-carving-backend).

## Configuration

Given a working flagfile for connecting osquery agents to Fleet, add the following flags to enable carving:

```
--disable_carver=false
--carver_start_endpoint=/api/v1/osquery/carve/begin
--carver_continue_endpoint=/api/v1/osquery/carve/block
--carver_block_size=2000000
```

The default flagfile provided in the "Add New Host" dialog also includes this configuration.

### Carver Block Size

The `carver_block_size` flag should be configured in osquery. 2MB (`2000000`) is a good starting value.

The configured value must be less than the value of `max_allowed_packet` in the MySQL connection, allowing for some overhead. The default for MySQL 5.7 is 4MB and for MySQL 8 it is 64MB.

In case S3 is used as the storage backend, this value must be instead set to be at least 5MB due to the [constraints of S3's multipart uploads](https://docs.aws.amazon.com/AmazonS3/latest/dev/qfacts.html).

Using a smaller value for `carver_block_size` will lead to more HTTP requests during the carving process, resulting in longer carve times and higher load on the Fleet server. If the value is too high, HTTP requests may run long enough to cause server timeouts.

### Compression

Compression of the carve contents can be enabled with the `carver_compression` flag in osquery. When used, the carve results will be compressed with [Zstandard](https://facebook.github.io/zstd/) compression.

## Usage

File carves are initiated with osquery queries. Issue a query to the `carves` table, providing `carve = 1` along with the desired path(s) as constraints.

For example, to extract the `/etc/hosts` file on a host with hostname `mac-workstation`:

```
fleetctl query --hosts mac-workstation --query 'SELECT * FROM carves WHERE carve = 1 AND path = "/etc/hosts"'
```

The standard osquery file globbing syntax is also supported to carve entire directories or more:
```
fleetctl query --hosts mac-workstation --query 'SELECT * FROM carves WHERE carve = 1 AND path LIKE "/etc/%%"'
```

### Retrieving Carves

List the non-expired (see below) carves with `fleetctl get carves`. Note that carves will not be available through this command until osquery checks in to the Fleet server with the first of the carve contents. This can take some time from initiation of the carve.

To also retrieve expired carves, use `fleetctl get carves --expired`.

Contents of carves are returned as .tar archives, and compressed if that option is configured.

To download the contents of a carve with ID 3, use

```
fleetctl get carve 3 --outfile carve.tar
```

It can also be useful to pipe the results directly into the tar command for unarchiving:

```
fleetctl get carve 3 --stdout | tar -x
```

### Expiration

Carve contents remain available for 24 hours after the first data is provided from the osquery client. After this time, the carve contents are cleaned from the database and the carve is marked as "expired".

The same is not true if S3 is used as the storage backend. In that scenario, it is suggested to setup a [bucket lifecycle configuration](https://docs.aws.amazon.com/AmazonS3/latest/dev/object-lifecycle-mgmt.html) to avoid retaining data in excess. Fleet, in an "eventual consistent" manner (i.e. by periodically performing comparisons), will keep the metadata relative to the files carves in sync with what it is actually available in the bucket.

## Troubleshooting

### Check carve status in osquery

Osquery can report on the status of carves through queries to the `carves` table.

The details provided by

```
fleetctl query --labels 'All Hosts' --query 'SELECT * FROM carves'
```

can be helpful to debug carving problems.

### Ensure  `carver_block_size` is set appropriately

This value must be less than the `max_allowed_packet` setting in MySQL. If it is too large, MySQL will reject the writes.

The value must be small enough that HTTP requests do not time out.

