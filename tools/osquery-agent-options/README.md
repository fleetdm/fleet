# osquery-agent-options

This directory contains a script (a Go command) that generates the struct needed to unmarshal the Agent Options' `options` values that the current version of osquery supports. It extracts this information from `osqueryd --help` to identify which osquery command-line flags can be set via the options and which are only for the command-line (i.e. require a restart), and running a query in `osqueryi` (`osqueryd -S`) to get the data type of those options.

It writes the resulting Go code to stdout (the `osqueryOptions` and the `osqueryCommandLineFlags` structs) to a file provided as argument.

This command only supports macOS.

Whenever there's a new version of osquery, just update the variable `osqueryVersion`.

## OS-specific flags

Some osquery flags are OS-specific and will not show up either with `osqueryd --help` or with the `osqueryi` query, depending on the OS you're running those on. In the code (in `server/fleet/agent_options.go`), those OS-specific flags are defined in the `OsqueryCommandLineFlags{Linux,MacOS,Windows}` structs, and the `osquery-agent-options` tool will automatically ignore from its generated struct any flag already defined as part of one of the OS-specific structs.

It can be hard to even know what OS-specific flags exist, because of the fact they don't show up in `osqueryd --help` or the `osqueryi` query when not running that specific OS, and the fact that not all flags are documented in [the osquery docs](https://osquery.readthedocs.io/en/stable/). To help with this, the following bash command can be executed assuming you have the osquery repository cloned locally and checked out to the latest release version:

```
# ag is the Silver Searcher, a grep alternative, but it should work with grep too, maybe
# with some small adjustments to the flags.
$ ag --nofilename -o 'FLAGS_[a-z0-9_]+' ./osquery/ ./plugins/ | sort | uniq | gcut -d _ --complement -f 1
```

This finds all flags defined in the osquery codebase (assuming all flags are built the same way). It is then possible to run a diff of this list with the list from the `osqueryi` query (e.g. `osqueryi --list 'select name from osquery_flags;'`), and the missing ones are _possibly/likely_ OS-specific. It's not an automatable task, as some judgement and manual code inspection may be necessary (some flags may be just in a test file, there may be some false-positives like `FLAGS_start` and `FLAGS_end` that are only sentinel values, the code line may be commented-out, etc.), but at least it gives a list of potential such flags.

To help with the future updates to those osquery flags, the output of this shell pipe is saved to a file that is included in this directory under the name `osquery_<version>_codeflags.txt`. Please store this output for each osquery version that we process for new flags, as it allows diffing the new output with the one from the previous version and quickly know if there was any new or deleted flags.
