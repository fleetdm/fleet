package packaging

import "text/template"

// Partially adapted from Launcher's wix XML in
// https://github.com/kolide/launcher/blob/master/pkg/packagekit/internal/assets/main.wxs.
var windowsWixTemplate = template.Must(template.New("").Option("missingkey=error").Parse(
	`<?xml version="1.0" encoding="UTF-8"?>
<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi" xmlns:util="http://schemas.microsoft.com/wix/UtilExtension">
  <Product
    Id="*"
    Name="Fleet osquery"
    Language="1033"
    Version="{{.Version}}"
    Manufacturer="Fleet Device Management (fleetdm.com)"
    UpgradeCode="B681CB20-107E-428A-9B14-2D3C1AFED244" >

    <Package
      Keywords='Fleet osquery'
      Description="Fleet osquery"
      InstallerVersion="500"
      Compressed="yes"
      InstallScope="perMachine"
      InstallPrivileges="elevated"
      Languages="1033" />

    <Property Id="REINSTALLMODE" Value="amus" />

    <Property Id="APPLICATIONFOLDER">
      <RegistrySearch Key="SOFTWARE\FleetDM\Orbit" Root="HKLM" Type="raw" Id="APPLICATIONFOLDER_REGSEARCH" Name="Path" />
    </Property>

    <Property Id="ARPNOREPAIR" Value="yes" Secure="yes" />
    <Property Id="ARPNOMODIFY" Value="yes" Secure="yes" />

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
          <Component Id="C_ORBITROOT_REMOVAL" Guid="B7DFD19E-3D2B-4536-A04F-5D4DE90F3863">
            <RegistryValue Root="HKLM" Key="SOFTWARE\FleetDM\Orbit" Name="Path" Type="string" Value="[ORBITROOT]" KeyPath="yes" />
            <util:RemoveFolderEx On="uninstall" Property="APPLICATIONFOLDER" />
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
                <Environment Id='OrbitUpdateInterval' Name='ORBIT_UPDATE_INTERVAL' Value='{{ .OrbitUpdateInterval }}' Action='set' System='yes' />
                <!--
                  ##############################################################################################
                  NOTE: We've seen some system fail to install the MSI when using Account="NT AUTHORITY\SYSTEM"
                  ##############################################################################################
                  -->
                <ServiceInstall
                  Name="Fleet osquery"
                  Account="LocalSystem"
                  ErrorControl="ignore"
                  Start="auto"
                  Type="ownProcess"
                  Description="This service runs Fleet's osquery runtime and autoupdater (Orbit)."
                  Arguments='--root-dir "[ORBITROOT]." --log-file "[System64Folder]config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log"{{ if .FleetURL }} --fleet-url "{{ .FleetURL }}"{{ end }}{{ if .FleetCertificate }} --fleet-certificate "[ORBITROOT]fleet.pem"{{ end }}{{ if .EnrollSecret }} --enroll-secret-path "[ORBITROOT]secret.txt"{{ end }}{{if .Insecure }} --insecure{{ end }}{{ if .Debug }} --debug{{ end }}{{ if .UpdateURL }} --update-url "{{ .UpdateURL }}"{{ end }}{{ if .DisableUpdates }} --disable-updates{{ end }}{{ if .Desktop }} --fleet-desktop --desktop-channel {{ .DesktopChannel }}{{ end }} --orbit-channel "{{ .OrbitChannel }}" --osqueryd-channel "{{ .OsquerydChannel }}"'
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
      <ComponentRef Id="C_ORBITROOT_REMOVAL" />
      <ComponentRef Id="C_ORBITBIN" />
      <ComponentRef Id="C_ORBITROOT" />
    </Feature>

  </Product>
</Wix>
`))

