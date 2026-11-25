import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import createMockUser from "__mocks__/userMock";
import { ILabelPolicy } from "interfaces/label";
import { IPolicy, PolicyAutomationType } from "interfaces/policy";
import mockServer from "test/mock-server";
import {
  createCustomRenderer,
  baseUrl,
  createMockRouter,
} from "test/test-utils";

import PolicyDetailsPage, { getLabelModalData } from "./PolicyDetailsPage";

// Stub SoftwareIcon to avoid asset resolution when importing the page module.
jest.mock("pages/SoftwarePage/components/icons/SoftwareIcon", () => {
  return () => null;
});

// Avoid depending on react-router's browserHistory inside BackButton.
jest.mock("components/BackButton", () => ({
  __esModule: true,
  default: ({ text }: { text: string }) => (
    <button type="button" data-testid="back-button">
      {text}
    </button>
  ),
}));

// Surface the modal's `query` prop as plain text (the real modal renders it in
// an Ace editor that isn't reliably assertable in jsdom).
jest.mock("components/modals/ShowQueryModal", () => ({
  __esModule: true,
  default: ({ query }: { query?: string }) => (
    <div data-testid="show-query-modal">{query}</div>
  ),
}));

// Activities table fetches on mount; stub it out so the render test stays
// focused on the policy's own fields. Like the real table, it reads the host
// search once on mount; it shows it and lets a test change it.
jest.mock("../components/PolicyAutomationsActivitiesTable", () => ({
  __esModule: true,
  default: function MockActivitiesTable({
    initialSearchQuery,
    onSearchQueryChange,
  }: {
    initialSearchQuery?: string;
    onSearchQueryChange?: (query: string) => void;
  }) {
    const [search] = jest
      .requireActual("react")
      .useState(initialSearchQuery ?? "");
    return (
      <div>
        <div data-testid="activities-table">{search}</div>
        <button
          type="button"
          onClick={() => onSearchQueryChange?.("Other host")}
        >
          Search other host
        </button>
        <button type="button" onClick={() => onSearchQueryChange?.("")}>
          Clear search
        </button>
      </div>
    );
  },
}));

const labels = (...names: string[]): ILabelPolicy[] =>
  names.map((name, i) => ({ id: i + 1, name }));

const createMockPolicy = (overrides?: Partial<IPolicy>): IPolicy => ({
  id: 1,
  name: "Test policy",
  query: "SELECT 1;",
  description: "",
  author_id: 1,
  author_name: "Admin",
  author_email: "admin@example.com",
  resolution: "",
  platform: "darwin",
  fleet_id: 1,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
  critical: false,
  calendar_events_enabled: false,
  conditional_access_enabled: false,
  type: "dynamic",
  ...overrides,
});

