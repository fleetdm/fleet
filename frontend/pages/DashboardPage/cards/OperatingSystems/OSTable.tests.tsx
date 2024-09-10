import React from "react";
import { render, screen } from "@testing-library/react";

import OSTable from "./OSTable";

describe("Dashboard OS table", () => {
  it("renders data normally when present", () => {
    render(
      <OSTable
        currentTeamId={undefined}
        osVersions={[
          {
            os_version_id: 1,
            hosts_count: 234,
            name: "Microsoft Windows 11 Enterprise 22H2 10.0.22621",
            name_only: "Microsoft Windows 11 Enterprise 22H2",
            version: "1.2.3",
            platform: "windows",
            vulnerabilities: [],
          },
          {
            os_version_id: 2,
            hosts_count: 567,
            name: "Microsoft Windows 11 Pro 22H2 10.0.22621.3880",
            name_only: "Microsoft Windows 11 Pro 22H2",
            version: "4.5.6",
            platform: "windows",
            vulnerabilities: [
              {
                cve: "CVE-2022-2601",
                details_link: "https://nvd.nist.gov/vuln/detail/CVE-2022-2601",
                created_at: "2024-08-14T01:01:19Z",
                cvss_score: 8.6,
                epss_probability: 0.00075,
                cisa_known_exploit: false,
                cve_published: "2022-12-14T21:15:00Z",
                cve_description: "Very bad",
                resolved_in_version: "10.0.22621.4037",
              },
            ],
          },
        ]}
        selectedPlatform="windows"
        isLoading={false}
      />
    );

    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("Hosts")).toBeInTheDocument();
    expect(screen.getByText("1.2.3")).toBeInTheDocument();
    expect(screen.getByText("234")).toBeInTheDocument();
    expect(screen.getByText("4.5.6")).toBeInTheDocument();
    expect(screen.getByText("567")).toBeInTheDocument();
  });
});
