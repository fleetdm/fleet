# Windows MDM Development Guide

This guide provides instructions for developing Windows Mobile Device Management (MDM) functionality in Fleet.

## Prerequisites

Before you begin developing Windows MDM functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of the [Windows MDM Protocol](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mdm/33769a92-ac31-47ef-ae7b-dc8501f7104f)
- Access to Windows devices for testing
- Understanding of SyncML and OMA-DM

## Setting Up the Development Environment

### Certificates

Windows MDM requires certificates for secure communication:

1. **SSL Certificate**: Used for securing the MDM server endpoint
   - For development, you can use a self-signed certificate
   - For production, you need a trusted SSL certificate

2. **Signing Certificate**: Used for signing enrollment packages
   - For development, you can use a self-signed certificate
   - For production, you need a code signing certificate

### Setting Up Certificates

```bash
# Generate a self-signed SSL certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout server.key -out server.crt

# Generate a self-signed code signing certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout signing.key -out signing.crt
```

### Configuration

Configure Fleet for Windows MDM development:

1. Edit your Fleet configuration file to include the MDM certificates
2. Set the MDM server URL to your development server
3. Configure the enrollment settings

## Development Workflow

### 1. Enrollment Package Development

The enrollment package is used to enroll Windows devices:

1. Create an enrollment package with the required settings
2. Sign the package with the signing certificate
3. Test the package on a device

Example enrollment package structure:

```xml
<wap-provisioningdoc>
  <characteristic type="EnterpriseExternalMDMDiscovery">
    <characteristic type="DiscoveryServiceFullURL">
      <parm name="URL" value="https://mdm.example.com/EnrollmentServer/Discovery.svc" />
    </characteristic>
  </characteristic>
</wap-provisioningdoc>
```

### 2. SyncML Message Development

SyncML messages are used for MDM communication:

1. Implement SyncML message handling in the Fleet server
2. Test the messages with an enrolled device
3. Handle the message responses

Example SyncML message structure:

```xml
<SyncML xmlns="SYNCML:SYNCML1.2">
  <SyncHdr>
    <VerDTD>1.2</VerDTD>
    <VerProto>DM/1.2</VerProto>
    <SessionID>1</SessionID>
    <MsgID>1</MsgID>
    <Target>
      <LocURI>urn:uuid:device-id</LocURI>
    </Target>
    <Source>
      <LocURI>https://mdm.example.com</LocURI>
    </Source>
  </SyncHdr>
  <SyncBody>
    <Get>
      <CmdID>1</CmdID>
      <Item>
        <Target>
          <LocURI>./DevDetail/DevInfo/DevId</LocURI>
        </Target>
      </Item>
    </Get>
    <Final/>
  </SyncBody>
</SyncML>
```

### 3. Configuration Service Provider (CSP) Development

CSPs are used to configure Windows devices:

1. Identify the appropriate CSP for the configuration
2. Implement the SyncML commands to use the CSP
3. Test the configuration on an enrolled device

Example CSP usage in SyncML:

```xml
<SyncML xmlns="SYNCML:SYNCML1.2">
  <SyncHdr>
    <!-- Header content -->
  </SyncHdr>
  <SyncBody>
    <Replace>
      <CmdID>1</CmdID>
      <Item>
        <Target>
          <LocURI>./Vendor/MSFT/Policy/Config/WiFi/AllowWiFi</LocURI>
        </Target>
        <Meta>
          <Format xmlns="syncml:metinf">int</Format>
        </Meta>
        <Data>1</Data>
      </Item>
    </Replace>
    <Final/>
  </SyncBody>
</SyncML>
```

## Testing

### Manual Testing

1. Enroll a test device using the enrollment package
2. Execute MDM commands from the Fleet server
3. Verify the device responds correctly
4. Check the MDM logs on the device

### Automated Testing

Fleet includes automated tests for Windows MDM functionality:

```bash
# Run Windows MDM tests
go test -v ./server/mdm/windows/...
```

## Debugging

### Server-Side Debugging

1. Enable debug logging in the Fleet server
2. Monitor the MDM API endpoints
3. Inspect the SyncML messages

### Device-Side Debugging

On Windows, you can view MDM logs using:

```powershell
# View MDM logs in Event Viewer
Get-WinEvent -LogName "Microsoft-Windows-DeviceManagement-Enterprise-Diagnostics-Provider/Admin"

# Export MDM logs to a file
Get-WinEvent -LogName "Microsoft-Windows-DeviceManagement-Enterprise-Diagnostics-Provider/Admin" | Export-Csv -Path mdm-logs.csv
```

## Common Issues and Solutions

### Enrollment Issues

- **Certificate Issues**: Ensure all certificates are valid and properly configured
- **Package Signing**: Verify the enrollment package is properly signed
- **Server URL**: Check the MDM server URL is accessible from the device

### SyncML Issues

- **Message Format**: Verify the SyncML messages are properly formatted
- **Command Sequence**: Check if commands are sent in the correct sequence
- **Status Codes**: Understand the status codes returned by the device

### CSP Issues

- **CSP Support**: Verify the CSP is supported on the target Windows version
- **Permission Issues**: Check if the MDM server has permission to use the CSP
- **Value Format**: Ensure the values sent to the CSP are in the correct format

## Working with Configuration Service Providers (CSPs)

Windows MDM uses CSPs to configure various aspects of the device. Here are some commonly used CSPs:

### Policy CSP

Used for configuring device policies:

```xml
<Replace>
  <CmdID>1</CmdID>
  <Item>
    <Target>
      <LocURI>./Vendor/MSFT/Policy/Config/Security/RequireDeviceEncryption</LocURI>
    </Target>
    <Meta>
      <Format xmlns="syncml:metinf">int</Format>
    </Meta>
    <Data>1</Data>
  </Item>
</Replace>
```

### EnterpriseModernAppManagement CSP

Used for managing modern applications:

```xml
<Add>
  <CmdID>1</CmdID>
  <Item>
    <Target>
      <LocURI>./Vendor/MSFT/EnterpriseModernAppManagement/AppInstallation</LocURI>
    </Target>
    <Meta>
      <Format xmlns="syncml:metinf">chr</Format>
    </Meta>
    <Data><!-- App installation data --></Data>
  </Item>
</Add>
```

### DeviceStatus CSP

Used for retrieving device status information:

```xml
<Get>
  <CmdID>1</CmdID>
  <Item>
    <Target>
      <LocURI>./Vendor/MSFT/DeviceStatus/DeviceGuard/VirtualizationBasedSecurityStatus</LocURI>
    </Target>
  </Item>
</Get>
```

## Related Resources

- [Windows MDM Protocol Reference](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mdm/33769a92-ac31-47ef-ae7b-dc8501f7104f)
- [Configuration Service Provider Reference](https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-reference)
- [SyncML Reference](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mdm/9aea82c7-d898-4681-bff5-ce6f7e1107cc)
- [Windows MDM Architecture](../../architecture/mdm/windows-mdm-architecture.md)
- [Windows MDM Glossary and Protocol](../../product-groups/mdm/windows-mdm-glossary-and-protocol.md)