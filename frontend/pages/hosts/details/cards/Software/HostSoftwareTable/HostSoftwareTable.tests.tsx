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
    pathPrefix: "/hosts/1/software",
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

  it("renders the Android not supported state", () => {
    renderWithContext({ platform: "android" });
    expect(
      screen.getByText(/software is not supported for this host/i)
    ).toBeInTheDocument();
    expect(screen.getByText(/let us know/i)).toBeInTheDocument();
    expect(
      screen.queryByText(/Software installed on this host/i)
    ).not.toBeInTheDocument();
  });

  it("renders custom filter button when filters are applied", () => {
    renderWithContext({
      vulnFilters: { vulnerable: true },
    });
    expect(screen.getByRole("button", { name: /filter/i })).toBeInTheDocument();
  });
});
