import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import { createMockOSVersionsResponse } from "__mocks__/softwareMock";

import SoftwareOSTable from "./SoftwareOSTable";

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

describe("Software operating systems table", () => {
  it("Renders the page-wide disabled state when software inventory is disabled", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

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
        resetPageIndex={false}
      />
    );

    expect(screen.getByText("Software inventory disabled")).toBeInTheDocument();
  });

  it("Renders the page-wide empty state when no software is present", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

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
        resetPageIndex={false}
      />
    );

    expect(
      screen.getByText("No operating systems detected")
    ).toBeInTheDocument();
    expect(screen.getByText("0 items")).toBeInTheDocument();
    expect(screen.queryByText("Search")).toBeNull();
    expect(screen.queryByText("Updated")).toBeNull();
  });
});