var windowsOsqueryEventLogTemplate = template.Must(template.New("").Option("missingkey=error").Parse(
	`<?xml version="1.0"?>
<instrumentationManifest xsi:schemaLocation="http://schemas.microsoft.com/win/2004/08/events eventman.xsd" xmlns="http://schemas.microsoft.com/win/2004/08/events" xmlns:win="http://manifests.microsoft.com/win/2004/08/windows/events" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:trace="http://schemas.microsoft.com/win/2004/08/events/trace">
	<instrumentation>
		<events>
			<provider name="FleetDM" guid="{F7740E18-3259-434F-9759-976319968900}" symbol="OsqueryWindowsEventLogProvider" resourceFileName="%systemdrive%\Program Files\Orbit\bin\osqueryd\windows\{{ .OsquerydChannel }}\osqueryd.exe" messageFileName="%systemdrive%\Program Files\Orbit\bin\osqueryd\windows\{{ .OsquerydChannel }}\osqueryd.exe">
				<events>
					<event symbol="DebugMessage" value="1" version="0" channel="osquery" level="win:Warning" task="LogMessage" opcode="MessageOpcode" template="_template_message" keywords="DebugWindowsEventLogMessage " message="$(string.osquery.event.1.message)"></event>
					<event symbol="InfoMessage" value="2" version="0" channel="osquery" level="win:Informational" task="LogMessage" opcode="MessageOpcode" template="_template_message" keywords="InfoWindowsEventLogMessage " message="$(string.osquery.event.2.message)"></event>
					<event symbol="WarningMessage" value="3" version="0" channel="osquery" level="win:Warning" task="LogMessage" opcode="MessageOpcode" template="_template_message" keywords="WarningWindowsEventLogMessage " message="$(string.osquery.event.3.message)"></event>
					<event symbol="ErrorMessage" value="4" version="0" channel="osquery" level="win:Error" task="LogMessage" opcode="MessageOpcode" template="_template_message" keywords="ErrorWindowsEventLogMessage " message="$(string.osquery.event.4.message)"></event>
					<event symbol="FatalMessage" value="5" version="0" channel="osquery" level="win:Critical" task="LogMessage" opcode="MessageOpcode" template="_template_message" keywords="FatalWindowsEventLogMessage " message="$(string.osquery.event.5.message)"></event>
				</events>
				<levels></levels>
				<tasks>
					<task name="LogMessage" symbol="WindowsEventLogMessage" value="1" eventGUID="{D3C2B9E0-4AFE-41BD-99BE-F00EE4DFEB17}"></task>
				</tasks>
				<opcodes>
					<opcode name="MessageOpcode" symbol="_opcode_message" value="10"></opcode>
				</opcodes>
				<channels>
					<channel name="osquery" chid="osquery" symbol="OsqueryWindowsEventLogChannel" type="Admin" enabled="true" message="$(string.osquery.channel.PrimaryWindowsEventLogChannel.message)"></channel>
				</channels>
				<keywords>
					<keyword name="InfoWindowsEventLogMessage" symbol="_keyword_info_message" mask="0x1"></keyword>
					<keyword name="WarningWindowsEventLogMessage" symbol="_keyword_warning_message" mask="0x2"></keyword>
					<keyword name="ErrorWindowsEventLogMessage" symbol="_keyword_error_message" mask="0x4"></keyword>
					<keyword name="FatalWindowsEventLogMessage" symbol="_keyword_fatal_message" mask="0x8"></keyword>
					<keyword name="DebugWindowsEventLogMessage" symbol="_keyword_debug_message" mask="0x10"></keyword>
				</keywords>
				<templates>
					<template tid="_template_message">
						<data name="Message" inType="win:AnsiString" outType="xs:string"></data>
						<data name="Location" inType="win:AnsiString" outType="xs:string"></data>
					</template>
				</templates>
			</provider>
		</events>
	</instrumentation>
	<localization>
		<resources culture="en-US">
			<stringTable>
				<string id="osquery.event.5.message" value="Fatal error"></string>
				<string id="osquery.event.4.message" value="Error"></string>
				<string id="osquery.event.3.message" value="Warning"></string>
				<string id="osquery.event.2.message" value="Information"></string>
				<string id="osquery.event.1.message" value="Debug"></string>
				<string id="osquery.channel.PrimaryWindowsEventLogChannel.message" value="osquery"></string>
				<string id="level.Warning" value="Warning"></string>
				<string id="level.Verbose" value="Verbose"></string>
				<string id="level.Informational" value="Information"></string>
				<string id="level.Error" value="Error"></string>
				<string id="level.Critical" value="Critical"></string>
			</stringTable>
		</resources>
	</localization>
</instrumentationManifest>
`))
