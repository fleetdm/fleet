// Package toast posts Windows toast notifications for Fleet Desktop.
package toast

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// FleetDesktopAppID is the AppUserModelID orbit registers so toasts are labeled "Fleet Desktop" with its icon.
const FleetDesktopAppID = "FleetDM.FleetDesktop"

// maxTagLength is what Windows accepts for a toast's tag and group.
const maxTagLength = 64

// fleetDesktopEnvPrefix marks Fleet Desktop's own environment variables, which carry the Fleet client TLS key and the device URL.
const fleetDesktopEnvPrefix = "FLEET_DESKTOP_"

// ErrAppIDNotRegistered means orbit has not registered Fleet Desktop's AppUserModelID. Windows drops a toast posted under one it does not know.
var ErrAppIDNotRegistered = errors.New("the Fleet Desktop notification identity is not registered")

// Notification is a toast with a heading, a body, and one button that opens a URL.
type Notification struct {
	// Tag and Group identify the toast, so posting it again replaces the earlier one.
	Tag   string
	Group string
	Title string
	Body  string
	// ButtonLabel is the text of the button that opens URL.
	ButtonLabel string
	URL         string
	// ExpiresIn is how long the toast stays in Notification Center.
	ExpiresIn time.Duration
	// SuppressPopup places the toast in Notification Center without showing it on screen.
	SuppressPopup bool
	// StayOnScreen keeps the popup up until the end user acts on it or dismisses it, instead of timing out after the few
	// seconds their Windows settings allow.
	StayOnScreen bool
}

type toastXML struct {
	XMLName        xml.Name `xml:"toast"`
	Launch         string   `xml:"launch,attr"`
	ActivationType string   `xml:"activationType,attr"`
	Scenario       string   `xml:"scenario,attr,omitempty"`
	Binding        struct {
		Template string   `xml:"template,attr"`
		Text     []string `xml:"text"`
	} `xml:"visual>binding"`
	Actions []toastAction `xml:"actions>action"`
}

type toastAction struct {
	Content        string `xml:"content,attr"`
	ActivationType string `xml:"activationType,attr"`
	Arguments      string `xml:"arguments,attr"`
}

// validate reports whether Windows will accept the notification, so a caller gets a clear error instead of a PowerShell one.
func (n Notification) validate() error {
	for _, field := range []struct{ name, value string }{
		{name: "title", value: n.Title},
		{name: "body", value: n.Body},
		{name: "button label", value: n.ButtonLabel},
		{name: "URL", value: n.URL},
	} {
		if field.value == "" {
			return fmt.Errorf("toast %s is empty", field.name)
		}
	}
	return validateTagAndGroup(n.Tag, n.Group)
}

func validateTagAndGroup(tag, group string) error {
	for _, field := range []struct{ name, value string }{{name: "tag", value: tag}, {name: "group", value: group}} {
		switch {
		case field.value == "":
			return fmt.Errorf("toast %s is empty", field.name)
		case len(field.value) > maxTagLength:
			return fmt.Errorf("toast %s is longer than the %d characters Windows accepts", field.name, maxTagLength)
		}
	}
	return nil
}

// xml renders the toast payload. Protocol activation hands the URL to its default handler, so clicking the toast or the
// button opens the browser without any Fleet Desktop code running.
func (n Notification) xml() (string, error) {
	payload := toastXML{Launch: n.URL, ActivationType: "protocol"}
	if n.StayOnScreen {
		// Windows keeps a reminder on screen, but only when it has a button, which every Notification does.
		payload.Scenario = "reminder"
	}
	payload.Binding.Template = "ToastGeneric"
	payload.Binding.Text = []string{n.Title, n.Body}
	payload.Actions = []toastAction{{Content: n.ButtonLabel, ActivationType: "protocol", Arguments: n.URL}}
	b, err := xml.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("rendering toast XML: %w", err)
	}
	return string(b), nil
}

// The scripts are constant and read everything else from the environment.
const (
	// PowerShell serializes its error stream as CLIXML when it is redirected, so the body runs in a try and reports the
	// message itself. Exiting non-zero is what tells the caller it failed.
	scriptPrologue = `$ErrorActionPreference = 'Stop'
# Progress records are serialized as CLIXML on the error stream, which would bury the message below.
$ProgressPreference = 'SilentlyContinue'
try {
$null = [Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime]
$null = [Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime]
`
	scriptEpilogue = `} catch {
Write-Output $_.Exception.Message
exit 1
}
`
	showScript = scriptPrologue + `$xml = [Windows.Data.Xml.Dom.XmlDocument]::new()
$xml.LoadXml($env:FLEET_TOAST_XML)
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
$toast.Tag = $env:FLEET_TOAST_TAG
$toast.Group = $env:FLEET_TOAST_GROUP
$expiresIn = [int]$env:FLEET_TOAST_EXPIRES_SECONDS
if ($expiresIn -gt 0) { $toast.ExpirationTime = [DateTimeOffset]::Now.AddSeconds($expiresIn) }
$toast.SuppressPopup = $env:FLEET_TOAST_SUPPRESS_POPUP -eq 'true'
$notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($env:FLEET_TOAST_APP_ID)
# Show() is silent when notifications are off for this user, this app, or by policy, so report which it is instead.
# Setting reads as null on hosts where Show() works fine, so only a value that names an actual block stops us.
$setting = "$($notifier.Setting)"
if ($setting -ne '' -and $setting -ne 'Enabled') { throw "Windows notifications are $setting" }
$notifier.Show($toast)
` + scriptEpilogue
	removeScript = scriptPrologue + `[Windows.UI.Notifications.ToastNotificationManager]::History.Remove(
	$env:FLEET_TOAST_TAG, $env:FLEET_TOAST_GROUP, $env:FLEET_TOAST_APP_ID)
` + scriptEpilogue
)

func showScriptEnv(n Notification) ([]string, error) {
	if err := n.validate(); err != nil {
		return nil, err
	}
	payload, err := n.xml()
	if err != nil {
		return nil, err
	}
	return []string{
		"FLEET_TOAST_XML=" + payload,
		"FLEET_TOAST_TAG=" + n.Tag,
		"FLEET_TOAST_GROUP=" + n.Group,
		"FLEET_TOAST_EXPIRES_SECONDS=" + strconv.Itoa(int(n.ExpiresIn.Seconds())),
		"FLEET_TOAST_SUPPRESS_POPUP=" + strconv.FormatBool(n.SuppressPopup),
		"FLEET_TOAST_APP_ID=" + FleetDesktopAppID,
	}, nil
}

func removeScriptEnv(tag, group string) []string {
	return []string{
		"FLEET_TOAST_TAG=" + tag,
		"FLEET_TOAST_GROUP=" + group,
		"FLEET_TOAST_APP_ID=" + FleetDesktopAppID,
	}
}

// childEnv is the environment for the PowerShell process: this process's, without Fleet Desktop's own variables, plus the
// toast values.
func childEnv(parent, values []string) []string {
	env := make([]string, 0, len(parent)+len(values))
	for _, variable := range parent {
		// Windows environment variable names are case-insensitive.
		if strings.HasPrefix(strings.ToUpper(variable), fleetDesktopEnvPrefix) {
			continue
		}
		env = append(env, variable)
	}
	return append(env, values...)
}

// encodeCommand encodes a script for powershell.exe -EncodedCommand, which takes base64 of UTF-16LE.
func encodeCommand(script string) string {
	units := utf16.Encode([]rune(script))
	b := make([]byte, 0, 2*len(units))
	for _, u := range units {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return base64.StdEncoding.EncodeToString(b)
}
