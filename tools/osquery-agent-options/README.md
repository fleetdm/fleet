# osquery-agent-options

This directory contains a script (a Go command) that generates the struct needed to unmarshal the Agent Options' `options` values that the current version of osquery supports. It extracts this information from `osqueryd --help` to identify which osquery command-line flags can be set via the options, and running a query in `osqueryi` to get the data type of those options.

It prints the resulting Go code to stdout, you can just copy it and insert it in the proper location in the source code.

Note that the latest version of osquery should be installed for this tool to work properly (`osqueryd` and `osqueryi` must be in your $PATH).
