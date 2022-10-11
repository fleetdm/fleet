# osquery-agent-options

This directory contains a script (a Go command) that generates the struct needed to unmarshal the Agent Options' `options` values that the current version of osquery supports. It extracts this information from `osqueryd --help` to identify which osquery command-line flags can be set via the options and which are only for the command-line (i.e. require a restart), and running a query in `osqueryi` to get the data type of those options.

It prints the resulting Go code to stdout (the `osqueryOptions` and the `osqueryCommandLineFlags` structs), you can just copy it and insert it in the proper location in the source code to replace the existing struct (in `server/fleet/agent_options.go`).

Note that the latest version of osquery should be installed for this tool to work properly (`osqueryd` and `osqueryi` must be in your $PATH).
