import React from "react";
import { screen, waitFor } from "@testing-library/react";
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
  it("Renders the page-wide empty state when no software vulnerabilities are present", () => {
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
        emptyStateReason="no-vulns-detected"
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

    expect(screen.getByText("No vulnerabilities detected")).toBeInTheDocument();
    expect(screen.getByText("0 items")).toBeInTheDocument();
    expect(
      screen.getByText("Expecting to see vulnerabilities? Check back later.")
    ).toBeInTheDocument();
    expect(screen.queryByText("Vulnerability")).toBeNull();
  });

  it("Renders the empty search state when search query does not exist but exploited vulnerabilities dropdown is applied", () => {
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
        showExploitedVulnerabilitiesOnly
        currentPage={0}
        isLoading={false}
        resetPageIndex={false}
      />
    );

    expect(
      screen.getByText("No items match the current search criteria")
    ).toBeInTheDocument();
    expect(
      screen.getByText("Expecting to see vulnerabilities? Check back later.")
    ).toBeInTheDocument();
    expect(screen.queryByText("Vulnerability")).toBeNull();
  });

  it("Renders the invalid CVE empty search state when search query wrapped in quotes is invalid with no results", () => {
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
        emptyStateReason="invalid-cve"
        query='"abcdefg"'
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
      screen.getByText("That vulnerability (CVE) is not valid")
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Try updating your search to use CVE format:/i)
    ).toBeInTheDocument();
    expect(screen.queryByText("Vulnerability")).toBeNull();
  });

  it("Renders the valid known CVE empty search state when search query wrapped in quotes is valid known CVE with no results", () => {
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
        emptyStateReason="known-vuln"
        query='"cve-2002-1000"'
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
      screen.getByText(
        "This is a known vulnerability (CVE), but it wasn't detected on any hosts"
      )
    ).toBeInTheDocument();
    expect(
      screen.getByText("Expecting to see vulnerabilities? Check back later.")
    ).toBeInTheDocument();
    expect(screen.queryByText("Vulnerability")).toBeNull();
  });

  it("Renders the valid unknown CVE empty search state when search query wrapped in quotes is not a valid known CVE with no results", () => {
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
        emptyStateReason="unknown-cve"
        query="cve-2002-12345"
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        showExploitedVulnerabilitiesOnly={false}
        currentPage={0}
        isLoading={false}
        resetPageIndex={false}
      />
    );

    expect(screen.getByText("This is not a known CVE")).toBeInTheDocument();
    expect(
      screen.getByText(
        "None of Fleet's vulnerability sources are aware of this CVE."
      )
    ).toBeInTheDocument();
    expect(screen.queryByText("Vulnerability")).toBeNull();
  });

  it("Renders premium columns", () => {
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
        query="CVE-2018-16463"
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
    ).toHaveClass("react-select__option--is-disabled");

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
