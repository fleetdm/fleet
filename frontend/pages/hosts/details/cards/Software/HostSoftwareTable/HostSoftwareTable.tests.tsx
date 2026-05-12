import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { noop } from "lodash";
import { HostPlatform } from "interfaces/platform";

import createMockUser from "__mocks__/userMock";
import { createMockGetHostSoftwareResponse } from "__mocks__/hostMock";
import HostSoftwareTable from "./HostSoftwareTable";

const mockRouter = createMockRouter();

describe("HostSoftwareTable", () => {
  const baseProps = {
    tableConfig: [],
    data: createMockGetHostSoftwareResponse(),
    platform: "windows" as HostPlatform,
    isLoading: false,
    router: mockRouter,
    sortHeader: "name",
    sortDirection: "asc" as "asc" | "desc",
    searchQuery: "",
    page: 0,
    pagePath: "/hosts/1/software",
    vulnFilters: {},
    onAddFiltersClick: noop,
    onShowInventoryVersions: noop,
  };

  const renderWithContext = (props = {}) =>
    createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    })(<HostSoftwareTable {...baseProps} {...props} />);

  it("renders truly empty state with disabled controls", () => {
    renderWithContext({
      data: createMockGetHostSoftwareResponse({
        count: 0,
        software: [],
      }),
    });

    // Empty state copy
    expect(screen.getByText("No software found")).toBeInTheDocument();
    expect(
      screen.getByText(/Expecting to see software\? Check back later/i)
    ).toBeInTheDocument();

    // Shows 0 items count
    expect(screen.getByText("0 items")).toBeInTheDocument();

    // Search is disabled
    const searchInput = screen.getByPlaceholderText(/search by name/i);
    expect(searchInput).toBeDisabled();

    // Filter button is disabled
    const filterBtn = screen.getByRole("button", { name: /filter/i });
    expect(filterBtn).toBeDisabled();
  });

  it("renders filtered empty state with enabled controls", () => {
    renderWithContext({
      data: createMockGetHostSoftwareResponse({
        count: 0,
        software: [],
      }),
      searchQuery: "nonexistent",
    });

    // Falls through to the standard empty software table
    expect(
      screen.getByText(/no items match the current search criteria/i)
    ).toBeInTheDocument();

    // Search is NOT disabled
    const searchInput = screen.getByPlaceholderText(/search by name/i);
    expect(searchInput).not.toBeDisabled();
  });

  it("renders custom filter button when filters are applied", () => {
    renderWithContext({
      vulnFilters: { vulnerable: true },
    });
    expect(screen.getByRole("button", { name: /filter/i })).toBeInTheDocument();
  });

  it("renders VulnsNotSupported when vulns filter applied and platform is iPad/iPhone", () => {
    renderWithContext({
      platform: "ipados",
      vulnFilters: { vulnerable: true },
      data: createMockGetHostSoftwareResponse({
        count: 0,
        software: [],
      }),
    });
    expect(
      screen.getByText(/vulnerabilities are not supported/i)
    ).toBeInTheDocument();
  });

  it("renders truly empty state for iPad/iPhone", () => {
    renderWithContext({
      platform: "ipados",
      data: createMockGetHostSoftwareResponse({
        count: 0,
        software: [],
      }),
    });

    expect(screen.getByText("No software found")).toBeInTheDocument();
  });
});
