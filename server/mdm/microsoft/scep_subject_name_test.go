package microsoft_mdm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testDeviceSubjectNameLocURI = "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/SubjectName"
	testUserSubjectNameLocURI   = "./User/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/SubjectName"
	testSANLocURI               = "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/SubjectAlternativeNames"
)

func testSyncMLItem(locURI, data string) string {
	return `<Add><Item><Target><LocURI>` + locURI + `</LocURI></Target>` +
		`<Meta><Format xmlns="syncml:metinf">chr</Format></Meta>` +
		`<Data>` + data + `</Data></Item></Add>`
}

// deliverSubjectName runs a profile through the real deployment entry point. $FLEET_VAR_HOST_UUID
// resolves straight from the params with no datastore call, so the resolved value is the caller's
// to choose, and the assertion stays on what a device receives.
func deliverSubjectName(t *testing.T, profileContents, resolvedValue string) string {
	t.Helper()
	delivered, err := PreprocessWindowsProfileContentsForDeployment(
		ProfilePreprocessDependencies{Context: t.Context()},
		ProfilePreprocessParams{HostUUID: resolvedValue, ProfileUUID: "prof-1"},
		profileContents,
	)
	require.NoError(t, err)
	return delivered
}

func TestSCEPSubjectNameQuoting(t *testing.T) {
	tests := []struct {
		name          string
		data          string
		resolvedValue string
		wantData      string
	}{
		{
			// Quoting is unconditional so it can happen before substitution, when the attribute
			// boundaries are still knowable. Windows reads a quoted plain value identically.
			name:          "value with nothing special in it is quoted too",
			data:          `CN=$FLEET_VAR_HOST_UUID,O=Fleet QA`,
			resolvedValue: "host-uuid-1234",
			wantData:      `CN="host-uuid-1234",O=Fleet QA`,
		},
		{
			name:          "variable mid value quotes the whole value",
			data:          `CN=$FLEET_VAR_HOST_UUID NDES Device Cert,O=Fleet QA`,
			resolvedValue: "user+idp@example.com",
			wantData:      `CN="user+idp@example.com NDES Device Cert",O=Fleet QA`,
		},
		{
			name:          "attribute without a variable keeps its own form",
			data:          `CN=$FLEET_VAR_HOST_UUID,OU=static+ou`,
			resolvedValue: "user+idp@example.com",
			wantData:      `CN="user+idp@example.com",OU=static+ou`,
		},
		{
			name:          "semicolon separated template keeps the trailing attribute separate",
			data:          `CN=$FLEET_VAR_HOST_UUID;O=Fleet QA`,
			resolvedValue: "user+idp@example.com",
			wantData:      `CN="user+idp@example.com";O=Fleet QA`,
		},
		{
			// Unquoted, this payload parses as two CN attributes on Windows.
			name:          "value crafted to add an attribute is contained",
			data:          `CN=$FLEET_VAR_HOST_UUID,O=Fleet QA`,
			resolvedValue: `Evil, CN=admin`,
			wantData:      `CN="Evil, CN=admin",O=Fleet QA`,
		},
		{
			// Also the only coverage of quote doubling: the value's quote is written twice, so it
			// stays content instead of ending the attribute and letting the rest become attributes.
			name:          "crafted value cannot close the quote Fleet added",
			data:          `CN=$FLEET_VAR_HOST_UUID,O=Fleet QA`,
			resolvedValue: `Evil", CN=admin`,
			wantData:      `CN="Evil&#34;&#34;, CN=admin",O=Fleet QA`,
		},
		{
			name:          "crafted value cannot close a quote the admin added",
			data:          `CN="$FLEET_VAR_HOST_UUID",O=Fleet QA`,
			resolvedValue: `Evil", CN=admin`,
			wantData:      `CN="Evil&#34;&#34;, CN=admin",O=Fleet QA`,
		},
		{
			name:          "multi valued rdn quotes only the substituted half",
			data:          `CN=$FLEET_VAR_HOST_UUID+OU=static-ou`,
			resolvedValue: "user+idp@example.com",
			wantData:      `CN="user+idp@example.com"+OU=static-ou`,
		},
		{
			name:          "braces variable form is recognized",
			data:          `CN=${FLEET_VAR_HOST_UUID}`,
			resolvedValue: "user+idp@example.com",
			wantData:      `CN="user+idp@example.com"`,
		},
		{
			name:          "template whitespace around the value is dropped",
			data:          `CN=  $FLEET_VAR_HOST_UUID  ,O=Fleet QA`,
			resolvedValue: "user+idp@example.com",
			wantData:      `CN="user+idp@example.com",O=Fleet QA`,
		},
		{
			name:          "attribute with no value is left alone",
			data:          `CN=,OU=$FLEET_VAR_HOST_UUID`,
			resolvedValue: "host-uuid-1234",
			wantData:      `CN=,OU="host-uuid-1234"`,
		},
		{
			name:          "component without an equals sign is untouched",
			data:          `$FLEET_VAR_HOST_UUID`,
			resolvedValue: "user+idp@example.com",
			wantData:      `user+idp@example.com`,
		},
		{
			name:          "cdata wrapped subject name is quoted inside the section",
			data:          `<![CDATA[CN=$FLEET_VAR_HOST_UUID,O=Fleet QA]]>`,
			resolvedValue: "user+idp@example.com",
			wantData:      `<![CDATA[CN="user+idp@example.com",O=Fleet QA]]>`,
		},
		{
			name:          "whitespace around a cdata section is preserved",
			data:          "\n  <![CDATA[CN=$FLEET_VAR_HOST_UUID]]>\n  ",
			resolvedValue: "user+idp@example.com",
			wantData:      "\n  <![CDATA[CN=\"user+idp@example.com\"]]>\n  ",
		},
		{
			name:          "cdata section without a substitution is untouched",
			data:          `<![CDATA[CN=static-cn]]>`,
			resolvedValue: "host-uuid-1234",
			wantData:      `<![CDATA[CN=static-cn]]>`,
		},
		{
			name:          "cdata opener inside the section is content, not a second section",
			data:          `<![CDATA[CN=$FLEET_VAR_HOST_UUID<![CDATA[,O=Fleet QA]]>`,
			resolvedValue: "user+idp@example.com",
			wantData:      `<![CDATA[CN="user+idp@example.com<![CDATA[",O=Fleet QA]]>`,
		},
		{
			name:          "mixed text around a cdata section is quoted as one value",
			data:          `CN=pre<![CDATA[$FLEET_VAR_HOST_UUID]]>post,O=Fleet QA`,
			resolvedValue: "Doe, Jane",
			wantData:      `CN="pre<![CDATA[Doe, Jane]]>post",O=Fleet QA`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deliverSubjectName(t, testSyncMLItem(testDeviceSubjectNameLocURI, tt.data), tt.resolvedValue)
			require.Equal(t, testSyncMLItem(testDeviceSubjectNameLocURI, tt.wantData), got)
		})
	}
}

