import React from "react";

import { IOperatingSystemVersion } from "interfaces/operating_system";

import TableContainer from "components/TableContainer";

import { generateTableHeaders } from "./OSVersionTableConfig";
import OSVersionsEmptyState from "../OSVersionsEmptyState";

const baseClass = "os-version-table";

interface IOSVersionTableProps {
  osVersionData: IOperatingSystemVersion[];
  currentTeamId: number;
  isLoading: boolean;
}

const DEFAULT_SORT_HEADER = "hosts_count";
const DEFAULT_SORT_DIRECTION = "desc";

const OSVersionTable = ({
  osVersionData,
  currentTeamId,
  isLoading,
}: IOSVersionTableProps) => {
  const columns = generateTableHeaders(currentTeamId);

  const osVersionData2 = [
    {
      os_version_id: 179,
      hosts_count: 38,
      name: "iOS Name 10.0",
      name_only: "iOS",
      version: "14.5",
      platform: "ios",
      vulnerabilities: [],
    },
    {
      os_version_id: 180,
      hosts_count: 38,
      name: "iPadOS 17.5.1",
      name_only: "iPadOS",
      version: "17.5.1",
      platform: "ipados",
      vulnerabilities: [],
    },
    {
      os_version_id: 79,
      hosts_count: 38,
      name: "macOS 14.5",
      name_only: "macOS",
      version: "14.5",
      platform: "darwin",
      generated_cpes: [
        "cpe:2.3:o:apple:macos:14.5:*:*:*:*:*:*:*",
        "cpe:2.3:o:apple:mac_os_x:14.5:*:*:*:*:*:*:*",
      ],
      vulnerabilities: [],
    },
    {
      os_version_id: 51,
      hosts_count: 13,
      name: "macOS 14.4.1",
      name_only: "macOS",
      version: "14.4.1",
      platform: "darwin",
      generated_cpes: [
        "cpe:2.3:o:apple:macos:14.4.1:*:*:*:*:*:*:*",
        "cpe:2.3:o:apple:mac_os_x:14.4.1:*:*:*:*:*:*:*",
      ],
      vulnerabilities: [],
    },
    {
      os_version_id: 45,
      hosts_count: 3,
      name: "macOS 14.4",
      name_only: "macOS",
      version: "14.4",
      platform: "darwin",
      generated_cpes: [
        "cpe:2.3:o:apple:macos:14.4:*:*:*:*:*:*:*",
        "cpe:2.3:o:apple:mac_os_x:14.4:*:*:*:*:*:*:*",
      ],
      vulnerabilities: [],
    },
    {
      os_version_id: 47,
      hosts_count: 3,
      name: "Microsoft Windows 11 Pro 23H2 10.0.22621.3296",
      name_only: "Microsoft Windows 11 Pro 23H2",
      version: "10.0.22621.3296",
      platform: "windows",
      vulnerabilities: [],
    },
    {
      os_version_id: 87,
      hosts_count: 2,
      name: "Microsoft Windows 11 Pro 23H2 10.0.22631.3737",
      name_only: "Microsoft Windows 11 Pro 23H2",
      version: "10.0.22631.3737",
      platform: "windows",
      vulnerabilities: [],
    },
    {
      os_version_id: 82,
      hosts_count: 2,
      name: "ChromeOS 124.0.6367.225",
      name_only: "ChromeOS",
      version: "124.0.6367.225",
      platform: "chrome",
      vulnerabilities: [],
    },
    {
      os_version_id: 86,
      hosts_count: 2,
      name: "macOS 15.0",
      name_only: "macOS",
      version: "15.0",
      platform: "darwin",
      generated_cpes: [
        "cpe:2.3:o:apple:macos:15.0:*:*:*:*:*:*:*",
        "cpe:2.3:o:apple:mac_os_x:15.0:*:*:*:*:*:*:*",
      ],
      vulnerabilities: [
        {
          cve: "CVE-2024-23252",
          details_link: "https://nvd.nist.gov/vuln/detail/CVE-2024-23252",
          epss_probability: 0.00043,
          cisa_known_exploit: false,
          cve_published: "2024-03-08T02:15:00Z",
          cve_description:
            "Rejected reason: This CVE ID has been rejected or withdrawn by its CVE Numbering Authority.",
          resolved_in_version: "17.4",
        },
      ],
    },
    {
      os_version_id: 39,
      hosts_count: 2,
      name: "Microsoft Windows 11 Pro 22H2 10.0.22621.3155",
      name_only: "Microsoft Windows 11 Pro 22H2",
      version: "10.0.22621.3155",
      platform: "windows",
      vulnerabilities: [],
    },
    {
      os_version_id: 90,
      hosts_count: 1,
      name: "Microsoft Windows 11 Pro 23H2 10.0.22631.3810",
      name_only: "Microsoft Windows 11 Pro 23H2",
      version: "10.0.22631.3810",
      platform: "windows",
      vulnerabilities: [],
    },
    {
      os_version_id: 64,
      hosts_count: 1,
      name: "macOS 13.5",
      name_only: "macOS",
      version: "13.5",
      platform: "darwin",
      generated_cpes: [
        "cpe:2.3:o:apple:macos:13.5:*:*:*:*:*:*:*",
        "cpe:2.3:o:apple:mac_os_x:13.5:*:*:*:*:*:*:*",
      ],
      vulnerabilities: [],
    },
    {
      os_version_id: 73,
      hosts_count: 1,
      name: "Microsoft Windows 11 Pro 23H2 10.0.22631.3447",
      name_only: "Microsoft Windows 11 Pro 23H2",
      version: "10.0.22631.3447",
      platform: "windows",
      vulnerabilities: [],
    },
    {
      os_version_id: 76,
      hosts_count: 1,
      name: "Microsoft Windows 11 Pro 23H2 10.0.22631.3593",
      name_only: "Microsoft Windows 11 Pro 23H2",
      version: "10.0.22631.3593",
      platform: "windows",
      vulnerabilities: [],
    },
    {
      os_version_id: 88,
      hosts_count: 1,
      name: "Microsoft Windows 11 Pro 23H2 10.0.22631.3155",
      name_only: "Microsoft Windows 11 Pro 23H2",
      version: "10.0.22631.3155",
      platform: "windows",
      vulnerabilities: [],
    },
    {
      os_version_id: 72,
      hosts_count: 1,
      name: "Microsoft Windows Server 2022 Datacenter 21H2 10.0.20348.2113",
      name_only: "Microsoft Windows Server 2022 Datacenter 21H2",
      version: "10.0.20348.2113",
      platform: "windows",
      vulnerabilities: [],
    },
    {
      os_version_id: 34,
      hosts_count: 1,
      name: "Microsoft Windows 11 Pro 23H2 10.0.22621.3155",
      name_only: "Microsoft Windows 11 Pro 23H2",
      version: "10.0.22621.3155",
      platform: "windows",
      vulnerabilities: [],
    },
    {
      os_version_id: 83,
      hosts_count: 1,
      name: "Microsoft Windows 11 Pro 23H2 10.0.22631.3296",
      name_only: "Microsoft Windows 11 Pro 23H2",
      version: "10.0.22631.3296",
      platform: "windows",
      vulnerabilities: [],
    },
    {
      os_version_id: 7,
      hosts_count: 1,
      name: "macOS 13.5.1",
      name_only: "macOS",
      version: "13.5.1",
      platform: "darwin",
      generated_cpes: [
        "cpe:2.3:o:apple:macos:13.5.1:*:*:*:*:*:*:*",
        "cpe:2.3:o:apple:mac_os_x:13.5.1:*:*:*:*:*:*:*",
      ],
      vulnerabilities: [],
    },
    {
      os_version_id: 10,
      hosts_count: 1,
      name: "macOS 14.1",
      name_only: "macOS",
      version: "14.1",
      platform: "darwin",
      generated_cpes: [
        "cpe:2.3:o:apple:macos:14.1:*:*:*:*:*:*:*",
        "cpe:2.3:o:apple:mac_os_x:14.1:*:*:*:*:*:*:*",
      ],
      vulnerabilities: [],
    },
    {
      os_version_id: 65,
      hosts_count: 1,
      name: "macOS 14.2",
      name_only: "macOS",
      version: "14.2",
      platform: "darwin",
      generated_cpes: [
        "cpe:2.3:o:apple:macos:14.2:*:*:*:*:*:*:*",
        "cpe:2.3:o:apple:mac_os_x:14.2:*:*:*:*:*:*:*",
      ],
      vulnerabilities: [],
    },
    {
      os_version_id: 13,
      hosts_count: 1,
      name: "macOS 14.2.1",
      name_only: "macOS",
      version: "14.2.1",
      platform: "darwin",
      generated_cpes: [
        "cpe:2.3:o:apple:macos:14.2.1:*:*:*:*:*:*:*",
        "cpe:2.3:o:apple:mac_os_x:14.2.1:*:*:*:*:*:*:*",
      ],
      vulnerabilities: [],
    },
    {
      os_version_id: 14,
      hosts_count: 1,
      name: "macOS 14.3",
      name_only: "macOS",
      version: "14.3",
      platform: "darwin",
      generated_cpes: [
        "cpe:2.3:o:apple:macos:14.3:*:*:*:*:*:*:*",
        "cpe:2.3:o:apple:mac_os_x:14.3:*:*:*:*:*:*:*",
      ],
      vulnerabilities: [],
    },
    {
      os_version_id: 89,
      hosts_count: 1,
      name: "Microsoft Windows 11 Home 23H2 10.0.22631.3737",
      name_only: "Microsoft Windows 11 Home 23H2",
      version: "10.0.22631.3737",
      platform: "windows",
      vulnerabilities: [],
    },
    {
      os_version_id: 77,
      hosts_count: 1,
      name: "Microsoft Windows 11 Enterprise Evaluation 22H2 10.0.22621.3447",
      name_only: "Microsoft Windows 11 Enterprise Evaluation 22H2",
      version: "10.0.22621.3447",
      platform: "windows",
      vulnerabilities: [],
    },
    {
      os_version_id: 74,
      hosts_count: 1,
      name: "Microsoft Windows 11 Enterprise 23H2 10.0.22631.3447",
      name_only: "Microsoft Windows 11 Enterprise 23H2",
      version: "10.0.22631.3447",
      platform: "windows",
      vulnerabilities: [],
    },
  ];

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={columns}
        data={osVersionData2}
        isLoading={isLoading}
        resultsTitle=""
        emptyComponent={OSVersionsEmptyState}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        defaultSortHeader={DEFAULT_SORT_HEADER}
        defaultSortDirection={DEFAULT_SORT_DIRECTION}
        disableTableHeader
        disableCount
        pageSize={8}
        isClientSidePagination
      />
    </div>
  );
};

export default OSVersionTable;
