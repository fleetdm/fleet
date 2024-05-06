### File scripts

This folder contains scripts to install/remove software for different types of installers.

Scripts are stored on their own files for two reasons:

1. Some of them are read and displayed in the UI.
2. It's helpful to have good syntax highlighting and easy ways to run them.

#### Variables

The scripts in this folder accept variables like `$VAR_NAME` that will be replaced/populated by `fleetd` when they run.

Supported variables are:

- `$INSTALLER_PATH` path to the installer file.

