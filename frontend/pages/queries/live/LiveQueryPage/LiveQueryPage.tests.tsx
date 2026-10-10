import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import createMockHost from "__mocks__/hostMock";
import createMockPolicy from "__mocks__/policyMock";
import createMockQuery from "__mocks__/scheduleableQueryMock";
import createMockUser from "__mocks__/userMock";
import QueryProvider from "context/query";
import LivePolicyPage from "pages/policies/live/LivePolicyPage/LivePolicyPage";
import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";
import { getPathWithQueryParams } from "utilities/url";

import LiveQueryPage from "./LiveQueryPage";

// Keep the navigation test independent of distributed-query sockets/results.
jest.mock("pages/queries/live/screens/RunQuery", () => ({
  __esModule: true,
  default: ({ goToQueryEditor }: { goToQueryEditor: () => void }) => (
    <button onClick={goToQueryEditor}>Close</button>
  ),
}));
jest.mock("pages/policies/live/screens/RunQuery", () => ({
  __esModule: true,
  default: ({ goToQueryEditor }: { goToQueryEditor: () => void }) => (
    <button onClick={goToQueryEditor}>Close</button>
  ),
}));

const renderPage = createCustomRenderer({
  withBackendMock: true,
  context: {
    app: {
      config: createMockConfig(),
      currentUser: createMockUser({ global_role: "admin" }),
      isOnGlobalTeam: true,
      isPremiumTier: true,
      availableTeams: [
        { id: -1, name: "All fleets" },
        { id: 0, name: "Unassigned" },
        { id: 2, name: "Fleet 2" },
      ],
      setCurrentTeam: jest.fn(),
    },
  },
});

beforeEach(() => {
  mockServer.use(
    http.get(baseUrl("/reports/1"), () =>
      HttpResponse.json({ query: createMockQuery({ id: 1 }) })
    ),
    http.get(baseUrl("/policies/1"), () =>
      HttpResponse.json({ policy: createMockPolicy({ id: 1 }) })
    ),
    http.get(baseUrl("/hosts/42"), () =>
      HttpResponse.json({ host: createMockHost({ id: 42 }) })
    ),
    http.get(baseUrl("/fleets"), () => HttpResponse.json({ teams: [] })),
    http.get(baseUrl("/labels/summary"), () =>
      HttpResponse.json({
        labels: [{ id: 1, name: "All Hosts", label_type: "builtin" }],
      })
    ),
    http.post(baseUrl("/targets/count"), () =>
      HttpResponse.json({
        targets_count: 1,
        targets_online: 1,
        targets_offline: 0,
      })
    )
  );
});

describe.each([
  { kind: "reports", Page: LiveQueryPage, hostParam: "host_id" },
  { kind: "policies", Page: LivePolicyPage, hostParam: "host_ids" },
])("Live $kind return navigation", ({ kind, Page: Component, hostParam }) => {
  const Page = (props: React.ComponentProps<typeof LiveQueryPage>) => (
    <QueryProvider>
      <Component {...props} />
    </QueryProvider>
  );
  it.each([
    [undefined, "1", "1"],
    ["edit", "1", "1/edit"],
    ["unexpected", "1", "1"],
    [undefined, undefined, "new"],
    ["edit", undefined, "new"],
  ])(
    "returns to the originating page (from=%s, id=%s)",
    async (from, id, destination) => {
      const router = createMockRouter();
      const query = { fleet_id: "2", from };
      const { user } = renderPage(
        <Page
          router={router}
          params={id ? { id } : {}}
          location={{
            pathname: `/${kind}/${id || "new"}/live`,
            query,
            search: "?fleet_id=2",
          }}
        />
      );
      await user.click(await screen.findByRole("button", { name: "Cancel" }));
      expect(router.push).toHaveBeenLastCalledWith(
        `/${kind}/${destination}?fleet_id=2`
      );

      await user.click(screen.getByRole("button", { name: /All hosts/ }));
      await waitFor(() =>
        expect(screen.getByRole("button", { name: "Run" })).toBeEnabled()
      );
      await user.click(screen.getByRole("button", { name: "Run" }));
      await user.click(screen.getByRole("button", { name: "Close" }));
      expect(router.push).toHaveBeenLastCalledWith(
        `/${kind}/${destination}?fleet_id=2`
      );
    }
  );

  it("keeps the origin and fleet when consuming a preselected host", async () => {
    const router = createMockRouter();
    const query = { fleet_id: "2", from: "edit", [hostParam]: "42" };
    const pathname = `/${kind}/1/live`;
    const { user, rerender } = renderPage(
      <Page
        router={router}
        params={{ id: "1" }}
        location={{ pathname, query, search: "?fleet_id=2" }}
      />
    );
    await waitFor(() =>
      expect(router.replace).toHaveBeenCalledWith(
        getPathWithQueryParams(pathname, { fleet_id: "2", from: "edit" })
      )
    );
    rerender(
      <Page
        router={router}
        params={{ id: "1" }}
        location={{
          pathname,
          query: { fleet_id: "2", from: "edit" },
          search: "?fleet_id=2&from=edit",
        }}
      />
    );
    await user.click(await screen.findByRole("button", { name: "Cancel" }));
    expect(router.push).toHaveBeenLastCalledWith(`/${kind}/1/edit?fleet_id=2`);
  });

  it("omits the fleet parameter for All fleets", async () => {
    const router = createMockRouter();
    const { user } = renderPage(
      <Page
        router={router}
        params={{ id: "1" }}
        location={{
          pathname: `/${kind}/1/live`,
          query: {},
          search: "",
        }}
      />
    );
    await user.click(await screen.findByRole("button", { name: "Cancel" }));
    expect(router.push).toHaveBeenLastCalledWith(`/${kind}/1`);
  });
});
