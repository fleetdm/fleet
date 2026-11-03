import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import {
  createMockGetHostSoftwareResponse,
  createMockHostSoftware,
} from "__mocks__/hostMock";
import createMockUser from "__mocks__/userMock";
import { HostPlatform } from "interfaces/platform";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

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
    filters: {},
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

  it("renders VulnsNotSupported when vulns filter applied and platform is iPad/iPhone", () => {
    renderWithContext({
      platform: "ipados",
      filters: { vulnerable: true },
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

  it("renders Show helpers off when only top-level applications are shown", () => {
    renderWithContext({
      platform: "darwin",
      macosApplicationsFilter: true,
    });

    expect(
      screen.getByRole("switch", { name: "Show helpers" })
    ).toHaveAttribute("aria-checked", "false");
  });

  it("renders Show helpers on when the /Applications filter is off", () => {
    renderWithContext({
      platform: "darwin",
      macosApplicationsFilter: false,
    });

    expect(
      screen.getByRole("switch", { name: "Show helpers" })
    ).toHaveAttribute("aria-checked", "true");
  });

  it("does not render Show helpers when the /Applications filter doesn't apply", () => {
    renderWithContext({
      platform: "darwin",
      macosApplicationsFilter: undefined,
    });

    expect(
      screen.queryByRole("switch", { name: "Show helpers" })
    ).not.toBeInTheDocument();
  });

  it("writes macos_applications=false when Show helpers is turned on", async () => {
    const router = createMockRouter({ replace: jest.fn() });
    const { user } = renderWithContext({
      router,
      platform: "darwin",
      macosApplicationsFilter: true,
      filters: { types: ["macos_app"] },
    });

    await user.click(screen.getByRole("switch", { name: "Show helpers" }));

    expect(router.replace).toHaveBeenCalledWith(
      expect.stringContaining("macos_applications=false")
    );
    expect(router.replace).toHaveBeenCalledWith(
      expect.stringContaining("types=macos_app")
    );
  });

  it("places the controls in order: Show helpers, filters button, search", () => {
    renderWithContext({
      platform: "darwin",
      macosApplicationsFilter: true,
      isMyDevicePage: true,
    });

    const toggle = screen.getByRole("switch", { name: "Show helpers" });
    const button = screen.getByRole("button", { name: "Add filters" });
    const search = screen.getByPlaceholderText(/search by name/i);
    const nodes = Array.from(document.body.querySelectorAll("*"));
    expect(nodes.indexOf(toggle)).toBeLessThan(nodes.indexOf(button));
    expect(nodes.indexOf(button)).toBeLessThan(nodes.indexOf(search));
  });

  it("labels the filters button Filtered when a type is selected", () => {
    renderWithContext({ filters: { types: ["windows_app"] } });

    expect(
      screen.getByRole("button", { name: "Filtered" })
    ).toBeInTheDocument();
  });

  it("shows the filtered empty state when a type matches nothing", () => {
    renderWithContext({
      filters: { types: ["windows_app"] },
      data: createMockGetHostSoftwareResponse({ count: 0, software: [] }),
    });

    expect(
      screen.getByText(/no items match the current search criteria/i)
    ).toBeInTheDocument();
  });

  it("shows the no-software empty state under the macOS default with controls enabled", () => {
    renderWithContext({
      platform: "darwin",
      macosApplicationsFilter: true,
      filters: { types: ["macos_app"] },
      data: createMockGetHostSoftwareResponse({ count: 0, software: [] }),
    });

    expect(screen.getByText("No software found")).toBeInTheDocument();
    // The modal is the only way to widen the selection, so it stays reachable.
    expect(screen.getByRole("button", { name: "Filtered" })).toBeEnabled();
    expect(
      screen.getByRole("switch", { name: "Show helpers" })
    ).toBeInTheDocument();
  });

  it("shows the filtered empty state when types beyond the macOS default match nothing", () => {
    renderWithContext({
      platform: "darwin",
      macosApplicationsFilter: true,
      filters: { types: ["macos_app", "chrome_extension"] },
      data: createMockGetHostSoftwareResponse({ count: 0, software: [] }),
    });

    expect(
      screen.getByText(/no items match the current search criteria/i)
    ).toBeInTheDocument();
  });

  it.each([
    { name: "vulnerable", filter: { vulnerable: true } },
    { name: "AI tools", filter: { aiTool: true } },
  ])(
    "shows the filtered empty state when the $name filter joins the macOS default",
    ({ filter }) => {
      renderWithContext({
        platform: "darwin",
        macosApplicationsFilter: true,
        filters: { types: ["macos_app"], ...filter },
        data: createMockGetHostSoftwareResponse({ count: 0, software: [] }),
      });

      expect(
        screen.getByText(/no items match the current search criteria/i)
      ).toBeInTheDocument();
    }
  );

  it("keeps macos_applications and the filters in the URL on pagination", async () => {
    const router = createMockRouter();
    const { user } = renderWithContext({
      router,
      platform: "darwin",
      macosApplicationsFilter: true,
      filters: { types: ["macos_app"], aiTool: true },
      data: fullPageWithNextResults,
    });

    await user.click(screen.getByRole("button", { name: /next/i }));

    const url = new URL(
      (router.replace as jest.Mock).mock.calls[0][0],
      "http://fleet"
    );
    expect(url.searchParams.get("macos_applications")).toBe("true");
    expect(url.searchParams.get("ai_tool")).toBe("true");
    expect(url.searchParams.get("page")).toBe("1");
  });

  it("appends macos_applications to the URL on pagination on the My device page", async () => {
    const router = createMockRouter();
    const { user } = renderWithContext({
      router,
      platform: "darwin",
      macosApplicationsFilter: true,
      isMyDevicePage: true,
      data: fullPageWithNextResults,
    });

    await user.click(screen.getByRole("button", { name: /next/i }));

    expect(router.replace).toHaveBeenCalledWith(
      expect.stringContaining("macos_applications=true")
    );
  });

  it("does not append macos_applications to the URL on pagination when the filter is undefined (non-macOS host)", async () => {
    const router = createMockRouter();
    const { user } = renderWithContext({
      router,
      platform: "windows",
      // Non-macOS platforms leave the filter undefined since it doesn't apply.
      macosApplicationsFilter: undefined,
      data: fullPageWithNextResults,
    });

    await user.click(screen.getByRole("button", { name: /next/i }));

    expect(router.replace).toHaveBeenCalledTimes(1);
    expect(router.replace).not.toHaveBeenCalledWith(
      expect.stringContaining("macos_applications")
    );
  });

  it("keeps a cleared macOS selection as types=none on pagination", async () => {
    const router = createMockRouter();
    const { user } = renderWithContext({
      router,
      platform: "darwin",
      filters: { types: [] },
      data: fullPageWithNextResults,
    });

    await user.click(screen.getByRole("button", { name: /next/i }));

    expect(router.replace).toHaveBeenCalledWith(
      expect.stringContaining("types=none")
    );
  });

  it("keeps the selected types in the URL on pagination", async () => {
    const router = createMockRouter();
    const { user } = renderWithContext({
      router,
      filters: { types: ["windows_app"] },
      data: fullPageWithNextResults,
    });

    await user.click(screen.getByRole("button", { name: /next/i }));

    expect(router.replace).toHaveBeenCalledWith(
      expect.stringContaining("types=windows_app")
    );
  });
});
