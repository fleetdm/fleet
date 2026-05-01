import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import {
  createMockSoftwareTitlesResponse,
  createMockSoftwareVersionsResponse,
} from "__mocks__/softwareMock";
import { noop } from "lodash";

import SoftwareInventoryTable from "./SoftwareInventoryTable";

const mockRouter = createMockRouter();

describe("Software inventory table", () => {
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
      screen.getByText("Expecting to see software? Check back later.")
    ).toBeInTheDocument();
    expect(screen.getByText("0 items")).toBeInTheDocument();
    expect(screen.queryByText("Search")).toBeNull();
    expect(screen.queryByText("Updated")).toBeNull();
    expect(screen.queryByText("Add filters")).toBeNull();
  });

  it("Renders the page-wide empty state hiding vulnerability filtering when search query does not exist but versions toggle is applied", () => {
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
      screen.getByText("Expecting to see software? Check back later.")
    ).toBeInTheDocument();
    expect(screen.queryByText("Search")).toBeNull();
    expect(screen.queryByText("Add filters")).toBeNull();
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
