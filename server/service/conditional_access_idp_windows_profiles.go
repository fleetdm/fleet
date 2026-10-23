package service

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // CSP CAThumbprint is defined as the SHA-1 hash
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"slices"
	"strings"
	"text/template"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
)

// Fleet requires a SCEP profile to contain only SCEP nodes and its Enroll Exec, so the browser configuration is a
// separate profile. The SCEP URL and challenge are Fleet variables filled in at delivery, which makes Fleet track the
// certificate like a proxied certificate authority's.
const conditionalAccessWindowsSCEPProfileTemplate = `<!-- Fleet conditional access for Okta: SCEP certificate (user store) -->
<Add>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">node</Format></Meta>
  </Item>
</Add>
<Add>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/ServerURL</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
    <Data>$FLEET_VAR_CONDITIONAL_ACCESS_SCEP_URL</Data>
  </Item>
</Add>
<Add>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Challenge</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
    <Data>$FLEET_VAR_CONDITIONAL_ACCESS_SCEP_CHALLENGE</Data>
  </Item>
</Add>
<Add>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/CAThumbprint</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
    <Data>{{ xml .CAThumbprintSHA1 }}</Data>
  </Item>
</Add>
<Add>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/SubjectName</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
    <Data>CN={{ xml .CertificateCN }},OU=$FLEET_VAR_CERTIFICATE_RENEWAL_ID</Data>
  </Item>
</Add>
<Add>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/SubjectAlternativeNames</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
    <Data>7+urn:device:fleet:uuid:$FLEET_VAR_HOST_UUID</Data>
  </Item>
</Add>
<Add>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/EKUMapping</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
    <Data>1.3.6.1.5.5.7.3.2</Data>
  </Item>
</Add>
<Add>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/KeyUsage</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">int</Format></Meta>
    <Data>160</Data>
  </Item>
</Add>
<Add>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/KeyLength</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">int</Format></Meta>
    <Data>2048</Data>
  </Item>
</Add>
<Add>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/HashAlgorithm</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
    <Data>SHA-2</Data>
  </Item>
</Add>
<Add>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/KeyProtection</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">int</Format></Meta>
    <Data>2</Data>
  </Item>
</Add>
<Add>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/RetryCount</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">int</Format></Meta>
    <Data>3</Data>
  </Item>
</Add>
<Add>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/RetryDelay</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">int</Format></Meta>
    <Data>10</Data>
  </Item>
</Add>
<Exec>
  <Item>
    <Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll</LocURI></Target>
  </Item>
</Exec>
`

const (
	// conditionalAccessWindowsADMXInstallURI ingests conditionalAccessWindowsADMX. The path is
	// ADMXInstall/{AppName}/{SettingType}/{FileUID}.
	conditionalAccessWindowsADMXInstallURI = "./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/FleetConditionalAccess/Policy/FleetConditionalAccessAdmx"
	// The policy area is {AppName}~{SettingType}~{CategoryFromTheADMX}.
	conditionalAccessWindowsChromePolicyURI = "./Device/Vendor/MSFT/Policy/Config/FleetConditionalAccess~Policy~fleetca/ChromeAutoSelectCertificate"
	conditionalAccessWindowsEdgePolicyURI   = "./Device/Vendor/MSFT/Policy/Config/FleetConditionalAccess~Policy~fleetca/EdgeAutoSelectCertificate"
)

// .ADMX, .ChromePolicy and .EdgePolicy are escaped in Go, so they don't go through xml. The ADMX must be ingested
// before the policies it declares are set.
const conditionalAccessWindowsBrowserProfileTemplate = `<!-- Fleet conditional access for Okta: Chrome and Edge auto-select the certificate for the mTLS endpoint -->
<Add>
  <Item>
    <Target><LocURI>` + conditionalAccessWindowsADMXInstallURI + `</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format><Type xmlns="syncml:metinf">text/plain</Type></Meta>
    <Data>{{ .ADMX }}</Data>
  </Item>
</Add>
<Replace>
  <Item>
    <Target><LocURI>` + conditionalAccessWindowsChromePolicyURI + `</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format><Type xmlns="syncml:metinf">text/plain</Type></Meta>
    <Data>{{ .ChromePolicy }}</Data>
  </Item>
</Replace>
<Replace>
  <Item>
    <Target><LocURI>` + conditionalAccessWindowsEdgePolicyURI + `</LocURI></Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format><Type xmlns="syncml:metinf">text/plain</Type></Meta>
    <Data>{{ .EdgePolicy }}</Data>
  </Item>
</Replace>
`

