package packaging

import "text/template"

// Partially adapted from Launcher's wix XML in
// https://github.com/kolide/launcher/blob/master/pkg/packagekit/internal/assets/main.wxs.
var windowsWixTemplate = template.Must(template.New("").Option("missingkey=error").Parse(
	`<?xml version="1.0" encoding="UTF-8"?>
<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi" xmlns:util="http://schemas.microsoft.com/wix/UtilExtension">
  <Product
    Id="C2C2437D-0562-465E-A0BB-2C4484025BD6"
    Name="Fleet osquery"
    Language="1033"
    Version="{{.Version}}"
    Manufacturer="Fleet Device Management (fleetdm.com)"
    UpgradeCode="B681CB20-107E-428A-9B14-2D3C1AFED244" >

    <Package
      Id="*"
      Keywords='Fleet osquery'
      Description="Fleet osquery"
      InstallerVersion="500"
      Compressed="yes"
      InstallScope="perMachine"
      InstallPrivileges="elevated"
      Languages="1033" />

    <MediaTemplate EmbedCab="yes" />

    <MajorUpgrade AllowDowngrades="yes" />

    <Directory Id="TARGETDIR" Name="SourceDir">
      <Directory Id="ProgramFiles64Folder">
        <Directory Id="ORBITROOT" Name="Orbit">
          <Component Id="C_ORBITROOT" Guid="A7DFD09E-2D2B-4535-A04F-5D4DE90F3863">
            <CreateFolder>
              <PermissionEx Sddl="O:SYG:SYD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)" />
            </CreateFolder>
          </Component>
          <Directory Id="ORBITBIN" Name="bin">
            <Directory Id="ORBITBINORBIT" Name="orbit">
              <Component Id="C_ORBITBIN" Guid="AF347B4E-B84B-4DD4-9C4D-133BE17B613D">
                <CreateFolder>
                  <PermissionEx Sddl="O:SYG:SYD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)" />
                </CreateFolder>
                <File Source="root\bin\orbit\windows\{{ .OrbitChannel }}\orbit.exe">
                  <PermissionEx Sddl="O:SYG:SYD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)" />
                </File>
                <ServiceInstall
                  Name="Fleet osquery"
                  Account="NT AUTHORITY\SYSTEM"
                  ErrorControl="ignore"
                  Start="auto"
                  Type="ownProcess"
                  Arguments='--root-dir "[ORBITROOT]." --log-file "[ORBITROOT]orbit-log.txt" {{ if .FleetURL }}--fleet-url "{{ .FleetURL }}"{{ end }} {{ if .FleetCertificate }}--fleet-certificate "[ORBITROOT]fleet.pem"{{ end }} {{ if .EnrollSecret }}--enroll-secret-path "[ORBITROOT]secret.txt"{{ end }} {{if .Insecure }}--insecure{{ end }} {{ if .UpdateURL }}--update-url "{{ .UpdateURL }}" {{ end }} --orbit-channel "{{ .OrbitChannel }}" --osqueryd-channel "{{ .OsquerydChannel }}"'
                >
                  <util:ServiceConfig
                    FirstFailureActionType="restart"
                    SecondFailureActionType="restart"
                    ThirdFailureActionType="restart"
                    ResetPeriodInDays="1"
                    RestartServiceDelayInSeconds="1"
                  />
                </ServiceInstall>
                <ServiceControl
                  Id="StartOrbitService"
                  Name="Fleet osquery"
                  Start="install"
                  Stop="both"
                  Remove="uninstall"
                />
              </Component>
            </Directory>
          </Directory>
        </Directory>
      </Directory>
    </Directory>

    <Feature Id="Orbit" Title="Fleet osquery" Level="1" Display="hidden">
      <ComponentGroupRef Id="OrbitFiles" />
      <ComponentRef Id="C_ORBITBIN" />
      <ComponentRef Id="C_ORBITROOT" />
    </Feature>

  </Product>
</Wix>
`))
