import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import {
  createMockSoftwarePackage,
  createMockSoftwareTitle,
  createMockSoftwareTitlesResponse,
  createMockSoftwareVersionsResponse,
} from "__mocks__/softwareMock";
import createMockUser from "__mocks__/userMock";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import SoftwareInventoryTable from "./SoftwareInventoryTable";

const mockRouter = createMockRouter();

describe("Software inventory table", () => {
  it("distinguishes script packages with the same title name", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const softwareTitles = [
      createMockSoftwareTitle({
        id: 1,
        name: "hello",
        source: "py_packages",
        software_package: createMockSoftwarePackage({ name: "hello.py" }),
      }),
      createMockSoftwareTitle({
        id: 2,
        name: "hello",
        source: "sh_packages",
        software_package: createMockSoftwarePackage({ name: "hello.sh" }),
      }),
    ];

    render(
      <SoftwareInventoryTable
        router={mockRouter}
        isSoftwareEnabled
        showVersions={false}
        data={createMockSoftwareTitlesResponse({
          count: 2,
          software_titles: softwareTitles,
        })}
        installableSoftwareExists
        query=""
        perPage={20}
        orderDirection="asc"
        orderKey="name"
        vulnFilters={{
          vulnerable: false,
          exploit: false,
          minCvssScore: undefined,
          maxCvssScore: undefined,
        }}
        currentPage={0}
        teamId={1}
        isLoading={false}
        onAddFiltersClick={noop}
      />
    );

    expect(
      screen.getByRole("row", { name: /hello \(hello\.py\)/ })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("row", { name: /hello \(hello\.sh\)/ })
    ).toBeInTheDocument();
  });

  it("Renders the page-wide disabled state when software inventory is disabled", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareInventoryTable
        router={mockRouter}
        isSoftwareEnabled={false} // Set to false
        showVersions={false}
        data={createMockSoftwareTitlesResponse({
          counts_updated_at: null,
          software_titles: [],
        })}
        installableSoftwareExists={false}
        query=""
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        vulnFilters={{
          vulnerable: false,
          exploit: false,
          minCvssScore: undefined,
          maxCvssScore: undefined,
        }}
        currentPage={0}
        teamId={1}
        isLoading={false}
        onAddFiltersClick={noop}
      />
    );

    expect(screen.getByText("Software inventory disabled")).toBeInTheDocument();
    expect(screen.queryByText("Vulnerability")).toBeNull();
    expect(screen.queryByText("All software")).toBeNull();
    expect(screen.queryByText("Available for install")).toBeNull();
  });

  it("Renders the page-wide empty state when no software are present hiding search bar and vulnerability filtering", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareInventoryTable
        router={mockRouter}
        isSoftwareEnabled
        showVersions={false}
        data={createMockSoftwareTitlesResponse({
          count: 0,
          counts_updated_at: null,
          software_titles: [],
        })}
        installableSoftwareExists={false}
        query=""
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        vulnFilters={{
          vulnerable: false,
          exploit: false,
          minCvssScore: undefined,
          maxCvssScore: undefined,
        }}
        currentPage={0}
        teamId={1}
        isLoading={false}
        onAddFiltersClick={noop}
      />
    );

    expect(screen.getByText("No software detected")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Recently installed software will appear after the next scheduled check-in."
      )
    ).toBeInTheDocument();
    expect(screen.getByText("0 items")).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText("Search by name or vulnerability (CVE)")
    ).toBeDisabled();
    expect(screen.getByRole("button", { name: /add filters/i })).toBeDisabled();
    expect(screen.getByText("Show versions")).toBeInTheDocument();
  });

  it("Keeps controls enabled when versions toggle is applied but no data so users can toggle back", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareInventoryTable
        router={mockRouter}
        isSoftwareEnabled
        showVersions // Versions toggle applied
        data={createMockSoftwareVersionsResponse({
          counts_updated_at: null,
          software: [],
        })}
        installableSoftwareExists={false}
        query=""
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        vulnFilters={{
          vulnerable: false,
          exploit: false,
          minCvssScore: undefined,
          maxCvssScore: undefined,
        }}
        currentPage={0}
        teamId={1}
        isLoading={false}
        onAddFiltersClick={noop}
      />
    );

    expect(screen.getByText("No software detected")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Recently installed software will appear after the next scheduled check-in."
      )
    ).toBeInTheDocument();
    // Controls stay enabled so users can toggle back to the titles view,
    // which may have installers even when the versions view is empty.
    expect(
      screen.getByPlaceholderText("Search by name or vulnerability (CVE)")
    ).toBeEnabled();
    expect(screen.getByRole("button", { name: /add filters/i })).toBeEnabled();
  });

  it("Renders the empty search state and vulnerability filtering when search query does not exist but vulnerability filter is applied", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareInventoryTable
        router={mockRouter}
        isSoftwareEnabled
        showVersions={false}
        data={createMockSoftwareTitlesResponse({
          counts_updated_at: null,
          software_titles: [],
        })}
        installableSoftwareExists={false}
        query=""
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        vulnFilters={{
          vulnerable: true,
          exploit: false,
          minCvssScore: undefined,
          maxCvssScore: undefined,
        }}
        currentPage={0}
        teamId={1}
        isLoading={false}
        onAddFiltersClick={noop}
      />
    );

    expect(
      screen.getByText("No items match the current search criteria")
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Expecting to see vulnerable software? Check back later."
      )
    ).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText("Search by name or vulnerability (CVE)")
    ).toBeInTheDocument();
    expect(screen.getByText("1 filter")).toBeInTheDocument();
  });
});
