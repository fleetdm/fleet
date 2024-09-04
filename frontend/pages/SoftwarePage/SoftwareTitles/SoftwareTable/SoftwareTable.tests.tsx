import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import {
  createMockSoftwareTitlesResponse,
  createMockSoftwareVersionsResponse,
} from "__mocks__/softwareMock";
import { noop } from "lodash";

import SoftwareTable from "./SoftwareTable";

// TODO: figure out how to mock the router properly.
const mockRouter = {
  push: jest.fn(),
  replace: jest.fn(),
  goBack: jest.fn(),
  goForward: jest.fn(),
  go: jest.fn(),
  setRouteLeaveHook: jest.fn(),
  isActive: jest.fn(),
  createHref: jest.fn(),
  createPath: jest.fn(),
};

describe("Software table", () => {
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
      <SoftwareTable
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
        softwareFilter="allSoftware"
        vulnFilters={{
          vulnerable: false,
          exploit: false,
          minCvssScore: undefined,
          maxCvssScore: undefined,
        }}
        currentPage={0}
        teamId={1}
        isLoading={false}
        resetPageIndex={false}
        onAddFiltersClick={noop}
      />
    );

    expect(screen.getByText("Software inventory disabled")).toBeInTheDocument();
    expect(screen.queryByText("Vulnerability")).toBeNull();
  });

  it("Renders the page-wide empty state when no software are present", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareTable
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
        softwareFilter="allSoftware"
        vulnFilters={{
          vulnerable: false,
          exploit: false,
          minCvssScore: undefined,
          maxCvssScore: undefined,
        }}
        currentPage={0}
        teamId={1}
        isLoading={false}
        resetPageIndex={false}
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
  });

  it("Renders the page-wide empty state when search query does not exist but versions toggle is applied", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareTable
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
        softwareFilter="allSoftware"
        vulnFilters={{
          vulnerable: false,
          exploit: false,
          minCvssScore: undefined,
          maxCvssScore: undefined,
        }}
        currentPage={0}
        teamId={1}
        isLoading={false}
        resetPageIndex={false}
        onAddFiltersClick={noop}
      />
    );

    expect(screen.getByText("No software detected")).toBeInTheDocument();
    expect(
      screen.getByText("Expecting to see software? Check back later.")
    ).toBeInTheDocument();
  });

  it("Renders the empty search state when search query does not exist but dropdown is applied", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareTable
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
        softwareFilter="installableSoftware" // Dropdown applied
        vulnFilters={{
          vulnerable: false,
          exploit: false,
          minCvssScore: undefined,
          maxCvssScore: undefined,
        }}
        currentPage={0}
        teamId={1}
        isLoading={false}
        resetPageIndex={false}
        onAddFiltersClick={noop}
      />
    );

    expect(
      screen.getByText("No items match the current search criteria")
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Expecting to see installable software? Check back later."
      )
    ).toBeInTheDocument();
  });

  it("Renders the empty search state when search query does not exist but vulnerability filter is applied", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareTable
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
        softwareFilter="allSoftware"
        vulnFilters={{
          vulnerable: true,
          exploit: false,
          minCvssScore: undefined,
          maxCvssScore: undefined,
        }}
        currentPage={0}
        teamId={1}
        isLoading={false}
        resetPageIndex={false}
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
  });
});
