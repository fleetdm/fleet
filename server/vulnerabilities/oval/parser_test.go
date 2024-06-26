package oval

import (
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	oval_input "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/input"
	oval_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/parsed"
	"github.com/stretchr/testify/require"
)

func TestOvalParser(t *testing.T) {
	ubuntuOvalXml := `
<oval_definitions
    xmlns="http://oval.mitre.org/XMLSchema/oval-definitions-5"
    xmlns:ind="http://oval.mitre.org/XMLSchema/oval-definitions-5#independent"
    xmlns:oval="http://oval.mitre.org/XMLSchema/oval-common-5"
    xmlns:unix="http://oval.mitre.org/XMLSchema/oval-definitions-5#unix"
    xmlns:linux="http://oval.mitre.org/XMLSchema/oval-definitions-5#linux"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://oval.mitre.org/XMLSchema/oval-common-5 oval-common-schema.xsd   http://oval.mitre.org/XMLSchema/oval-definitions-5 oval-definitions-schema.xsd   http://oval.mitre.org/XMLSchema/oval-definitions-5#independent independent-definitions-schema.xsd   http://oval.mitre.org/XMLSchema/oval-definitions-5#unix unix-definitions-schema.xsd   http://oval.mitre.org/XMLSchema/oval-definitions-5#macos linux-definitions-schema.xsd">
	<definitions>
	   <definition id="oval:com.ubuntu.jammy:def:53901000000" version="1" class="patch">
           <metadata>
              <title>5390-1 -- Linux kernel vulnerabilities</title>
              <affected family="unix">
                 <platform>Ubuntu 22.04 LTS</platform>
              </affected>
              <reference source="USN" ref_url="https://ubuntu.com/security/notices/USN-5390-1" ref_id="USN-5390-1"/>
              <reference source="CVE" ref_url="https://ubuntu.com/security/CVE-2022-1015" ref_id="CVE-2022-1015"/>
              <reference source="CVE" ref_url="https://ubuntu.com/security/CVE-2022-1016" ref_id="CVE-2022-1016"/>
              <reference source="CVE" ref_url="https://ubuntu.com/security/CVE-2022-26490" ref_id="CVE-2022-26490"/>
              <description>Some long description</description>
              <advisory from="security@ubuntu.com">
                 <severity>High</severity>
                 <issued date="2022-04-26"/>
              </advisory>
           </metadata>
           <criteria operator="OR">
		     <criterion test_ref="oval:com.ubuntu.jammy:tst:540210000000" comment="Long Term Support" />
           </criteria>
        </definition>
		<definition id="oval:com.ubuntu.jammy:def:54291000000" version="1" class="patch">
           <metadata>
              <title>5429-1 -- Bind vulnerability</title>
              <affected family="unix">
                 <platform>Ubuntu 22.04 LTS</platform>
              </affected>
              <reference source="USN" ref_url="https://ubuntu.com/security/notices/USN-5429-1" ref_id="USN-5429-1"/>
              <reference source="CVE" ref_url="https://ubuntu.com/security/CVE-2022-1183" ref_id="CVE-2022-1183"/>
              <description>Some desc</description>
              <advisory from="security@ubuntu.com">
                 <severity>Medium</severity>
                 <issued date="2022-05-18"/>
              </advisory>
           </metadata>
           <criteria operator="OR">
              <criterion test_ref="oval:com.ubuntu.jammy:tst:542910000000" comment="Long Term Support" />
           </criteria>
        </definition>
	</definitions>
	<definition id="oval:com.ubuntu.jammy:def:55441000000" version="1" class="patch">
		<metadata>
			<title>USN-5544-1 -- Linux kernel vulnerabilities</title>
			<affected family="unix">
				<platform>Ubuntu 22.04 LTS</platform>
			</affected>
			<reference source="USN" ref_id="USN-5544-1" ref_url="https://ubuntu.com/security/notices/USN-5544-1"/>
			<reference source="CVE" ref_id="CVE-2022-1652" ref_url="https://ubuntu.com/security/CVE-2022-1652"/>
			<reference source="CVE" ref_id="CVE-2022-1679" ref_url="https://ubuntu.com/security/CVE-2022-1679"/>
			<reference source="CVE" ref_id="CVE-2022-28893" ref_url="https://ubuntu.com/security/CVE-2022-28893"/>
			<reference source="CVE" ref_id="CVE-2022-34918" ref_url="https://ubuntu.com/security/CVE-2022-34918"/>
			<description>Some long description</description>
			<advisory from="security@ubuntu.com">
				<severity>High</severity>
				<issued date="2022-08-02"/>
				<cve href="https://ubuntu.com/security/CVE-2022-1652" priority="medium" public="20220602" cvss_score="7.8" cvss_vector="CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H" cvss_severity="high" usns="5500-1,5505-1,5513-1,5529-1,5544-1,5560-1,5560-2,5562-1,5564-1,5566-1,5582-1">CVE-2022-1652</cve>
				<cve href="https://ubuntu.com/security/CVE-2022-1679" priority="medium" public="20220516" cvss_score="7.8" cvss_vector="CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H" cvss_severity="high" usns="5500-1,5505-1,5513-1,5529-1,5517-1,5544-1,5560-1,5560-2,5562-1,5564-1,5566-1,5582-1">CVE-2022-1679</cve>
				<cve href="https://ubuntu.com/security/CVE-2022-28893" priority="medium" public="20220411" cvss_score="7.8" cvss_vector="CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H" cvss_severity="high" usns="5544-1,5562-1,5564-1,5566-1,5582-1">CVE-2022-28893</cve>
				<cve href="https://ubuntu.com/security/CVE-2022-34918" priority="high" public="20220704" cvss_score="7.8" cvss_vector="CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H" cvss_severity="high" usns="5540-1,5544-1,5545-1,5560-1,5560-2,5562-1,5564-1,5566-1,5582-1">CVE-2022-34918</cve>
				
			</advisory>
		</metadata>
		<criteria>
			<extend_definition definition_ref="oval:com.ubuntu.jammy:def:100" comment="Ubuntu 22.04 LTS (jammy) is installed." applicability_check="true" />
			<criteria operator="OR">
				<criteria operator="AND">
					<criterion test_ref="oval:com.ubuntu.jammy:tst:554410000000" comment="Long Term Support" />
					<criterion test_ref="oval:com.ubuntu.jammy:tst:554410000010" comment="Long Term Support" />
				</criteria>
			</criteria>
		</criteria>
	</definition>
	<tests>
		<linux:dpkginfo_test id="oval:com.ubuntu.jammy:tst:540210000000" version="1" check_existence="at_least_one_exists" check="at least one" comment="Long Term Support">
           <linux:object object_ref="oval:com.ubuntu.jammy:obj:540210000000"/>
           <linux:state state_ref="oval:com.ubuntu.jammy:ste:540210000000"/>
        </linux:dpkginfo_test>
		<linux:dpkginfo_test id="oval:com.ubuntu.jammy:tst:542910000000" version="1" check_existence="at_least_one_exists" check="at least one" comment="Long Term Support">
           <linux:object object_ref="oval:com.ubuntu.jammy:obj:542910000000"/>
           <linux:state state_ref="oval:com.ubuntu.jammy:ste:542910000000"/>
        </linux:dpkginfo_test>
		<ind:variable_test id="oval:com.ubuntu.jammy:tst:554410000010" version="1" check="all" check_existence="all_exist" comment="kernel version comparison">
            <ind:object object_ref="oval:com.ubuntu.jammy:obj:554410000010"/>
            <ind:state state_ref="oval:com.ubuntu.jammy:ste:554410000010"/>
        </ind:variable_test>
	</tests>
	<unix:uname_test check="at least one" comment="Is kernel 5.15.0-\d+(-generic|-generic-64k|-generic-lpae|-lowlatency|-lowlatency-64k) currently running?" id="oval:com.ubuntu.jammy:tst:554410000000" version="1">
            <unix:object object_ref="oval:com.ubuntu.jammy:obj:554410000000"/>
            <unix:state state_ref="oval:com.ubuntu.jammy:ste:554410000000"/>
	</unix:uname_test>
	<objects>
		<linux:dpkginfo_object id="oval:com.ubuntu.jammy:obj:540210000000" version="1" comment="Long Term Support">
           <linux:name var_ref="oval:com.ubuntu.jammy:var:540210000000" var_check="at least one" />
        </linux:dpkginfo_object>
		<linux:dpkginfo_object id="oval:com.ubuntu.jammy:obj:542910000000" version="1" comment="Long Term Support">
           <linux:name var_ref="oval:com.ubuntu.jammy:var:542910000000" var_check="at least one" />
        </linux:dpkginfo_object>
		<unix:uname_object id="oval:com.ubuntu.jammy:obj:554410000000" version="1"/>
		<ind:variable_object id="oval:com.ubuntu.jammy:obj:554410000010" version="1">
            <ind:var_ref>oval:com.ubuntu.jammy:var:554410000000</ind:var_ref>
        </ind:variable_object>
	</objects>
	<states>
		<linux:dpkginfo_state id="oval:com.ubuntu.jammy:ste:540210000000" version="1" comment="Long Term Support">
           <linux:evr datatype="evr_string" operation="less than">0:3.0.2-0ubuntu1.1</linux:evr>
        </linux:dpkginfo_state>
		<linux:dpkginfo_state id="oval:com.ubuntu.jammy:ste:542910000000" version="1" comment="Long Term Support">
           <linux:evr datatype="evr_string" operation="less than">1:9.18.1-1ubuntu1.1</linux:evr>
        </linux:dpkginfo_state>
		<unix:uname_state id="oval:com.ubuntu.jammy:ste:554410000000" version="1">
            <unix:os_release operation="pattern match">5.15.0-\d+(-generic|-generic-64k|-generic-lpae|-lowlatency|-lowlatency-64k)</unix:os_release>
        </unix:uname_state>
		<ind:variable_state id="oval:com.ubuntu.jammy:ste:554410000010" version="1">
            <ind:value datatype="debian_evr_string" operation="less than">0:5.15.0-43</ind:value>
        </ind:variable_state>
	</states>
	<variables>
		<constant_variable id="oval:com.ubuntu.jammy:var:540210000000" version="1" datatype="string" comment="Long Term Support">
			<value>libssl-dev</value>
            <value>openssl</value>
            <value>libssl-doc</value>
            <value>libssl3</value>
        </constant_variable>
		<constant_variable id="oval:com.ubuntu.jammy:var:542910000000" version="1" datatype="string" comment="Long Term Support">
           <value>dnsutils</value>
            <value>bind9-libs</value>
            <value>bind9utils</value>
            <value>bind9-dev</value>
            <value>bind9-doc</value>
            <value>bind9-utils</value>
            <value>bind9</value>
            <value>bind9-dnsutils</value>
            <value>bind9-host</value>
        </constant_variable>
	</variables>
</oval_definitions>
		`
	rhelOvalXML := `
<?xml version="1.0" encoding="utf-8"?>
<oval_definitions xmlns="http://oval.mitre.org/XMLSchema/oval-definitions-5" xmlns:oval="http://oval.mitre.org/XMLSchema/oval-common-5" xmlns:unix-def="http://oval.mitre.org/XMLSchema/oval-definitions-5#unix" xmlns:red-def="http://oval.mitre.org/XMLSchema/oval-definitions-5#linux" xmlns:ind-def="http://oval.mitre.org/XMLSchema/oval-definitions-5#independent" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://oval.mitre.org/XMLSchema/oval-common-5 oval-common-schema.xsd http://oval.mitre.org/XMLSchema/oval-definitions-5 oval-definitions-schema.xsd http://oval.mitre.org/XMLSchema/oval-definitions-5#unix unix-definitions-schema.xsd http://oval.mitre.org/XMLSchema/oval-definitions-5#linux linux-definitions-schema.xsd">
	<generator>
		<oval:product_name>Red Hat OVAL Patch Definition Merger</oval:product_name>
		<oval:product_version>3</oval:product_version>
		<oval:schema_version>5.10</oval:schema_version>
		<oval:timestamp>2022-06-04T02:29:15</oval:timestamp>
		<oval:content_version>1654309755</oval:content_version>
	</generator>
	<definitions>
		<definition class="patch" id="oval:com.redhat.rhsa:def:20224584" version="635">
			<metadata>
				<title>RHSA-2022:4584: zlib security update (Important)</title>
				<affected family="unix">
					<platform>Red Hat Enterprise Linux 9</platform>
				</affected>
				<reference ref_id="RHSA-2022:4584" ref_url="https://access.redhat.com/errata/RHSA-2022:4584" source="RHSA"/>
				<reference ref_id="CVE-2018-25032" ref_url="https://access.redhat.com/security/cve/CVE-2018-25032" source="CVE"/>
				<description></description>
				<advisory from="secalert@redhat.com">
					<severity>Important</severity>
					<rights>Copyright 2022 Red Hat, Inc.</rights>
					<issued date="2022-05-17"/>
					<updated date="2022-05-17"/>
					<cve cvss3="8.2/CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:L/A:H" cwe="CWE-119" href="https://access.redhat.com/security/cve/CVE-2018-25032" impact="important" public="20180420">CVE-2018-25032</cve>
					<bugzilla href="https://bugzilla.redhat.com/2067945" id="2067945">CVE-2018-25032 zlib: A flaw found in zlib when compressing (not decompressing) certain inputs</bugzilla>
					<affected_cpe_list>
						<cpe>cpe:/a:redhat:enterprise_linux:9</cpe>
						<cpe>cpe:/a:redhat:enterprise_linux:9::appstream</cpe>
						<cpe>cpe:/a:redhat:enterprise_linux:9::crb</cpe>
						<cpe>cpe:/o:redhat:enterprise_linux:9</cpe>
						<cpe>cpe:/o:redhat:enterprise_linux:9::baseos</cpe>
					</affected_cpe_list>
				</advisory>
			</metadata>
			<criteria operator="OR">
				<criterion comment="Red Hat Enterprise Linux must be installed" test_ref="oval:com.redhat.rhsa:tst:20221728048"/>
				<criteria operator="AND">
					<criterion comment="Red Hat Enterprise Linux 9 is installed" test_ref="oval:com.redhat.rhsa:tst:20221728047"/>
					<criteria operator="OR">
						<criteria operator="AND">
							<criterion comment="zlib is earlier than 0:1.2.11-31.el9_0.1" test_ref="oval:com.redhat.rhsa:tst:20224584001"/>
							<criterion comment="zlib is signed with Red Hat redhatrelease2 key" test_ref="oval:com.redhat.rhsa:tst:20224584002"/>
						</criteria>
						<criteria operator="AND">
							<criterion comment="zlib-devel is earlier than 0:1.2.11-31.el9_0.1" test_ref="oval:com.redhat.rhsa:tst:20224584003"/>
							<criterion comment="zlib-devel is signed with Red Hat redhatrelease2 key" test_ref="oval:com.redhat.rhsa:tst:20224584004"/>
						</criteria>
						<criteria operator="AND">
							<criterion comment="zlib-static is earlier than 0:1.2.11-31.el9_0.1" test_ref="oval:com.redhat.rhsa:tst:20224584005"/>
							<criterion comment="zlib-static is signed with Red Hat redhatrelease2 key" test_ref="oval:com.redhat.rhsa:tst:20224584006"/>
						</criteria>
					</criteria>
				</criteria>
			</criteria>
		</definition>
	</definitions>
	<tests>
		<red-def:rpmverifyfile_test check="none satisfy" comment="Red Hat Enterprise Linux must be installed" id="oval:com.redhat.rhsa:tst:20221728048" version="635">
			<red-def:object object_ref="oval:com.redhat.rhsa:obj:20221728024"/>
			<red-def:state state_ref="oval:com.redhat.rhsa:ste:20221728005"/>
		</red-def:rpmverifyfile_test>
		<red-def:rpmverifyfile_test check="at least one" comment="Red Hat Enterprise Linux 9 is installed" id="oval:com.redhat.rhsa:tst:20221728047" version="635">
			<red-def:object object_ref="oval:com.redhat.rhsa:obj:20221728024"/>
			<red-def:state state_ref="oval:com.redhat.rhsa:ste:20221728004"/>
		</red-def:rpmverifyfile_test>
		<red-def:rpminfo_test check="at least one" comment="zlib is earlier than 0:1.2.11-31.el9_0.1" id="oval:com.redhat.rhsa:tst:20224584001" version="635">
			<red-def:object object_ref="oval:com.redhat.rhsa:obj:20224584001"/>
			<red-def:state state_ref="oval:com.redhat.rhsa:ste:20224584001"/>
		</red-def:rpminfo_test>
		<red-def:rpminfo_test check="at least one" comment="zlib is signed with Red Hat redhatrelease2 key" id="oval:com.redhat.rhsa:tst:20224584002" version="635">
			<red-def:object object_ref="oval:com.redhat.rhsa:obj:20224584001"/>
			<red-def:state state_ref="oval:com.redhat.rhsa:ste:20221728002"/>
		</red-def:rpminfo_test>
		<red-def:rpminfo_test check="at least one" comment="zlib-devel is earlier than 0:1.2.11-31.el9_0.1" id="oval:com.redhat.rhsa:tst:20224584003" version="635">
			<red-def:object object_ref="oval:com.redhat.rhsa:obj:20224584002"/>
			<red-def:state state_ref="oval:com.redhat.rhsa:ste:20224584001"/>
		</red-def:rpminfo_test>
		<red-def:rpminfo_test check="at least one" comment="zlib-devel is signed with Red Hat redhatrelease2 key" id="oval:com.redhat.rhsa:tst:20224584004" version="635">
			<red-def:object object_ref="oval:com.redhat.rhsa:obj:20224584002"/>
			<red-def:state state_ref="oval:com.redhat.rhsa:ste:20221728002"/>
		</red-def:rpminfo_test>
		<red-def:rpminfo_test check="at least one" comment="zlib-static is earlier than 0:1.2.11-31.el9_0.1" id="oval:com.redhat.rhsa:tst:20224584005" version="635">
			<red-def:object object_ref="oval:com.redhat.rhsa:obj:20224584003"/>
			<red-def:state state_ref="oval:com.redhat.rhsa:ste:20224584001"/>
		</red-def:rpminfo_test>
		<red-def:rpminfo_test check="at least one" comment="zlib-static is signed with Red Hat redhatrelease2 key" id="oval:com.redhat.rhsa:tst:20224584006" version="635">
			<red-def:object object_ref="oval:com.redhat.rhsa:obj:20224584003"/>
			<red-def:state state_ref="oval:com.redhat.rhsa:ste:20221728002"/>
		</red-def:rpminfo_test>
	</tests>
	<objects>
		<red-def:rpmverifyfile_object id="oval:com.redhat.rhsa:obj:20221728024" version="635">
			<red-def:behaviors noconfigfiles="true" noghostfiles="true" nogroup="true" nolinkto="true" nomd5="true" nomode="true" nomtime="true" nordev="true" nosize="true" nouser="true"/>
			<red-def:name operation="pattern match"/>
			<red-def:epoch operation="pattern match"/>
			<red-def:version operation="pattern match"/>
			<red-def:release operation="pattern match"/>
			<red-def:arch operation="pattern match"/>
			<red-def:filepath>/etc/redhat-release</red-def:filepath>
		</red-def:rpmverifyfile_object>
		<red-def:rpminfo_object id="oval:com.redhat.rhsa:obj:20224584001" version="635">
			<red-def:name>zlib</red-def:name>
		</red-def:rpminfo_object>
		<red-def:rpminfo_object id="oval:com.redhat.rhsa:obj:20224584002" version="635">
			<red-def:name>zlib-devel</red-def:name>
		</red-def:rpminfo_object>
		<red-def:rpminfo_object id="oval:com.redhat.rhsa:obj:20224584003" version="635">
			<red-def:name>zlib-static</red-def:name>
		</red-def:rpminfo_object>
	</objects>
	<states>
		<red-def:rpmverifyfile_state id="oval:com.redhat.rhsa:ste:20221728005" version="635">
			<red-def:name operation="pattern match">^redhat-release</red-def:name>
		</red-def:rpmverifyfile_state>
		<red-def:rpmverifyfile_state id="oval:com.redhat.rhsa:ste:20221728004" version="635">
			<red-def:name operation="pattern match">^redhat-release</red-def:name>
			<red-def:version operation="pattern match">^7[^\d]</red-def:version>
		</red-def:rpmverifyfile_state>
		<red-def:rpminfo_state id="oval:com.redhat.rhsa:ste:20224584001" version="635">
			<red-def:arch datatype="string" operation="pattern match">aarch64|i686|ppc64le|s390x|x86_64</red-def:arch>
			<red-def:evr datatype="evr_string" operation="less than">0:1.2.11-31.el9_0.1</red-def:evr>
		</red-def:rpminfo_state>
		<red-def:rpminfo_state id="oval:com.redhat.rhsa:ste:20221728002" version="635">
			<red-def:signature_keyid operation="equals">199e2f91fd431d51</red-def:signature_keyid>
		</red-def:rpminfo_state>
	</states>
</oval_definitions>
`
	t.Run("#parseUbuntuXML", func(t *testing.T) {
		r := strings.NewReader(ubuntuOvalXml)

		result, err := parseUbuntuXML(r)
		require.NoError(t, err)

		require.Equal(t, result.Definitions[0].Id, "oval:com.ubuntu.jammy:def:53901000000")
		require.Equal(t, result.Definitions[1].Id, "oval:com.ubuntu.jammy:def:54291000000")

		require.ElementsMatch(t, result.Definitions[0].Vulnerabilities, []oval_input.ReferenceXML{
			{Id: "USN-5390-1"},
			{Id: "CVE-2022-1015"},
			{Id: "CVE-2022-1016"},
			{Id: "CVE-2022-26490"},
		})
		require.ElementsMatch(t, result.Definitions[1].Vulnerabilities, []oval_input.ReferenceXML{
			{Id: "USN-5429-1"},
			{Id: "CVE-2022-1183"},
		})

		require.Equal(t, result.Definitions[0].Criteria.Operator, "OR")
		require.Equal(t, result.Definitions[0].Criteria.Criteriums[0].TestId, "oval:com.ubuntu.jammy:tst:540210000000")
		require.Equal(t, result.Definitions[1].Criteria.Operator, "OR")
		require.Equal(t, result.Definitions[1].Criteria.Criteriums[0].TestId, "oval:com.ubuntu.jammy:tst:542910000000")

		firstTest := result.DpkgInfoTests[0]
		require.Equal(t, firstTest.Id, "oval:com.ubuntu.jammy:tst:540210000000")
		require.Equal(t, firstTest.CheckExistence, "at_least_one_exists")
		require.Equal(t, firstTest.Check, "at least one")
		require.Empty(t, firstTest.StateOperator)
		require.Equal(t, firstTest.Object.Id, "oval:com.ubuntu.jammy:obj:540210000000")
		require.Len(t, firstTest.States, 1)
		require.Equal(t, firstTest.States[0].Id, "oval:com.ubuntu.jammy:ste:540210000000")

		secondTest := result.DpkgInfoTests[1]
		require.Equal(t, secondTest.Id, "oval:com.ubuntu.jammy:tst:542910000000")
		require.Equal(t, secondTest.CheckExistence, "at_least_one_exists")
		require.Equal(t, secondTest.Check, "at least one")
		require.Empty(t, secondTest.StateOperator)
		require.Equal(t, secondTest.Object.Id, "oval:com.ubuntu.jammy:obj:542910000000")
		require.Len(t, secondTest.States, 1)
		require.Equal(t, secondTest.States[0].Id, "oval:com.ubuntu.jammy:ste:542910000000")

		firstObject := result.DpkgInfoObjects[0]
		require.Equal(t, firstObject.Id, "oval:com.ubuntu.jammy:obj:540210000000")
		require.Equal(t, firstObject.Name.VarRef, "oval:com.ubuntu.jammy:var:540210000000")
		require.Empty(t, firstObject.Name.Value)
		require.Equal(t, firstObject.Name.VarCheck, "at least one")

		secondObject := result.DpkgInfoObjects[1]
		require.Equal(t, secondObject.Id, "oval:com.ubuntu.jammy:obj:542910000000")
		require.Equal(t, secondObject.Name.VarRef, "oval:com.ubuntu.jammy:var:542910000000")
		require.Empty(t, secondObject.Name.Value)
		require.Equal(t, secondObject.Name.VarCheck, "at least one")

		firstState := result.DpkgInfoStates[0]
		require.Equal(t, firstState.Id, "oval:com.ubuntu.jammy:ste:540210000000")
		require.Nil(t, firstState.Arch)
		require.Nil(t, firstState.Epoch)
		require.Nil(t, firstState.Name)
		require.Nil(t, firstState.Release)
		require.Nil(t, firstState.Version)
		require.Equal(t, firstState.Evr.Value, "0:3.0.2-0ubuntu1.1")
		require.Equal(t, firstState.Evr.Op, "less than")

		secondState := result.DpkgInfoStates[1]
		require.Equal(t, secondState.Id, "oval:com.ubuntu.jammy:ste:542910000000")
		require.Nil(t, secondState.Arch)
		require.Nil(t, secondState.Epoch)
		require.Nil(t, secondState.Name)
		require.Nil(t, secondState.Release)
		require.Nil(t, secondState.Version)
		require.Equal(t, secondState.Evr.Value, "1:9.18.1-1ubuntu1.1")
		require.Equal(t, secondState.Evr.Op, "less than")

		expectedVariables := map[string]oval_input.ConstantVariableXML{
			"oval:com.ubuntu.jammy:var:540210000000": {
				Id:       "oval:com.ubuntu.jammy:var:540210000000",
				DataType: "string",
				Values: []string{
					"libssl-dev",
					"openssl",
					"libssl-doc",
					"libssl3",
				},
			},
			"oval:com.ubuntu.jammy:var:542910000000": {
				Id:       "oval:com.ubuntu.jammy:var:542910000000",
				DataType: "string",
				Values: []string{
					"dnsutils",
					"bind9-libs",
					"bind9utils",
					"bind9-dev",
					"bind9-doc",
					"bind9-utils",
					"bind9",
					"bind9-dnsutils",
					"bind9-host",
				},
			},
		}
		require.Equal(t, result.Variables, expectedVariables)
	})

	t.Run("#mapToUbuntuResult", func(t *testing.T) {
		r := strings.NewReader(ubuntuOvalXml)

		xmlResult, err := parseUbuntuXML(r)
		require.NoError(t, err)

		result, err := mapToUbuntuResult(xmlResult)
		require.NoError(t, err)

		var expectedVulns []string

		for _, d := range xmlResult.Definitions {
			for _, v := range d.Vulnerabilities {
				expectedVulns = append(expectedVulns, v.Id)
			}
		}

		var actualVulns []string
		var actualTestIds []int

		for _, d := range result.Definitions {
			actualTestIds = append(actualTestIds, d.CollectTestIds()...)
			actualVulns = append(actualVulns, d.Vulnerabilities...)
		}

		require.Equal(t, expectedVulns, actualVulns)

		expectedTestIds := []int{540210000000, 542910000000, 554410000000, 554410000010}
		require.ElementsMatch(t, expectedTestIds, actualTestIds)

		require.Len(t, result.PackageTests, 2)

		testOne, ok := result.PackageTests[540210000000]
		require.True(t, ok)
		require.ElementsMatch(t, testOne.Objects, []string{
			"libssl-dev",
			"openssl",
			"libssl-doc",
			"libssl3",
		})

		testTwo, ok := result.PackageTests[542910000000]
		require.True(t, ok)
		require.ElementsMatch(t, testTwo.Objects, []string{
			"dnsutils",
			"bind9-libs",
			"bind9utils",
			"bind9-dev",
			"bind9-doc",
			"bind9-utils",
			"bind9",
			"bind9-dnsutils",
			"bind9-host",
		})

		require.Len(t, result.UnameTests, 2)
		matchState := []oval_parsed.ObjectStateString{"pattern match|5.15.0-\\d+(-generic|-generic-64k|-generic-lpae|-lowlatency|-lowlatency-64k)"}
		require.ElementsMatch(t, result.UnameTests[554410000000].States, matchState)

		variableState := []oval_parsed.ObjectStateString{"less than|0:5.15.0-43"}
		require.ElementsMatch(t, result.UnameTests[554410000010].States, variableState)
	})

	t.Run("#parseRhelXML", func(t *testing.T) {
		r := strings.NewReader(rhelOvalXML)

		result, err := parseRhelXML(r)
		require.NoError(t, err)

		require.Equal(t, result.Definitions[0].Id, "oval:com.redhat.rhsa:def:20224584")

		require.ElementsMatch(t, result.Definitions[0].Vulnerabilities, []oval_input.ReferenceXML{
			{Id: "RHSA-2022:4584"},
			{Id: "CVE-2018-25032"},
		})

		require.Equal(t, result.Definitions[0].Criteria.Operator, "OR")
		require.Equal(t, result.Definitions[0].Criteria.Criteriums[0].TestId, "oval:com.redhat.rhsa:tst:20221728048")
		require.Equal(t, result.Definitions[0].Criteria.Criterias[0].Operator, "AND")
		require.Equal(t, result.Definitions[0].Criteria.Criterias[0].Criteriums[0].TestId, "oval:com.redhat.rhsa:tst:20221728047")
		require.Equal(t, result.Definitions[0].Criteria.Criterias[0].Criterias[0].Operator, "OR")

		require.Equal(t, result.Definitions[0].Criteria.Criterias[0].Criterias[0].Criterias[0].Operator, "AND")
		require.Equal(t, result.Definitions[0].Criteria.Criterias[0].Criterias[0].Criterias[0].Criteriums[0].TestId, "oval:com.redhat.rhsa:tst:20224584001")
		require.Equal(t, result.Definitions[0].Criteria.Criterias[0].Criterias[0].Criterias[0].Criteriums[1].TestId, "oval:com.redhat.rhsa:tst:20224584002")

		require.Equal(t, result.Definitions[0].Criteria.Criterias[0].Criterias[0].Criterias[1].Operator, "AND")
		require.Equal(t, result.Definitions[0].Criteria.Criterias[0].Criterias[0].Criterias[1].Criteriums[0].TestId, "oval:com.redhat.rhsa:tst:20224584003")
		require.Equal(t, result.Definitions[0].Criteria.Criterias[0].Criterias[0].Criterias[1].Criteriums[1].TestId, "oval:com.redhat.rhsa:tst:20224584004")

		require.Equal(t, result.Definitions[0].Criteria.Criterias[0].Criterias[0].Criterias[2].Operator, "AND")
		require.Equal(t, result.Definitions[0].Criteria.Criterias[0].Criterias[0].Criterias[2].Criteriums[0].TestId, "oval:com.redhat.rhsa:tst:20224584005")
		require.Equal(t, result.Definitions[0].Criteria.Criterias[0].Criterias[0].Criterias[2].Criteriums[1].TestId, "oval:com.redhat.rhsa:tst:20224584006")

		require.Len(t, result.RpmVerifyFileTests, 2)
		require.Equal(t, result.RpmVerifyFileTests[0].Id, "oval:com.redhat.rhsa:tst:20221728048")
		require.Equal(t, result.RpmVerifyFileTests[0].Object.Id, "oval:com.redhat.rhsa:obj:20221728024")
		require.Len(t, result.RpmVerifyFileTests[0].States, 1)
		require.Equal(t, result.RpmVerifyFileTests[0].States[0].Id, "oval:com.redhat.rhsa:ste:20221728005")
		require.Equal(t, result.RpmVerifyFileTests[1].Id, "oval:com.redhat.rhsa:tst:20221728047")
		require.Equal(t, result.RpmVerifyFileTests[1].Object.Id, "oval:com.redhat.rhsa:obj:20221728024")
		require.Len(t, result.RpmVerifyFileTests[1].States, 1)
		require.Equal(t, result.RpmVerifyFileTests[1].States[0].Id, "oval:com.redhat.rhsa:ste:20221728004")

		require.Len(t, result.RpmInfoTests, 6)
		require.Equal(t, result.RpmInfoTests[0].Id, "oval:com.redhat.rhsa:tst:20224584001")
		require.Empty(t, result.RpmInfoTests[0].CheckExistence)
		require.Equal(t, result.RpmInfoTests[0].Check, "at least one")
		require.Empty(t, result.RpmInfoTests[0].StateOperator)
		require.Equal(t, result.RpmInfoTests[0].Object.Id, "oval:com.redhat.rhsa:obj:20224584001")
		require.Len(t, result.RpmInfoTests[0].States, 1)
		require.Equal(t, result.RpmInfoTests[0].States[0].Id, "oval:com.redhat.rhsa:ste:20224584001")
		require.Equal(t, result.RpmInfoTests[1].Id, "oval:com.redhat.rhsa:tst:20224584002")
		require.Empty(t, result.RpmInfoTests[1].CheckExistence)
		require.Equal(t, result.RpmInfoTests[1].Check, "at least one")
		require.Empty(t, result.RpmInfoTests[1].StateOperator)
		require.Equal(t, result.RpmInfoTests[1].Object.Id, "oval:com.redhat.rhsa:obj:20224584001")
		require.Len(t, result.RpmInfoTests[1].States, 1)
		require.Equal(t, result.RpmInfoTests[1].States[0].Id, "oval:com.redhat.rhsa:ste:20221728002")
		require.Equal(t, result.RpmInfoTests[2].Id, "oval:com.redhat.rhsa:tst:20224584003")
		require.Empty(t, result.RpmInfoTests[2].CheckExistence)
		require.Equal(t, result.RpmInfoTests[2].Check, "at least one")
		require.Empty(t, result.RpmInfoTests[2].StateOperator)
		require.Equal(t, result.RpmInfoTests[2].Object.Id, "oval:com.redhat.rhsa:obj:20224584002")
		require.Len(t, result.RpmInfoTests[2].States, 1)
		require.Equal(t, result.RpmInfoTests[2].States[0].Id, "oval:com.redhat.rhsa:ste:20224584001")
		require.Equal(t, result.RpmInfoTests[3].Id, "oval:com.redhat.rhsa:tst:20224584004")
		require.Empty(t, result.RpmInfoTests[3].CheckExistence)
		require.Equal(t, result.RpmInfoTests[3].Check, "at least one")
		require.Empty(t, result.RpmInfoTests[3].StateOperator)
		require.Equal(t, result.RpmInfoTests[3].Object.Id, "oval:com.redhat.rhsa:obj:20224584002")
		require.Len(t, result.RpmInfoTests[3].States, 1)
		require.Equal(t, result.RpmInfoTests[3].States[0].Id, "oval:com.redhat.rhsa:ste:20221728002")
		require.Equal(t, result.RpmInfoTests[4].Id, "oval:com.redhat.rhsa:tst:20224584005")
		require.Empty(t, result.RpmInfoTests[4].CheckExistence)
		require.Equal(t, result.RpmInfoTests[4].Check, "at least one")
		require.Empty(t, result.RpmInfoTests[4].StateOperator)
		require.Equal(t, result.RpmInfoTests[4].Object.Id, "oval:com.redhat.rhsa:obj:20224584003")
		require.Len(t, result.RpmInfoTests[4].States, 1)
		require.Equal(t, result.RpmInfoTests[4].States[0].Id, "oval:com.redhat.rhsa:ste:20224584001")
		require.Equal(t, result.RpmInfoTests[5].Id, "oval:com.redhat.rhsa:tst:20224584006")
		require.Empty(t, result.RpmInfoTests[5].CheckExistence)
		require.Equal(t, result.RpmInfoTests[5].Check, "at least one")
		require.Empty(t, result.RpmInfoTests[5].StateOperator)
		require.Equal(t, result.RpmInfoTests[5].Object.Id, "oval:com.redhat.rhsa:obj:20224584003")
		require.Len(t, result.RpmInfoTests[5].States, 1)
		require.Equal(t, result.RpmInfoTests[5].States[0].Id, "oval:com.redhat.rhsa:ste:20221728002")

		require.Len(t, result.RpmInfoTestObjects, 3)
		require.Equal(t, result.RpmInfoTestObjects[0].Id, "oval:com.redhat.rhsa:obj:20224584001")
		require.Equal(t, result.RpmInfoTestObjects[0].Name.Value, "zlib")
		require.Empty(t, result.RpmInfoTestObjects[0].Name.VarRef)
		require.Empty(t, result.RpmInfoTestObjects[0].Name.VarCheck)

		require.Equal(t, result.RpmInfoTestObjects[1].Id, "oval:com.redhat.rhsa:obj:20224584002")
		require.Equal(t, result.RpmInfoTestObjects[1].Name.Value, "zlib-devel")
		require.Empty(t, result.RpmInfoTestObjects[1].Name.VarRef)
		require.Empty(t, result.RpmInfoTestObjects[1].Name.VarCheck)

		require.Equal(t, result.RpmInfoTestObjects[2].Id, "oval:com.redhat.rhsa:obj:20224584003")
		require.Equal(t, result.RpmInfoTestObjects[2].Name.Value, "zlib-static")
		require.Empty(t, result.RpmInfoTestObjects[2].Name.VarRef)
		require.Empty(t, result.RpmInfoTestObjects[2].Name.VarCheck)

		require.Len(t, result.RpmInfoTestStates, 2)
		require.Equal(t, result.RpmInfoTestStates[0].Id, "oval:com.redhat.rhsa:ste:20224584001")
		require.NotNil(t, result.RpmInfoTestStates[0].Arch)
		require.Equal(t, result.RpmInfoTestStates[0].Arch.Datatype, "string")
		require.Equal(t, result.RpmInfoTestStates[0].Arch.Op, "pattern match")
		require.Equal(t, result.RpmInfoTestStates[0].Arch.Value, "aarch64|i686|ppc64le|s390x|x86_64")

		require.Equal(t, result.RpmInfoTestStates[1].Id, "oval:com.redhat.rhsa:ste:20221728002")
		require.NotNil(t, result.RpmInfoTestStates[1].SignatureKeyId)
		require.Empty(t, result.RpmInfoTestStates[1].SignatureKeyId.Datatype)
		require.Equal(t, result.RpmInfoTestStates[1].SignatureKeyId.Op, "equals")
		require.Equal(t, result.RpmInfoTestStates[1].SignatureKeyId.Value, "199e2f91fd431d51")

		require.Len(t, result.RpmVerifyFileStates, 2)
		require.Equal(t, result.RpmVerifyFileStates[0].Id, "oval:com.redhat.rhsa:ste:20221728005")
		require.NotNil(t, result.RpmVerifyFileStates[0].Name)
		require.Equal(t, result.RpmVerifyFileStates[0].Name.Op, "pattern match")
		require.Empty(t, result.RpmVerifyFileStates[0].Name.Datatype)
		require.Equal(t, result.RpmVerifyFileStates[0].Name.Value, "^redhat-release")

		require.Equal(t, result.RpmVerifyFileStates[1].Id, "oval:com.redhat.rhsa:ste:20221728004")
		require.NotNil(t, result.RpmVerifyFileStates[1].Name)
		require.Equal(t, result.RpmVerifyFileStates[1].Name.Op, "pattern match")
		require.Equal(t, result.RpmVerifyFileStates[1].Name.Value, "^redhat-release")
		require.NotNil(t, result.RpmVerifyFileStates[1].Version)
		require.Equal(t, result.RpmVerifyFileStates[1].Version.Op, "pattern match")
		require.Equal(t, result.RpmVerifyFileStates[1].Version.Value, `^7[^\d]`)

		require.Len(t, result.RpmVerifyFileObjects, 1)
		require.Equal(t, result.RpmVerifyFileObjects[0].Id, "oval:com.redhat.rhsa:obj:20221728024")

		require.Equal(t, result.RpmVerifyFileObjects[0].Behaviors.NoConfigFiles, true)
		require.Equal(t, result.RpmVerifyFileObjects[0].Behaviors.NoGhostFiles, true)
		require.Equal(t, result.RpmVerifyFileObjects[0].Behaviors.NoGroup, true)
		require.Equal(t, result.RpmVerifyFileObjects[0].Behaviors.NoLinkTo, true)
		require.Equal(t, result.RpmVerifyFileObjects[0].Behaviors.NoMd5, true)
		require.Equal(t, result.RpmVerifyFileObjects[0].Behaviors.NoMode, true)
		require.Equal(t, result.RpmVerifyFileObjects[0].Behaviors.NoMtime, true)
		require.Equal(t, result.RpmVerifyFileObjects[0].Behaviors.NoRev, true)
		require.Equal(t, result.RpmVerifyFileObjects[0].Behaviors.NoSize, true)
		require.Equal(t, result.RpmVerifyFileObjects[0].Behaviors.NoUser, true)
		require.Equal(t, result.RpmVerifyFileObjects[0].Name.Op, "pattern match")
		require.Empty(t, result.RpmVerifyFileObjects[0].Name.Value)
		require.Equal(t, result.RpmVerifyFileObjects[0].Epoch.Op, "pattern match")
		require.Empty(t, result.RpmVerifyFileObjects[0].Epoch.Value)
		require.Equal(t, result.RpmVerifyFileObjects[0].Version.Op, "pattern match")
		require.Empty(t, result.RpmVerifyFileObjects[0].Version.Value)
		require.Equal(t, result.RpmVerifyFileObjects[0].Release.Op, "pattern match")
		require.Empty(t, result.RpmVerifyFileObjects[0].Arch.Value)
		require.Equal(t, result.RpmVerifyFileObjects[0].Arch.Op, "pattern match")
		require.Equal(t, result.RpmVerifyFileObjects[0].FilePath.Value, "/etc/redhat-release")
		require.Equal(t, result.RpmVerifyFileObjects[0].Arch.Op, "pattern match")
	})

	t.Run("#mapToRhelResult", func(t *testing.T) {
		r := strings.NewReader(rhelOvalXML)

		xmlResult, err := parseRhelXML(r)
		require.NoError(t, err)

		result, err := mapToRhelResult(xmlResult)
		require.NoError(t, err)

		var expectedVulns []string
		for _, d := range xmlResult.Definitions {
			for _, v := range d.Vulnerabilities {
				expectedVulns = append(expectedVulns, v.Id)
			}
		}

		var actualVulns []string
		var actualTestIds []int
		for _, d := range result.Definitions {
			actualTestIds = append(actualTestIds, d.CollectTestIds()...)
			actualVulns = append(actualVulns, d.Vulnerabilities...)
		}
		require.ElementsMatch(t, actualVulns, expectedVulns)
		require.ElementsMatch(t, actualTestIds, []int{
			20221728048,
			20221728047,
			20224584001,
			20224584002,
			20224584003,
			20224584004,
			20224584005,
			20224584006,
		})

		require.Len(t, result.RpmInfoTests, 6)

		testOne, ok := result.RpmInfoTests[20224584001]
		require.True(t, ok)
		require.ElementsMatch(t, testOne.Objects, []string{"zlib"})
		require.Len(t, testOne.States, 1)
		require.NotNil(t, testOne.States[0].Arch)
		require.NotNil(t, testOne.States[0].Evr)
		require.Equal(t, *testOne.States[0].Arch, oval_parsed.NewObjectStateString("pattern match", "aarch64|i686|ppc64le|s390x|x86_64"))
		require.Equal(t, *testOne.States[0].Evr, oval_parsed.NewObjectStateEvrString("less than", "0:1.2.11-31.el9_0.1"))

		testTwo, ok := result.RpmInfoTests[20224584002]
		require.True(t, ok)
		require.ElementsMatch(t, testTwo.Objects, []string{"zlib"})
		require.Len(t, testTwo.States, 1)
		require.NotNil(t, testTwo.States[0].SignatureKeyId)
		require.Equal(t, *testTwo.States[0].SignatureKeyId, oval_parsed.NewObjectStateString("equals", "199e2f91fd431d51"))

		testThree, ok := result.RpmInfoTests[20224584003]
		require.True(t, ok)
		require.ElementsMatch(t, testThree.Objects, []string{"zlib-devel"})
		require.Len(t, testThree.States, 1)
		require.NotNil(t, testThree.States[0].Arch)
		require.NotNil(t, testThree.States[0].Evr)
		require.Equal(t, *testThree.States[0].Arch, oval_parsed.NewObjectStateString("pattern match", "aarch64|i686|ppc64le|s390x|x86_64"))
		require.Equal(t, *testThree.States[0].Evr, oval_parsed.NewObjectStateEvrString("less than", "0:1.2.11-31.el9_0.1"))

		testFour, ok := result.RpmInfoTests[20224584004]
		require.True(t, ok)
		require.ElementsMatch(t, testFour.Objects, []string{"zlib-devel"})
		require.Len(t, testFour.States, 1)
		require.NotNil(t, testFour.States[0].SignatureKeyId)
		require.Equal(t, *testFour.States[0].SignatureKeyId, oval_parsed.NewObjectStateString("equals", "199e2f91fd431d51"))

		testFive, ok := result.RpmInfoTests[20224584005]
		require.True(t, ok)
		require.ElementsMatch(t, testFive.Objects, []string{"zlib-static"})
		require.Len(t, testFive.States, 1)
		require.NotNil(t, testFive.States[0].Arch)
		require.NotNil(t, testFive.States[0].Evr)
		require.Equal(t, *testFive.States[0].Arch, oval_parsed.NewObjectStateString("pattern match", "aarch64|i686|ppc64le|s390x|x86_64"))
		require.Equal(t, *testFive.States[0].Evr, oval_parsed.NewObjectStateEvrString("less than", "0:1.2.11-31.el9_0.1"))

		testSix, ok := result.RpmInfoTests[20224584006]
		require.True(t, ok)
		require.ElementsMatch(t, testSix.Objects, []string{"zlib-static"})
		require.Len(t, testFour.States, 1)
		require.NotNil(t, testFour.States[0].SignatureKeyId)
		require.Equal(t, *testFour.States[0].SignatureKeyId, oval_parsed.NewObjectStateString("equals", "199e2f91fd431d51"))
	})

	t.Run("RHEL OVAL definitions work with RHEL based distros", func(t *testing.T) {
		r := strings.NewReader(rhelOvalXML)

		xmlResult, err := parseRhelXML(r)
		require.NoError(t, err)

		result, err := mapToRhelResult(xmlResult)
		require.NoError(t, err)

		testCases := []fleet.OSVersion{
			{
				Platform: "rhel",
				Name:     "CentOS Linux 7.9.2009",
			},
			{
				Platform: "amzn",
				Name:     "Amazon Linux 2.0.0",
			},
			{
				Platform: "rhel",
				Name:     "Fedora Linux 19.0.0",
			},
			{
				Platform: "rhel",
				Name:     "Fedora Linux 20.0.0",
			},
			{
				Platform: "rhel",
				Name:     "Fedora Linux 21.0.0",
			},
		}

		for _, tCase := range testCases {
			rEval, err := result.RpmVerifyFileTests[20221728047].Eval(tCase)
			require.NoError(t, err)
			require.True(t, rEval, tCase)
		}
	})
}
