# MicroMDM touchless migration

This Go script reads information from a MicroMDM database and outputs:

- SCEP, APNs and ADE certificates.
- A file with MySQL statements to import the records into a Fleet database.

### Usage

The only requirement is to have a compatible version of Go installed, and a MicroMDM database.

The tool is its own Go module, so run it from this directory. The output files are written to the current directory.

Here's an example of how a successful run looks like:

```sh
$ cd tools/mdm/migration/micromdm/touchless
$ go run . --db ~/projects/micromdm/micromdm.db

2024/09/11 10:05:53 Open DB for devices
2024/09/11 10:05:53 Found 1 devices
2024/09/11 10:05:53 Wrote device/enrollment records to dump.sql
2024/09/11 10:05:53 Open DB for SCEP cert and key
2024/09/11 10:05:53 Wrote SCEP cert/key to scep.cert/scep.key
```

