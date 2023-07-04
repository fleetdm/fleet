# Fleet osquery extensions without fleetd

If you are interested in getting some of the `fleetd` tables but cannot run `fleetd` natively then its possible
to utilize this "fleetd_tables" extension with standalone `osqueryd`.

### Building the extension

First run (note `.ext` is required for osquery):
```shell
go build -o fleetd_tables.ext fleetd_tables.go
```

or using the Makefile
```shell
make fleetd-tables-linux
```

Then move it somewhere `osqueryd` can load it:
```shell
sudo cp fleetd_tables.ext /usr/local/osquery_extensions
```

And tell `osqueryd` to autoload your extension
```shell
echo "/usr/local/osquery_extensions/fleetd_tables.ext" > /tmp/extensions.load
```

Finally, launch `osqueryd`
```shell
sudo osqueryd --extensions_autoload=/tmp/extensions.load
```

### Local testing

Obtain the extensions_socket
```shell
osqueryi --nodisable_extensions
osquery> select value from osquery_flags where name = 'extensions_socket';
+-----------------------------------+
| value                             |
+-----------------------------------+
| /Users/USERNAME/.osquery/shell.em |
+-----------------------------------+
```

Then run the app
```shell
go run ./fleetd_tables.go --socket /Users/USERNAME/.osquery/shell.em
```

Or you can build the app and have `osqueryi` load it
```shell
go build -o fleetd_tables.ext fleetd_tables.go
osqueryi --extension /path/to/fleetd_tables.ext
```