describe("getLabelModalData", () => {
  it("returns no label data when the policy has no labels", () => {
    expect(getLabelModalData(createMockPolicy())).toEqual({
      includeLabels: undefined,
      includeScopeLabel: undefined,
      excludeLabels: undefined,
      excludeScopeLabel: undefined,
    });
  });

  it("treats empty label arrays as no labels", () => {
    const result = getLabelModalData(
      createMockPolicy({
        labels_include_any: [],
        labels_exclude_all: [],
      })
    );

    expect(result.includeLabels).toBeUndefined();
    expect(result.excludeLabels).toBeUndefined();
  });

  describe("include labels", () => {
    it("resolves labels_include_any with the 'have any' scope", () => {
      const result = getLabelModalData(
        createMockPolicy({ labels_include_any: labels("A") })
      );

      expect(result.includeLabels).toEqual(labels("A"));
      expect(result.includeScopeLabel).toBe("have any");
      expect(result.excludeLabels).toBeUndefined();
    });

    it("resolves labels_include_all with the 'have all' scope", () => {
      const result = getLabelModalData(
        createMockPolicy({ labels_include_all: labels("A") })
      );

      expect(result.includeLabels).toEqual(labels("A"));
      expect(result.includeScopeLabel).toBe("have all");
    });

    it("prefers labels_include_any over labels_include_all", () => {
      const result = getLabelModalData(
        createMockPolicy({
          labels_include_any: labels("Any"),
          labels_include_all: labels("All"),
        })
      );

      expect(result.includeLabels).toEqual(labels("Any"));
      expect(result.includeScopeLabel).toBe("have any");
    });
  });

  describe("exclude labels", () => {
    it("resolves labels_exclude_any with the 'exclude any' scope", () => {
      const result = getLabelModalData(
        createMockPolicy({ labels_exclude_any: labels("A") })
      );

      expect(result.excludeLabels).toEqual(labels("A"));
      expect(result.excludeScopeLabel).toBe("exclude any");
      expect(result.includeLabels).toBeUndefined();
    });

    it("resolves labels_exclude_all with the 'exclude all' scope", () => {
      const result = getLabelModalData(
        createMockPolicy({ labels_exclude_all: labels("A") })
      );

      expect(result.excludeLabels).toEqual(labels("A"));
      expect(result.excludeScopeLabel).toBe("exclude all");
    });

    it("prefers labels_exclude_any over labels_exclude_all", () => {
      const result = getLabelModalData(
        createMockPolicy({
          labels_exclude_any: labels("Any"),
          labels_exclude_all: labels("All"),
        })
      );

      expect(result.excludeLabels).toEqual(labels("Any"));
      expect(result.excludeScopeLabel).toBe("exclude any");
    });
  });

  describe("include + exclude combinations", () => {
    it("resolves include_any + exclude_any", () => {
      const result = getLabelModalData(
        createMockPolicy({
          labels_include_any: labels("Inc"),
          labels_exclude_any: labels("Exc"),
        })
      );

      expect(result.includeScopeLabel).toBe("have any");
      expect(result.excludeScopeLabel).toBe("exclude any");
    });

    it("resolves include_any + exclude_all", () => {
      const result = getLabelModalData(
        createMockPolicy({
          labels_include_any: labels("Inc"),
          labels_exclude_all: labels("Exc"),
        })
      );

      expect(result.includeScopeLabel).toBe("have any");
      expect(result.excludeScopeLabel).toBe("exclude all");
    });

    it("resolves include_all + exclude_any", () => {
      const result = getLabelModalData(
        createMockPolicy({
          labels_include_all: labels("Inc"),
          labels_exclude_any: labels("Exc"),
        })
      );

      expect(result.includeScopeLabel).toBe("have all");
      expect(result.excludeScopeLabel).toBe("exclude any");
    });

    it("resolves include_all + exclude_all", () => {
      const result = getLabelModalData(
        createMockPolicy({
          labels_include_all: labels("Inc"),
          labels_exclude_all: labels("Exc"),
        })
      );

      expect(result.includeScopeLabel).toBe("have all");
      expect(result.excludeScopeLabel).toBe("exclude all");
    });
  });
});

const POLICY_ID = 8;

const createProps = () => ({
  router: createMockRouter(),
  params: { id: String(POLICY_ID) },
  location: {
    pathname: `/policies/${POLICY_ID}`,
    search: "",
    query: {},
  },
});

const baseAppContext = {
  isGlobalAdmin: true,
  isOnGlobalTeam: true,
  // Free tier short-circuits useTeamIdParam's redirect logic when no fleet_id is
  // set, keeping the test focused on which data source the page renders from.
  isFreeTier: true,
  isPremiumTier: false,
  currentUser: createMockUser({ global_role: "admin" }),
  config: createMockConfig(),
  availableTeams: [],
};

