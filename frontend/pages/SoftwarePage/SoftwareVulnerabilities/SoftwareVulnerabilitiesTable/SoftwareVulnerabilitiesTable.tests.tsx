import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import { createCustomRenderer } from "test/test-utils";

import { createMockVulnerabilitiesResponse } from "__mocks__/vulnerabilitiesMock";
import createMockUser from "__mocks__/userMock";

import SoftwareVulnerabilitiesTable from "./SoftwareVulnerabilitiesTable";

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

describe("Software Vulnerabilities table", () => {
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
      <SoftwareVulnerabilitiesTable
        router={mockRouter}
        isSoftwareEnabled={false}
        data={createMockVulnerabilitiesResponse({
          count: 0,
          vulnerabilities: null,
          meta: {
            has_next_results: false,
            has_previous_results: false,
          },
        })}
        query=""
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        showExploitedVulnerabilitiesOnly={false}
        currentPage={0}
        isLoading={false}
        resetPageIndex={false}
      />
    );

    expect(screen.getByText("Software inventory disabled")).toBeInTheDocument();
    expect(screen.queryByText("Vulnerability")).toBeNull();
  });

  // TODO: Reinstate collecting software view
  it("Renders the page-wide empty state when no software vulnerabilities are present", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareVulnerabilitiesTable
        router={mockRouter}
        isSoftwareEnabled
        data={createMockVulnerabilitiesResponse({
          count: 0,
          vulnerabilities: null,
          meta: {
            has_next_results: false,
            has_previous_results: false,
          },
        })}
        query=""
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        showExploitedVulnerabilitiesOnly={false}
        currentPage={0}
        isLoading={false}
        resetPageIndex={false}
      />
    );

    expect(screen.getByText("No software detected")).toBeInTheDocument();
    expect(screen.queryByText("Vulnerability")).toBeNull();
  });

  it("Renders the empty search state when search query exists for server side search with no results", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareVulnerabilitiesTable
        router={mockRouter}
        isSoftwareEnabled
        data={createMockVulnerabilitiesResponse({
          count: 0,
          vulnerabilities: null,
          meta: {
            has_next_results: false,
            has_previous_results: false,
          },
        })}
        query="abcdefg"
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        showExploitedVulnerabilitiesOnly={false}
        currentPage={0}
        isLoading={false}
        resetPageIndex={false}
      />
    );

    expect(
      screen.getByText("No items match the current search criteria")
    ).toBeInTheDocument();
    expect(screen.queryByText("Vulnerability")).toBeNull();
  });

  it("Renders premium columns", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareVulnerabilitiesTable
        router={mockRouter}
        isSoftwareEnabled
        data={createMockVulnerabilitiesResponse()}
        query=""
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        showExploitedVulnerabilitiesOnly={false}
        currentPage={0}
        isLoading={false}
        resetPageIndex={false}
      />
    );

    expect(screen.getByText("Vulnerability")).toBeInTheDocument();
    expect(screen.getByText("Severity")).toBeInTheDocument();
    expect(screen.getByText("Probability of exploit")).toBeInTheDocument();
    expect(screen.getByText("Published")).toBeInTheDocument();
    expect(screen.getByText("Detected")).toBeInTheDocument();
    expect(screen.getByText("Hosts")).toBeInTheDocument();
  });

  it("Does not render premium only columns and disables exploited vulnerabilities dropdown", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: false,
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const { user } = render(
      <SoftwareVulnerabilitiesTable
        router={mockRouter}
        isSoftwareEnabled
        data={createMockVulnerabilitiesResponse()}
        query=""
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        showExploitedVulnerabilitiesOnly={false}
        currentPage={0}
        isLoading={false}
        resetPageIndex={false}
      />
    );

    expect(screen.getByText("Vulnerability")).toBeInTheDocument();
    expect(screen.queryByText("Severity")).not.toBeInTheDocument();
    expect(
      screen.queryByText("Probability of exploit")
    ).not.toBeInTheDocument();
    expect(screen.queryByText("Published")).not.toBeInTheDocument();
    expect(screen.getByText("Detected")).toBeInTheDocument();
    expect(screen.getByText("Hosts")).toBeInTheDocument();

    await user.click(screen.getByText("All vulnerabilities"));

    expect(
      screen.getByText("Exploited vulnerabilities").parentElement?.parentElement
        ?.parentElement
    ).toHaveClass("is-disabled");

    await waitFor(() => {
      waitFor(() => {
        user.hover(screen.getByText("Exploited vulnerabilities"));
      });

      expect(
        screen.getByText(/Available in Fleet Premium./i)
      ).toBeInTheDocument();
    });
  });
});
