import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { HostPlatform } from "interfaces/platform";

import createMockUser from "__mocks__/userMock";
import { createMockGetHostSoftwareResponse } from "__mocks__/hostMock";
import HostSoftwareLibraryTable from "./HostSoftwareLibraryTable";

const mockRouter = createMockRouter();

describe("HostSoftwareLibraryTable", () => {
  const baseProps = {
    tableConfig: [],
    data: createMockGetHostSoftwareResponse(),
    enhancedData: [],
    platform: "darwin" as HostPlatform,
    isLoading: false,
    router: mockRouter,
    sortHeader: "name",
    sortDirection: "asc" as "asc" | "desc",
    searchQuery: "",
    page: 0,
    pagePath: "/hosts/1/software",
    selfService: false,
  };

  const renderWithContext = (props = {}) =>
    createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    })(<HostSoftwareLibraryTable {...baseProps} {...props} />);

  it("renders truly empty state with disabled controls and Add software CTA", () => {
    const onAddSoftware = jest.fn();
    renderWithContext({
      data: createMockGetHostSoftwareResponse({ count: 0, software: [] }),
      enhancedData: [],
      canAddSoftware: true,
      onAddSoftware,
    });

    // Empty state copy
    expect(screen.getByText("No software found")).toBeInTheDocument();
    expect(screen.getByText("Add software to install.")).toBeInTheDocument();

    // Add software CTA button in empty state
    expect(
      screen.getByRole("button", { name: /add software/i })
    ).toBeInTheDocument();

    // Shows 0 items count
    expect(screen.getByText("0 items")).toBeInTheDocument();

    // Search is disabled
    const searchInput = screen.getByPlaceholderText(/search by name/i);
    expect(searchInput).toBeDisabled();
  });

  it("renders truly empty state without Add software CTA when user lacks permission", () => {
    renderWithContext({
      data: createMockGetHostSoftwareResponse({ count: 0, software: [] }),
      enhancedData: [],
      canAddSoftware: false,
    });

    expect(screen.getByText("No software found")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /add software/i })
    ).not.toBeInTheDocument();
  });

  it("renders filtered empty state with enabled controls", () => {
    renderWithContext({
      data: createMockGetHostSoftwareResponse({ count: 0, software: [] }),
      enhancedData: [],
      searchQuery: "nonexistent",
    });

    // Search is NOT disabled
    const searchInput = screen.getByPlaceholderText(/search by name/i);
    expect(searchInput).not.toBeDisabled();
  });

  // Android software installs are not currently supported from the library table.
  // The parent (HostDetailsPage) also guards this, but the table has a defensive fallback.
  it("renders Android unsupported state", () => {
    renderWithContext({
      platform: "android",
    });

    expect(
      screen.getByText(/installers are not supported for this host/i)
    ).toBeInTheDocument();
  });
});
