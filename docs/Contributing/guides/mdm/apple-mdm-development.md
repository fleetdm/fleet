# Apple MDM Development Guide

This guide provides instructions for developing Apple Mobile Device Management (MDM) functionality in Fleet.

## Prerequisites

Before you begin developing Apple MDM functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of the [Apple MDM Protocol](https://developer.apple.com/documentation/devicemanagement)
- Access to Apple devices for testing (macOS, iOS)
- Apple Developer account (for creating certificates)

## Setting Up the Development Environment

### Certificates

Apple MDM requires several certificates for development and testing:

1. **MDM Vendor Certificate**: Used to sign enrollment profiles
   - For development, you can use a self-signed certificate
   - For production, you need a certificate from Apple

2. **APNS Certificate**: Used for sending push notifications
   - For development, you can use a development APNS certificate
   - For production, you need a production APNS certificate

3. **SCEP Certificate**: Used for device identity
   - Fleet includes a built-in SCEP server for development

### Setting Up Certificates

```bash
# Generate a self-signed MDM vendor certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout mdm-vendor.key -out mdm-vendor.crt

# Convert to PKCS#12 format
openssl pkcs12 -export -out mdm-vendor.p12 -inkey mdm-vendor.key -in mdm-vendor.crt
```

### Configuration

Configure Fleet for Apple MDM development:

1. Edit your Fleet configuration file to include the MDM certificates
2. Set the MDM server URL to your development server
3. Configure the SCEP server settings

## Development Workflow

### 1. Enrollment Profile Development

The enrollment profile is the starting point for MDM enrollment:

1. Create an enrollment profile with the required payloads
2. Sign the profile with the MDM vendor certificate
3. Test the profile on a device

Example enrollment profile structure:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
        <!-- MDM Payload -->
        <dict>
            <key>PayloadType</key>
            <string>com.apple.mdm</string>
            <!-- Other MDM payload keys -->
        </dict>
        <!-- SCEP Payload -->
        <dict>
            <key>PayloadType</key>
            <string>com.apple.security.scep</string>
            <!-- Other SCEP payload keys -->
        </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>Fleet MDM (Development)</string>
    <key>PayloadIdentifier</key>
    <string>com.fleetdm.mdm.enrollment</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>00000000-0000-0000-0000-000000000000</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>
```

### 2. MDM Command Development

MDM commands are used to manage devices:

1. Implement the command in the Fleet server
2. Test the command on an enrolled device
3. Handle the command response

Example command structure:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>DeviceInformation</string>
        <key>Queries</key>
        <array>
            <string>DeviceName</string>
            <string>OSVersion</string>
            <string>SerialNumber</string>
        </array>
    </dict>
    <key>CommandUUID</key>
    <string>00000000-0000-0000-0000-000000000000</string>
</dict>
</plist>
```

### 3. Profile Development

Configuration profiles are used to configure devices:

1. Create a configuration profile with the required payloads
2. Implement the InstallProfile command
3. Test the profile on an enrolled device

Example profile structure:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
        <!-- Configuration Payload -->
        <dict>
            <key>PayloadType</key>
            <string>com.apple.wifi.managed</string>
            <!-- Other payload keys -->
        </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>Wi-Fi Configuration</string>
    <key>PayloadIdentifier</key>
    <string>com.fleetdm.mdm.wifi</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>00000000-0000-0000-0000-000000000000</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>
```

## Testing

### Manual Testing

1. Enroll a test device using the enrollment profile
2. Execute MDM commands from the Fleet server
3. Verify the device responds correctly
4. Check the MDM logs on the device

### Automated Testing

Fleet includes automated tests for Apple MDM functionality:

```bash
# Run Apple MDM tests
go test -v ./server/mdm/...
```

## Debugging

### Server-Side Debugging

1. Enable debug logging in the Fleet server
2. Monitor the MDM API endpoints
3. Inspect the MDM command queue

### Device-Side Debugging

On macOS, you can view MDM logs using:

```bash
# View MDM logs
log stream --predicate 'subsystem contains "com.apple.ManagedClient"'
```

On iOS, you can view MDM logs using:

```bash
# Connect device to Mac with Xcode
# Use Console app to view logs with filter: subsystem:com.apple.ManagedClient
```

## Common Issues and Solutions

### Enrollment Issues

- **Certificate Issues**: Ensure all certificates are valid and properly configured
- **Profile Signing**: Verify the enrollment profile is properly signed
- **Server URL**: Check the MDM server URL is accessible from the device

### Command Issues

- **Command Format**: Verify the command XML is properly formatted
- **Command Queue**: Check if commands are stuck in the queue
- **Push Notifications**: Ensure push notifications are being sent and received

## Related Resources

- [Apple MDM Documentation](https://developer.apple.com/documentation/devicemanagement)
- [MDM Protocol Reference](https://developer.apple.com/business/documentation/MDM-Protocol-Reference.pdf)
- [Configuration Profile Reference](https://developer.apple.com/business/documentation/Configuration-Profile-Reference.pdf)
- [Apple MDM Architecture](../../architecture/mdm/apple-mdm-architecture.md)