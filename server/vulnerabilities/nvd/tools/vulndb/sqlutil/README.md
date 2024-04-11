# sqlutil

Package sqlutil provides utilities for Go's database/sql for dealing with SQL queries and database records.

Supports the following features:

- Mapping structs to sets of columns ([]string) and values ([]interface{})
- Using structs to build queries with bindings for safe Exec and Query
- Binding struct fields to query response records
- Embedding SQL schema as Go code along with an InitSchema helper function
