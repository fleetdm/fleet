package main

import (
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/open"
	"github.com/getlantern/systray"
)

var version = "unknown"

type Response struct {
	Host struct {
		Policies []struct {
			ID          int         `json:"id"`
			Name        string      `json:"name"`
			Query       string      `json:"query"`
			Description string      `json:"description"`
			AuthorID    int         `json:"author_id"`
			AuthorName  string      `json:"author_name"`
			AuthorEmail string      `json:"author_email"`
			TeamID      interface{} `json:"team_id"`
			Resolution  string      `json:"resolution"`
			Platform    string      `json:"platform"`
			CreatedAt   time.Time   `json:"created_at"`
			UpdatedAt   time.Time   `json:"updated_at"`
			Response    string      `json:"response"`
		} `json:"policies"`
		Status      string `json:"status"`
		DisplayText string `json:"display_text"`
	} `json:"host"`
	License struct {
		Tier         string    `json:"tier"`
		Organization string    `json:"organization"`
		DeviceCount  int       `json:"device_count"`
		Expiration   time.Time `json:"expiration"`
		Note         string    `json:"note"`
	} `json:"license"`
}

func main() {
	// Our TUF provided targets must support launching with "--help".
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Fleet Desktop application executable")
		return
	}
	log.Printf("fleet-desktop version=%s\n", version)

	devURL := os.Getenv("FLEET_DESKTOP_DEVICE_URL")
	if devURL == "" {
		log.Println("missing URL environment FLEET_DESKTOP_DEVICE_URL")
		os.Exit(1)
	}
	deviceURL, err := url.Parse(devURL)
	if err != nil {
		log.Printf("invalid URL argument: %s\n", err)
		os.Exit(1)
	}
	devTestPath := os.Getenv("FLEET_DESKTOP_DEVICE_API_TEST_PATH")
	if devTestPath == "" {
		log.Println("missing URL environment FLEET_DESKTOP_DEVICE_API_TEST_PATH")
		os.Exit(1)
	}
	devTestURL := *deviceURL
	devTestURL.Path = devTestPath

	onReady := func() {
		log.Println("ready")

		systray.SetTemplateIcon(icoBytes, icoBytes)
		systray.SetTooltip("Fleet Device Management Menu.")
		myDeviceItem := systray.AddMenuItem("Initializing...", "")
		myDeviceItem.Disable()
		transparencyItem := systray.AddMenuItem("Transparency", "")

		// Perform API test call to enable the "My device" item as soon
		// as the device auth token is registered by Fleet.
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			tr := http.DefaultTransport.(*http.Transport)
			if os.Getenv("FLEET_DESKTOP_INSECURE") != "" {
				tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402
			}
			client := &http.Client{
				Transport: tr,
			}

			for {
				resp, err := client.Get(devTestURL.String())
				if err != nil {
					// To ease troubleshooting we set the tooltip as the error.
					myDeviceItem.SetTooltip(err.Error())
					log.Printf("get device URL: %s", err)
				} else {
					resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						myDeviceItem.SetTitle("My device")
						myDeviceItem.Enable()
						myDeviceItem.SetTooltip("")
						return
					}
				}
				<-ticker.C
			}
		}()

		go func() {
			tic := time.NewTicker(5 * time.Minute)
			defer tic.Stop()

			tr := http.DefaultTransport.(*http.Transport)
			if os.Getenv("FLEET_DESKTOP_INSECURE") != "" {
				tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			}
			client := &http.Client{
				Transport: tr,
			}

			for {
				<-tic.C
				// TODO: Use the policies endpoint instead of full device endpoint
				// https://github.com/fleetdm/fleet/issues/5697
				resp, err := client.Get(devTestURL.String())
				if err != nil {
					// To ease troubleshooting we set the tooltip as the error.
					myDeviceItem.SetTooltip(err.Error())
					log.Printf("get device URL: %s", err)
					continue
				}
				licensePass, allPoliciesPass, err := parseLicenseAndPoliciesFromResponse(resp)
				if err != nil {
					log.Println(err.Error())
					continue
				}
				if licensePass {
					if allPoliciesPass {
						myDeviceItem.SetTitle("🟢 My device")
					} else {
						myDeviceItem.SetTitle("🔴 My device")
					}
				} else {
					myDeviceItem.SetTitle("My device")
				}
			}
		}()

		go func() {
			for {
				select {
				case <-myDeviceItem.ClickedCh:
					if err := open.Browser(deviceURL.String()); err != nil {
						log.Printf("open browser my device: %s", err)
					}
				case <-transparencyItem.ClickedCh:
					if err := open.Browser("https://fleetdm.com/transparency"); err != nil {
						log.Printf("open browser transparency: %s", err)
					}
				}
			}
		}()
	}
	onExit := func() {
		log.Println("exit")
	}

	systray.Run(onReady, onExit)
}

func parseLicenseAndPoliciesFromResponse(resp *http.Response) (licensePass bool, policiesPass bool, err error) {
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, false, nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, false, err
	}
	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return false, false, err
	}
	licensePass = false
	if strings.Contains(strings.ToLower(result.License.Tier), "premium") {
		licensePass = true
	}
	if licensePass {
		for _, policy := range result.Host.Policies {
			if policy.Response != "pass" {
				return licensePass, false, nil
			}
		}
		return licensePass, true, nil
	} else {
		return false, false, nil
	}
}
