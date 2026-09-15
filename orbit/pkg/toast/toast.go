// Package toast posts Windows toast notifications for Fleet Desktop.
package toast

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"strconv"
	"time"
	"unicode/utf16"
)

const (
	// FleetDesktopAppID is the AppUserModelID orbit registers so toasts are labelled "Fleet Desktop" with its icon.
	FleetDesktopAppID = "FleetDM.FleetDesktop"
	// powerShellAppID is the built-in Windows PowerShell AppUserModelID, used when Fleet Desktop's is not registered.
	powerShellAppID = `{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe`
)

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
}

type toastXML struct {
	XMLName        xml.Name `xml:"toast"`
	Launch         string   `xml:"launch,attr"`
	ActivationType string   `xml:"activationType,attr"`
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

// xml renders the toast payload. Protocol activation hands the URL to its default handler, so clicking the toast or the
// button opens the browser without any Fleet Desktop code running.
func (n Notification) xml() (string, error) {
	payload := toastXML{Launch: n.URL, ActivationType: "protocol"}
	payload.Binding.Template = "ToastGeneric"
	payload.Binding.Text = []string{n.Title, n.Body}
	payload.Actions = []toastAction{{Content: n.ButtonLabel, ActivationType: "protocol", Arguments: n.URL}}
	b, err := xml.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("rendering toast XML: %w", err)
	}
	return string(b), nil
}

// The scripts are constant and read everything else from the environment, so no value needs PowerShell quoting and the
// token-bearing URL stays out of the command line, script block logging, and the script lines PowerShell quotes in errors.
const (
	loadToastTypes = `$ErrorActionPreference = 'Stop'
$null = [Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime]
$null = [Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime]
`
	showScript = loadToastTypes + `$xml = [Windows.Data.Xml.Dom.XmlDocument]::new()
$xml.LoadXml($env:FLEET_TOAST_XML)
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
$toast.Tag = $env:FLEET_TOAST_TAG
$toast.Group = $env:FLEET_TOAST_GROUP
$toast.ExpirationTime = [DateTimeOffset]::Now.AddSeconds([int]$env:FLEET_TOAST_EXPIRES_SECONDS)
$toast.SuppressPopup = $env:FLEET_TOAST_SUPPRESS_POPUP -eq 'true'
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($env:FLEET_TOAST_APP_ID).Show($toast)
`
	// removeScript clears the toast under both AppUserModelIDs, because orbit may have registered Fleet Desktop's
	// between posting the toast and removing it.
	removeScript = loadToastTypes + `$history = [Windows.UI.Notifications.ToastNotificationManager]::History
$history.Remove($env:FLEET_TOAST_TAG, $env:FLEET_TOAST_GROUP, $env:FLEET_TOAST_APP_ID)
$history.Remove($env:FLEET_TOAST_TAG, $env:FLEET_TOAST_GROUP, $env:FLEET_TOAST_FALLBACK_APP_ID)
`
)

func showEnv(n Notification, appID string) ([]string, error) {
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
		"FLEET_TOAST_APP_ID=" + appID,
	}, nil
}

func removeEnv(tag, group string) []string {
	return []string{
		"FLEET_TOAST_TAG=" + tag,
		"FLEET_TOAST_GROUP=" + group,
		"FLEET_TOAST_APP_ID=" + FleetDesktopAppID,
		"FLEET_TOAST_FALLBACK_APP_ID=" + powerShellAppID,
	}
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
