import { render, screen } from "@testing-library/react";
import React from "react";

import {
  createMockSoftwareVersion,
  createMockSoftwareVersionsResponse,
} from "__mocks__/softwareMock";
import { createMockRouter } from "test/test-utils";

import Software from "./Software";

describe("Dashboard software card", () => {
  const vulnSwInfo = {
    name: "ms-toolsai.jupyter",
    version: "2.3.4",
    hostsCount: 432,
  };
  const noVulnSwInfo = {
    name: "common.extension_2",
    version: "5.6.7",
    hostsCount: 543,
  };

  it("renders all software normally when present", () => {
    const allSwResponse = createMockSoftwareVersionsResponse({
      counts_updated_at: "2024-08-21T21:01:31Z",
      count: 2,
      software: [
        createMockSoftwareVersion({
          id: 784,
          name: vulnSwInfo.name,
          version: vulnSwInfo.version,
          source: "vscode_extensions",
          extension_for: "vscode",
          vendor: "Microsoft",
          hosts_count: vulnSwInfo.hostsCount,
        }),
        createMockSoftwareVersion({
          id: 758,
          name: noVulnSwInfo.name,
          version: noVulnSwInfo.version,
          source: "vscode_extensions",
          extension_for: "cursor",
          vulnerabilities: null,
          hosts_count: noVulnSwInfo.hostsCount,
        }),
      ],
    });
    render(
      <Software
        errorSoftware={null}
        isSoftwareFetching={false}
        isSoftwareEnabled
        navTabIndex={0}
        onTabChange={jest.fn()}
        onQueryChange={jest.fn()}
        software={allSwResponse}
        teamId={-1}
        router={createMockRouter()}
        softwarePageIndex={0}
      />
    );

    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("Hosts")).toBeInTheDocument();

    Object.keys(noVulnSwInfo).forEach((key) => {
      const value = noVulnSwInfo[key as keyof typeof noVulnSwInfo];
      const elements = screen.getAllByText((content, element) => {
        return element?.textContent === String(value);
      });
      expect(elements.length).toBeGreaterThan(0);
    });
  });
  it("renders vulnerable software normally when present", () => {
    const vulnSwResponse = createMockSoftwareVersionsResponse({
      counts_updated_at: "2024-08-21T21:01:31Z",
      count: 1,
      software: [
        createMockSoftwareVersion({
          id: 784,
          name: vulnSwInfo.name,
          version: vulnSwInfo.version,
          source: "vscode_extensions",
          extension_for: "vscode",
          vendor: "Microsoft",
          hosts_count: vulnSwInfo.hostsCount,
        }),
      ],
    });
    render(
      <Software
        errorSoftware={null}
        isSoftwareFetching={false}
        isSoftwareEnabled
        navTabIndex={1}
        onTabChange={jest.fn()}
        onQueryChange={jest.fn()}
        software={vulnSwResponse}
        teamId={-1}
        router={createMockRouter()}
        softwarePageIndex={0}
      />
    );

    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("Hosts")).toBeInTheDocument();

    Object.keys(vulnSwInfo).forEach((key) => {
      const value = vulnSwInfo[key as keyof typeof vulnSwInfo];
      const elements = screen.getAllByText((content, element) => {
        return element?.textContent === String(value);
      });
      expect(elements.length).toBeGreaterThan(0);
    });
  });
});
