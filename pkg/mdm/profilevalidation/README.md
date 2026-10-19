This package contains types and functions for validating MDM profiles on Apple and Windows devices.

It works by parsing the respective sources via tool (TODO: Insert tool link here), and parsing that into a Fleet defined schema type for further validation and processing.

The output of this process lands in outputs/profile_schema.json, which is embedded into the Go code and parsed into JSON struct on startup.