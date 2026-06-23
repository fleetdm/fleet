import React from "react";
import { screen, fireEvent } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { noop } from "lodash";
import { HostPlatform } from "interfaces/platform";

import createMockUser from "__mocks__/userMock";
import {
  createMockGetHostSoftwareResponse,
  createMockHostSoftware,
} from "__mocks__/hostMock";
import HostSoftwareTable from "./HostSoftwareTable";

const mockRouter = createMockRouter();

// Server-side pagination only renders when a full page of rows is present
// (DEFAULT_PAGE_SIZE = 20) and there are further results.
const fullPageWithNextResults = createMockGetHostSoftwareResponse({
  software: Array.from({ length: 20 }, (_, index) =>
    createMockHostSoftware({ id: index + 1 })
  ),
  meta: { has_next_results: true, has_previous_results: false },
});

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

  it("renders the /Applications filter for macOS hosts with the filter on by default", () => {
    renderWithContext({
      platform: "darwin",
      macosApplicationsFilter: true,
    });

    // The selected option label is shown in the dropdown
    expect(screen.getByText("Applications")).toBeInTheDocument();
  });

  it("shows 'Full inventory' selected when the /Applications filter is off", () => {
    renderWithContext({
      platform: "darwin",
      macosApplicationsFilter: false,
    });

    expect(screen.getByText("Full inventory")).toBeInTheDocument();
  });

  it("does not render the /Applications filter for non-macOS hosts", () => {
    renderWithContext({
      platform: "windows",
    });

    expect(screen.queryByText("Applications")).not.toBeInTheDocument();
    expect(screen.queryByText("Full inventory")).not.toBeInTheDocument();
  });

  it("appends macos_applications to the URL on pagination when the filter is set", () => {
    const router = createMockRouter();
    renderWithContext({
      router,
      platform: "darwin",
      macosApplicationsFilter: true,
      data: fullPageWithNextResults,
    });

    fireEvent.click(screen.getByRole("button", { name: /next/i }));

    expect(router.replace).toHaveBeenCalledWith(
      expect.stringContaining("macos_applications=true")
    );
  });

  it("does not append macos_applications to the URL on pagination when the filter is undefined (My device page)", () => {
    const router = createMockRouter();
    renderWithContext({
      router,
      platform: "darwin",
      // My device page leaves the filter undefined since it has no dropdown.
      macosApplicationsFilter: undefined,
      isMyDevicePage: true,
      data: fullPageWithNextResults,
    });

    fireEvent.click(screen.getByRole("button", { name: /next/i }));

    expect(router.replace).toHaveBeenCalledTimes(1);
    expect(router.replace).not.toHaveBeenCalledWith(
      expect.stringContaining("macos_applications")
    );
  });
});
