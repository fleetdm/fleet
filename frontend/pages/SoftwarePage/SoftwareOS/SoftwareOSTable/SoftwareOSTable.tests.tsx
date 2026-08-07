import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockRouter } from "test/test-utils";

import {
  createMockOSVersion,
  createMockOSVersionsResponse,
  createMockSoftwareVulnerability,
} from "__mocks__/softwareMock";

import SoftwareOSTable from "./SoftwareOSTable";

const mockRouter = createMockRouter();

describe("Software operating systems table", () => {
  it("Renders the page-wide disabled state when software inventory is disabled", async () => {
    render(
      <SoftwareOSTable
        router={mockRouter}
        isSoftwareEnabled={false} // Set to false
        data={createMockOSVersionsResponse({
          count: 0,
          os_versions: [],
        })}
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        currentPage={0}
        teamId={1}
        isLoading={false}
      />
    );

    expect(screen.getByText("Software inventory disabled")).toBeInTheDocument();
  });

  it("Renders the page-wide empty state when no software is present", () => {
    render(
      <SoftwareOSTable
        router={mockRouter}
        isSoftwareEnabled
        data={createMockOSVersionsResponse({
          count: 0,
          os_versions: [],
        })}
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        currentPage={0}
        teamId={1}
        isLoading={false}
      />
    );

    expect(
      screen.getByText("No operating systems detected")
    ).toBeInTheDocument();
    expect(screen.getByText("0 items")).toBeInTheDocument();
    expect(screen.getByText("All platforms")).toBeInTheDocument();
    expect(screen.queryByText("Search")).toBeNull();
    expect(screen.queryByText("Updated")).toBeNull();
  });

  it("Renders Android rows with their vulnerabilities, not a 'Not supported' state", () => {
    render(
      <SoftwareOSTable
        router={mockRouter}
        isSoftwareEnabled
        data={createMockOSVersionsResponse({
          count: 1,
          os_versions: [
            createMockOSVersion({
              os_version_id: 3,
              name: "Android 16 (2026-05-01)",
              name_only: "Android",
              version: "16 (2026-05-01)",
              platform: "android",
              hosts_count: 189,
              vulnerabilities: [
                createMockSoftwareVulnerability({ cve: "CVE-2026-0073" }),
              ],
            }),
          ],
        })}
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        currentPage={0}
        teamId={1}
        isLoading={false}
      />
    );

    expect(screen.getByText("16 (2026-05-01)")).toBeInTheDocument();
    expect(screen.getByText("CVE-2026-0073")).toBeInTheDocument();
    expect(screen.queryByText("Not supported")).toBeNull();
  });
});