describe("PolicyDetailsPage - renders fresh policy data (regression #43310)", () => {
  it("renders the loaded policy's fields, not stale PolicyContext values", async () => {
    mockServer.use(
      // fleet_id: null keeps the team query disabled, so no second endpoint to mock.
      http.get(baseUrl(`/policies/${POLICY_ID}`), () =>
        HttpResponse.json({
          policy: createMockPolicy({
            id: POLICY_ID,
            fleet_id: null,
            name: "Fresh policy name",
            description: "Fresh policy description",
            resolution: "Fresh resolution steps",
            platform: "darwin",
            query: "SELECT 'fresh';",
            critical: true,
          }),
        })
      )
    );

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: baseAppContext,
        // Stale values left over from a previously-viewed policy. The page must
        // ignore all of these and render the freshly-loaded policy instead.
        policy: {
          lastEditedQueryName: "Stale policy name",
          lastEditedQueryDescription: "Stale policy description",
          lastEditedQueryResolution: "Stale resolution steps",
          lastEditedQueryPlatform: "windows",
          lastEditedQueryBody: "SELECT 'stale';",
          lastEditedQueryCritical: false,
        },
      },
    });
    const { user, container } = render(
      <PolicyDetailsPage {...(createProps() as any)} />
    );

    // name + description
    expect(await screen.findByText("Fresh policy name")).toBeInTheDocument();
    expect(screen.getByText("Fresh policy description")).toBeInTheDocument();
    expect(screen.queryByText("Stale policy name")).not.toBeInTheDocument();
    expect(
      screen.queryByText("Stale policy description")
    ).not.toBeInTheDocument();

    // resolution
    expect(screen.getByText("Fresh resolution steps")).toBeInTheDocument();
    expect(
      screen.queryByText("Stale resolution steps")
    ).not.toBeInTheDocument();

    // platform ("darwin" displays as "macOS"; stale "windows" must not appear)
    expect(screen.getByText("macOS")).toBeInTheDocument();
    expect(screen.queryByText("Windows")).not.toBeInTheDocument();

    // critical (drives the critical-policy icon)
    expect(
      container.querySelector(".critical-policy-icon")
    ).toBeInTheDocument();

    // query (shown via the "Show query" modal)
    await user.click(screen.getByRole("button", { name: "Show query" }));
    expect(screen.getByTestId("show-query-modal")).toHaveTextContent(
      "SELECT 'fresh';"
    );
    expect(screen.getByTestId("show-query-modal")).not.toHaveTextContent(
      "SELECT 'stale';"
    );
  });
});

