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

  it("renders the empty state when no software is detected", () => {
    renderWithContext({
      data: createMockGetHostSoftwareResponse({
        count: 0,
        software: [],
      }),
    });
    expect(screen.getByText(/no software detected/i)).toBeInTheDocument();
  });

  it("renders Android software inventory", () => {
    renderWithContext({ platform: "android" });
    expect(
      screen.queryByText(/Software installed on this host/i)
    ).toBeInTheDocument();
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

  // This includes empty state for BYOD iphone/ipads
  it("renders generic empty state when no filters are applied and platform is iPad/iPhone", () => {
    renderWithContext({
      platform: "ipados",
      data: createMockGetHostSoftwareResponse({
        count: 0,
        software: [],
      }),
    });

    expect(screen.getByText(/no software detected/i)).toBeInTheDocument();
  });
});
