## Tools for testing osquery

### Testing queries

Use [test-tables.sh](./test-tables.sh) to run an entire set of queries, outputting the results. This script will automatically read the queries from the input path provided (see [queries.txt](./queries.txt) for an example), and output results to stdout. It is likely useful to pipe the output to a text file, as in:

```sh
./test-tables.sh queries.txt > results.txt
```

OS tailored query tables are named:
- macOS.txt
- windows.txt
- linux.txt

The following flags should be set in the Fleet agent options before running the test tables:

```
Options:
  file_paths:
    foo:
      - /tmp/%%
  yara:
    file_paths:
      system_binaries:
        - sig_group_1
      tmp:
        - sig_group_1
        - sig_group_2
    signatures:
      sig_group_1:
        - /tmp/foo.sig
        - /tmp/bar.sig
      sig_group_2:
        - /Users/wxs/sigs/baz.sig

command_line_flags:
  disable_audit: false
  disable_events: false
  audit_allow_config: true
  enable_file_events: true
  audit_allow_user_events: true
  audit_allow_process_events: true
  enable_keyboard_events: true 
  enable_mouse_events: true
```

Additional Setup:

- carves: A file must be placed at /tmp/carve.txt
- crashes: User should manually crash a benign process with the following command from a terminal:
  `kill -3 <pid>`
- crontab: User will need to make a cronjob if none exist. A simple cronjob can be made with the
  `crontab -e` command and saving the following: `0 10 * * * /usr/bin/curl -s http://stackoverflow.com > ~/stackoverflow.html`
- hardware_events: A usb device will need to be plugged into the machine **after** the events tables
  have been enabled.
- kernel_panics: Place a kernel panic report in `/Library/Logs/DiagnosticReports. An example can be
  found in the artifacts folder. More reports can be found at https://github.com/osquery/osquery/pull/7585
- user_events: Allow remote login via system settings; ssh into local machine
- system_extensions: Download an extension from https://opalcamera.com if no extension exists on the device.


Flakey Tests:

- Not working on arm based macs:
  - ibridge_info 
  - memory_device_mapped_addresses
  - memory_error_info
  - quicklook_cache
  - startup_items
  - device_file
  - device_hash
  - device_partitions