// conditionalAccessWindowsADMX declares the Chrome and Edge AutoSelectCertificateForUrls list policies, since Windows MDM
// can't write a registry value directly. Chromium reads every integer-named value under those keys, so value name 999
// leaves an admin's own 1, 2, ... entries alone.
//
//nolint:gosec // G101 false positive, a policy definition, not a credential
const conditionalAccessWindowsADMX = `<policyDefinitions revision="1.0" schemaVersion="1.0">
  <policyNamespaces>
    <target namespace="FleetDM.Policies.ConditionalAccess" prefix="fleetca"/>
    <using namespace="Microsoft.Policies.Windows" prefix="windows"/>
  </policyNamespaces>
  <resources minRequiredRevision="1.0"/>
  <categories>
    <category name="fleetca" displayName="Fleet conditional access"/>
  </categories>
  <policies>
    <policy name="ChromeAutoSelectCertificate" class="Machine" displayName="Chrome: auto-select the Fleet conditional access certificate"
            explainText="Makes Google Chrome present the Fleet conditional access certificate to the Okta mTLS endpoint without prompting."
            key="Software\Policies\Google\Chrome\AutoSelectCertificateForUrls">
      <parentCategory ref="fleetca"/>
      <supportedOn ref="windows:SUPPORTED_Windows7"/>
      <elements>
        <text id="ChromeAutoSelectCertificate_Value" valueName="999" required="true"/>
      </elements>
    </policy>
    <policy name="EdgeAutoSelectCertificate" class="Machine" displayName="Edge: auto-select the Fleet conditional access certificate"
            explainText="Makes Microsoft Edge present the Fleet conditional access certificate to the Okta mTLS endpoint without prompting."
            key="Software\Policies\Microsoft\Edge\AutoSelectCertificateForUrls">
      <parentCategory ref="fleetca"/>
      <supportedOn ref="windows:SUPPORTED_Windows7"/>
      <elements>
        <text id="EdgeAutoSelectCertificate_Value" valueName="999" required="true"/>
      </elements>
    </policy>
  </policies>
</policyDefinitions>`

var (
	conditionalAccessWindowsSCEPProfileTemplateParsed = template.Must(
		template.New("conditionalAccessWindowsSCEPProfile").
			Funcs(conditionalAccessProfileFuncMap).
			Parse(conditionalAccessWindowsSCEPProfileTemplate))
	conditionalAccessWindowsBrowserProfileTemplateParsed = template.Must(
		template.New("conditionalAccessWindowsBrowserProfile").
			Parse(conditionalAccessWindowsBrowserProfileTemplate))
)

// autoSelectCertificateRule is the Chromium AutoSelectCertificateForUrls rule, the same one the Apple profile writes.
type autoSelectCertificateRule struct {
	Pattern string `json:"pattern"`
	Filter  struct {
		Subject struct {
			CN string `json:"CN"`
		} `json:"SUBJECT"`
	} `json:"filter"`
}

type conditionalAccessGetIdPWindowsProfilesResponse struct {
	*fleet.ConditionalAccessWindowsProfiles
	Err error `json:"error,omitempty"`
}

func (r conditionalAccessGetIdPWindowsProfilesResponse) Error() error { return r.Err }

func conditionalAccessGetIdPWindowsProfilesEndpoint(ctx context.Context, _ any, svc fleet.Service) (fleet.Errorer, error) {
	profiles, err := svc.ConditionalAccessGetIdPWindowsProfiles(ctx)
	if err != nil {
		return conditionalAccessGetIdPWindowsProfilesResponse{Err: err}, nil
	}
	return conditionalAccessGetIdPWindowsProfilesResponse{ConditionalAccessWindowsProfiles: profiles}, nil
}

func (svc *Service) ConditionalAccessGetIdPWindowsProfiles(ctx context.Context) (*fleet.ConditionalAccessWindowsProfiles, error) {
	if err := svc.authz.Authorize(ctx, &fleet.ConditionalAccessIDPAssets{}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to authorize")
	}

	in, err := svc.loadConditionalAccessProfileInputs(ctx)
	if err != nil {
		return nil, err
	}
	// The profile doesn't carry the secret, but delivery uses the first global secret Windows can send, so the profile
	// fails on every host when none is.
	if !slices.ContainsFunc(in.enrollSecrets, func(s *fleet.EnrollSecret) bool { return microsoft_mdm.IsValidWindowsSCEPChallenge(s.Secret) }) {
		return nil, &fleet.BadRequestError{Message: microsoft_mdm.ConditionalAccessSCEPChallengeInvalidCharsMessage}
	}

	thumbprint := sha1.Sum(in.caCert.Raw) //nolint:gosec // CSP CAThumbprint is defined as the SHA-1 hash
	var scepProfile bytes.Buffer
	if err := conditionalAccessWindowsSCEPProfileTemplateParsed.Execute(&scepProfile, struct {
		CAThumbprintSHA1 string
		CertificateCN    string
	}{
		CAThumbprintSHA1: strings.ToUpper(hex.EncodeToString(thumbprint[:])),
		CertificateCN:    fleet.ConditionalAccessOktaCertificateCN,
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to execute Windows SCEP profile template")
	}

	rule := autoSelectCertificateRule{Pattern: in.mtlsURL}
	rule.Filter.Subject.CN = fleet.ConditionalAccessOktaCertificateCN
	ruleJSON, err := json.Marshal(rule)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to marshal auto-select certificate rule")
	}
	chromePolicy, err := windowsPolicyData("ChromeAutoSelectCertificate_Value", string(ruleJSON))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to render Chrome policy")
	}
	edgePolicy, err := windowsPolicyData("EdgeAutoSelectCertificate_Value", string(ruleJSON))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to render Edge policy")
	}
	var admx bytes.Buffer
	if err := xml.EscapeText(&admx, []byte(conditionalAccessWindowsADMX)); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to escape ADMX")
	}

	var browserProfile bytes.Buffer
	if err := conditionalAccessWindowsBrowserProfileTemplateParsed.Execute(&browserProfile, struct {
		ADMX         string
		ChromePolicy string
		EdgePolicy   string
	}{
		ADMX:         admx.String(),
		ChromePolicy: chromePolicy,
		EdgePolicy:   edgePolicy,
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to execute Windows browser profile template")
	}

	return &fleet.ConditionalAccessWindowsProfiles{
		SCEPProfile:    scepProfile.String(),
		BrowserProfile: browserProfile.String(),
	}, nil
}