describe("PolicyDetailsPage - host search", () => {
  type TestLocation = {
    query: Record<string, string>;
    state?: { hostSearch?: string };
  };

  const renderPage = ({
    automationType,
    query = { fleet_id: "2" },
    state,
    isFreeTier = false,
  }: {
    automationType: PolicyAutomationType | null;
    query?: Record<string, string>;
    state?: { hostSearch?: string };
    isFreeTier?: boolean;
  }) => {
    mockServer.use(
      http.get(baseUrl(`/policies/${POLICY_ID}`), () =>
        HttpResponse.json({
          policy: createMockPolicy({
            id: POLICY_ID,
            fleet_id: null,
            automation_type: automationType,
          }),
        })
      )
    );
    const router = createMockRouter({ replace: jest.fn() });
    const props = {
      ...createProps(),
      router,
      location: {
        pathname: `/policies/${POLICY_ID}`,
        search: "",
        query,
        state,
      },
    };
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          ...baseAppContext,
          isFreeTier,
          isPremiumTier: !isFreeTier,
          availableTeams: isFreeTier ? [] : [{ id: 2, name: "Workstations" }],
          setCurrentTeam: jest.fn(),
        },
      },
    });
    const page = (location: TestLocation) => (
      <PolicyDetailsPage
        {...(({
          ...props,
          location: {
            ...props.location,
            query: location.query,
            state: location.state,
          },
        } as unknown) as React.ComponentProps<typeof PolicyDetailsPage>)}
      />
    );
    const { user, rerender } = render(page({ query, state }));
    // Stands in for the router delivering a new location after a replace.
    const setLocation = (location: TestLocation) => rerender(page(location));
    return { user, router, setLocation };
  };

  it.each<PolicyAutomationType>([
    "script",
    "webhook_or_ticket",
    "calendar",
    "conditional_access",
  ])(
    "filters a %s policy's runs to the host it was opened from and puts the host in the URL",
    async (automationType) => {
      const { router } = renderPage({
        automationType,
        state: { hostSearch: "Workstation 1" },
      });

      expect(await screen.findByTestId("activities-table")).toHaveTextContent(
        "Workstation 1"
      );
      expect(router.replace).toHaveBeenCalledWith({
        pathname: `/policies/${POLICY_ID}`,
        query: { fleet_id: "2", query: "Workstation 1" },
      });
    }
  );

  it("doesn't filter a policy without an automation and drops the host", async () => {
    const { router } = renderPage({
      automationType: null,
      query: { fleet_id: "2" },
      state: { hostSearch: "Workstation 1" },
    });

    expect(await screen.findByTestId("activities-table")).toHaveTextContent(
      /^$/
    );
    expect(router.replace).toHaveBeenCalledWith({
      pathname: `/policies/${POLICY_ID}`,
      query: { fleet_id: "2" },
    });
  });

  it("applies a search from the URL to any policy", async () => {
    const { router } = renderPage({
      automationType: null,
      query: { fleet_id: "2", query: "Workstation 1" },
    });

    expect(await screen.findByTestId("activities-table")).toHaveTextContent(
      "Workstation 1"
    );
    expect(router.replace).not.toHaveBeenCalled();
  });

  it("keeps the host filter when Free tier drops the fleet from the URL", async () => {
    const { router, setLocation } = renderPage({
      automationType: "webhook_or_ticket",
      isFreeTier: true,
      query: { fleet_id: "0" },
      state: { hostSearch: "Workstation 1" },
    });
    // Free tier has no fleets, so the page redirects without fleet_id, and the
    // string redirect carries no location state.
    await waitFor(() => expect(router.replace).toHaveBeenCalled());
    setLocation({ query: {} });

    expect(await screen.findByTestId("activities-table")).toHaveTextContent(
      "Workstation 1"
    );
    expect(router.replace).toHaveBeenLastCalledWith({
      pathname: `/policies/${POLICY_ID}`,
      query: { query: "Workstation 1" },
    });
  });

  it("doesn't bring the host back after the search is cleared", async () => {
    const { user, router, setLocation } = renderPage({
      automationType: "script",
      state: { hostSearch: "Workstation 1" },
    });
    await screen.findByTestId("activities-table");
    await waitFor(() =>
      expect(router.replace).toHaveBeenLastCalledWith({
        pathname: `/policies/${POLICY_ID}`,
        query: { fleet_id: "2", query: "Workstation 1" },
      })
    );
    setLocation({ query: { fleet_id: "2", query: "Workstation 1" } });

    await user.click(screen.getByText("Clear search"));
    setLocation({ query: { fleet_id: "2" } });

    expect(router.replace).toHaveBeenLastCalledWith({
      pathname: `/policies/${POLICY_ID}`,
      query: { fleet_id: "2" },
    });
  });

  it("keeps the URL in sync with the search", async () => {
    const { user, router } = renderPage({
      automationType: "script",
      query: { fleet_id: "2", query: "Workstation 1" },
    });

    await user.click(await screen.findByText("Search other host"));
    expect(router.replace).toHaveBeenLastCalledWith({
      pathname: `/policies/${POLICY_ID}`,
      query: { fleet_id: "2", query: "Other host" },
    });

    await user.click(screen.getByText("Clear search"));
    expect(router.replace).toHaveBeenLastCalledWith({
      pathname: `/policies/${POLICY_ID}`,
      query: { fleet_id: "2" },
    });
  });
});