func TestSCEPSubjectNameQuotingXMLShapes(t *testing.T) {
	const dn = `CN=$FLEET_VAR_HOST_UUID,O=Fleet QA`
	const quotedDN = `CN="user+idp@example.com",O=Fleet QA`
	const bareDN = `CN=user+idp@example.com,O=Fleet QA`
	const target = `<Target><LocURI>` + testDeviceSubjectNameLocURI + `</LocURI></Target>`

	tests := []struct {
		name    string
		profile string
		want    string
	}{
		{
			name:    "data ahead of the target in the item is quoted",
			profile: `<Add><Item><Data>` + dn + `</Data>` + target + `</Item></Add>`,
			want:    `<Add><Item><Data>` + quotedDN + `</Data>` + target + `</Item></Add>`,
		},
		{
			name: "whitespace around the locuri is tolerated",
			profile: "<Add>\n  <Item>\n    <Target>\n      <LocURI>\n        " + testDeviceSubjectNameLocURI +
				"\n      </LocURI>\n    </Target>\n    <Data>" + dn + "</Data>\n  </Item>\n</Add>",
			want: "<Add>\n  <Item>\n    <Target>\n      <LocURI>\n        " + testDeviceSubjectNameLocURI +
				"\n      </LocURI>\n    </Target>\n    <Data>" + quotedDN + "</Data>\n  </Item>\n</Add>",
		},
		{
			name:    "commented out data inside the item is not touched",
			profile: `<Add><Item>` + target + `<!-- <Data>` + dn + `</Data> --><Data>` + dn + `</Data></Item></Add>`,
			want:    `<Add><Item>` + target + `<!-- <Data>` + bareDN + `</Data> --><Data>` + quotedDN + `</Data></Item></Add>`,
		},
		{
			name:    "a profile holding several items only rewrites the one that matched",
			profile: testSyncMLItem(testDeviceSubjectNameLocURI, dn) + testSyncMLItem("./Device/Test", dn),
			want:    testSyncMLItem(testDeviceSubjectNameLocURI, quotedDN) + testSyncMLItem("./Device/Test", bareDN),
		},
		{
			name:    "unparseable xml is returned with nothing quoted",
			profile: `<Add><Item>` + target + `<Data>` + dn + `</Data></Item>`,
			want:    `<Add><Item>` + target + `<Data>` + bareDN + `</Data></Item>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, deliverSubjectName(t, tt.profile, "user+idp@example.com"))
		})
	}
}

func TestSCEPSubjectNameQuotingTargeting(t *testing.T) {
	const data = `CN=$FLEET_VAR_HOST_UUID,O=Fleet QA`
	const quoted = `CN="user+idp@example.com",O=Fleet QA`
	const bare = `CN=user+idp@example.com,O=Fleet QA`

	tests := []struct {
		name     string
		locURI   string
		wantData string
	}{
		{
			name:     "device scoped subject name is quoted",
			locURI:   testDeviceSubjectNameLocURI,
			wantData: quoted,
		},
		{
			name:     "user scoped subject name is quoted",
			locURI:   testUserSubjectNameLocURI,
			wantData: quoted,
		},
		{
			name:     "subject name without the device scope prefix is quoted",
			locURI:   "Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/SubjectName",
			wantData: quoted,
		},
		{
			name:     "subject alternative names is left alone",
			locURI:   testSANLocURI,
			wantData: bare,
		},
		{
			name:     "unrelated locuri is left alone",
			locURI:   "./Device/Test",
			wantData: bare,
		},
		{
			name:     "locuri ending in subject name outside the scep path is left alone",
			locURI:   "./Device/Vendor/MSFT/Something/Else/Install/SubjectName",
			wantData: bare,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deliverSubjectName(t, testSyncMLItem(tt.locURI, data), "user+idp@example.com")
			require.Equal(t, testSyncMLItem(tt.locURI, tt.wantData), got)
		})
	}
}

func TestIndexDNSeparator(t *testing.T) {
	tests := []struct {
		name string
		dn   string
		want int
	}{
		{name: "no separator", dn: `CN=a`, want: -1},
		{name: "comma", dn: `CN=a,O=b`, want: 4},
		{name: "plus", dn: `CN=a+OU=b`, want: 4},
		{name: "semicolon", dn: `CN=a;O=b`, want: 4},
		{name: "separators inside quotes are skipped", dn: `CN="Doe, Jane+X;Y",O=b`, want: 18},
		{name: "unterminated quote hides the rest", dn: `CN="Doe, Jane,O=b`, want: -1},
		{name: "first of several", dn: `CN=a,OU=b;O=c`, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, indexDNSeparator(tt.dn))
		})
	}
}
