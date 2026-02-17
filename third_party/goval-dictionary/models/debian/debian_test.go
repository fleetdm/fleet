package debian

import (
	"encoding/xml"
	"reflect"
	"testing"

	"github.com/k0kubun/pp"

	"github.com/vulsio/goval-dictionary/models"
)

func TestWalkDebian(t *testing.T) {
	var tests = []struct {
		oval     string
		expected []models.Package
	}{
		{
			oval: `
<?xml version="1.0" ?>
<oval_definitions>
	<generator>
		<oval:product_name>Debian</oval:product_name>
		<oval:schema_version>5.3</oval:schema_version>
		<oval:timestamp>2017-04-07T03:47:55.188-04:00</oval:timestamp>
	</generator>
	<definitions>
		<definition class="vulnerability" id="oval:org.debian:def:20140001" version="1">
			<metadata>
				<title>CVE-2014-0001</title>
				<affected family="unix">
					<platform>Debian GNU/Linux 7.0</platform>
					<platform>Debian GNU/Linux 8.2</platform>
					<platform>Debian GNU/Linux 9.0</platform>
					<product>mysql-5.5</product>
				</affected>
				<reference ref_id="CVE-2014-0001" ref_url="http://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2014-0001" source="CVE"/>
				<description>Buffer overflow in client/mysql.cc in Oracle MySQL and MariaDB before 5.5.35 allows remote database servers to cause a denial of service (crash) and possibly execute arbitrary code via a long server version string.</description>
				<debian>
					<date>2014-05-03</date>
					<moreinfo>
DSA-2919
Several issues have been discovered in the MySQL database server. The
vulnerabilities are addressed by upgrading MySQL to the new upstream
version 5.5.37. Please see the MySQL 5.5 Release Notes and Oracle's
Critical Patch Update advisory for further details:
</moreinfo>
				</debian>
			</metadata>
			<criteria comment="Platform section" operator="OR">
				<criteria comment="Release section" operator="AND">
					<criterion comment="Debian 7.0 is installed" test_ref="oval:org.debian.oval:tst:1"/>
					<criteria comment="Architecture section" operator="OR">
						<criteria comment="Architecture independent section" operator="AND">
							<criterion comment="all architecture" test_ref="oval:org.debian.oval:tst:2"/>
							<criterion comment="mysql-5.5 DPKG is earlier than 5.5.37-0+wheezy1" test_ref="oval:org.debian.oval:tst:3"/>
						</criteria>
					</criteria>
				</criteria>
				<criteria comment="Release section" operator="AND">
					<criterion comment="Debian 8.2 is installed" test_ref="oval:org.debian.oval:tst:4"/>
					<criteria comment="Architecture section" operator="OR">
						<criteria comment="Architecture independent section" operator="AND">
							<criterion comment="all architecture" test_ref="oval:org.debian.oval:tst:2"/>
							<criterion comment="mysql-5.5 DPKG is earlier than 5.5.37-1" test_ref="oval:org.debian.oval:tst:5"/>
						</criteria>
					</criteria>
				</criteria>
				<criteria comment="Release section" operator="AND">
					<criterion comment="Debian 9.0 is installed" test_ref="oval:org.debian.oval:tst:6"/>
					<criteria comment="Architecture section" operator="OR">
						<criteria comment="Architecture independent section" operator="AND">
							<criterion comment="all architecture" test_ref="oval:org.debian.oval:tst:2"/>
							<criterion comment="mysql-5.5 DPKG is earlier than 5.5.37-1" test_ref="oval:org.debian.oval:tst:7"/>
						</criteria>
					</criteria>
					<criteria comment="Architecture section" operator="OR">
						<criteria comment="Architecture independent section" operator="AND">
							<criterion comment="all architecture" test_ref="oval:org.debian.oval:tst:2"/>
							<criterion comment="mysql-5.6 DPKG is earlier than 5.6.37-1" test_ref="oval:org.debian.oval:tst:7"/>
						</criteria>
					</criteria>
				</criteria>
			</criteria>
		</definition>
	</definitions>
</oval_definitions>
			`,
			expected: []models.Package{
				{
					Name:    "mysql-5.5",
					Version: "5.5.37-0+wheezy1",
				},
				{
					Name:    "mysql-5.5",
					Version: "5.5.37-1",
				},
				{
					Name:    "mysql-5.5",
					Version: "5.5.37-1",
				},
				{
					Name:    "mysql-5.6",
					Version: "5.6.37-1",
				},
			},
		},
	}

	for i, tt := range tests {
		var root *Root
		if err := xml.Unmarshal([]byte(tt.oval), &root); err != nil {
			t.Errorf("[%d] marshall error", i)
		}
		c := root.Definitions.Definitions[0].Criteria
		actual := collectDebianPacks(c)

		if !reflect.DeepEqual(tt.expected, actual) {
			e := pp.Sprintf("%v", tt.expected)
			a := pp.Sprintf("%v", actual)
			t.Errorf("[%d]: expected: %s\n, actual: %s\n", i, e, a)
		}
	}
}
