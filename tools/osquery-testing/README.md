## Tools for testing osquery

### Testing queries

Use [test-tables.sh](./test-tables.sh) to run an entire set of queries, outputting the results. This script will automatically read the queries from the input path provided (see [queries.txt](./queries.txt) for an example), and output results to stdout. It is likely useful to pipe the output to a text file, as in:

```sh
./test-tables.sh queries.txt > results.txt
```
