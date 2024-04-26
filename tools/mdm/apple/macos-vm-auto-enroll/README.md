# macOS VM Auto Enroll

A script to automate the manual enrollment process in a macOS virtual machine.

## Usage

The script takes no arguments, but can be configured through three environment variables.

- `FLEET_ENROLL_SECRET` (required) The fleet enrollment secret
- `FLEET_URL` (required) The fleet base url
- `MACOS_ENROLLMENT_VM_NAME` (optional) The name of the VM. If nothing is specified, the default name is `enrollment-test`.
- `MACOS_ENROLLMENT_VM_IMAGE` (optional) The image to use for the VM. If nothing is specified, the default image is `ghcr.io/cirruslabs/macos-sonoma-base:latest`

The entire process from the generation of the `pkg` file to the installation is automated. The only part that requires user intervention is installing the MDM profile.

## Steps

The script goes through the following steps.

1. Change to the correct directory
2. Delete the old `pkg` file if one exists
3. Build a new `pkg` file using the supplied variables
4. Delete the existing VM if one with the same name exists
5. Create a new VM with the chosen name
6. Launch the VM
7. Copy the `pkg` file into the VM
8. Install the fleet and orbit
9. Fetch the MDM profile from the fleet server after registration is complete
10. Open the MDM profile, adding it to the profile list
11. Open the settings app to the profile page
12. [ACTION REQUIRED] The user has to double click on the new profile and then click `Enroll`.
13. Open a shell in the terminal running the script
14. Once the shell is exited, the VM process is reattached to the terminal
