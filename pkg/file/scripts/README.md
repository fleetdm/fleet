### File scripts

This folder contains scripts to install/remove software for different types of installers.

Scripts are stored on their own files for two reasons:

1. Some of them are read and displayed in the UI.
2. It's helpful to have good syntax highlighting and easy ways to run them.

#### Variables

Because the scripts are shared between Go and JS, the convention is to declare variables using `$VAR_NAME` and document its intended usage here.

- `$INSTALLER_PATH` path to the installer file.

