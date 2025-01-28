import React from "react";

import { screen, waitFor } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import createMockUser from "__mocks__/userMock";
import createMockQuery from "__mocks__/queryMock";

import { ISchedulableQuery } from "interfaces/schedulable_query";
import QueriesTable, { IQueriesTableProps } from "./QueriesTable";
import { enhanceQuery } from "../../ManageQueriesPage";

const testRawGlobalQueries: ISchedulableQuery[] = [
  {
    created_at: "2024-03-22T19:01:20Z",
    updated_at: "2024-03-22T19:01:20Z",
    id: 1,
    team_id: null,
    interval: 0,
    platform: "linux",
    min_osquery_version: "",
    automations_enabled: false,
    logging: "snapshot",
    name: "Global query 1",
    description: "Retrieves the OpenSSL version.",
    query:
      "SELECT name AS name, version AS version, 'deb_packages' AS source FROM deb_packages WHERE name LIKE 'openssl%' UNION SELECT name AS name, version AS version, 'apt_sources' AS source FROM apt_sources WHERE name LIKE 'openssl%' UNION SELECT name AS name, version AS version, 'rpm_packages' AS source FROM rpm_packages WHERE name LIKE 'openssl%';",
    saved: true,
    observer_can_run: true,
    author_id: 1,
    author_name: "Tess Tuser",
    author_email: "tess@fake.com",
    packs: [],
    stats: {
      system_time_p50: null,
      system_time_p95: null,
      user_time_p50: null,
      user_time_p95: null,
      total_executions: 0,
    },
    discard_data: false,
  },
  {
    created_at: "2024-03-22T19:01:20Z",
    updated_at: "2024-03-22T19:01:20Z",
    id: 2,
    team_id: null,
    interval: 0,
    platform: "linux",
    min_osquery_version: "",
    automations_enabled: false,
    logging: "snapshot",
    name: "Global query 2",
    description: "Retrieves the OpenSSL version.",
    query:
      "SELECT name AS name, version AS version, 'deb_packages' AS source FROM deb_packages WHERE name LIKE 'openssl%' UNION SELECT name AS name, version AS version, 'apt_sources' AS source FROM apt_sources WHERE name LIKE 'openssl%' UNION SELECT name AS name, version AS version, 'rpm_packages' AS source FROM rpm_packages WHERE name LIKE 'openssl%';",
    saved: true,
    observer_can_run: true,
    author_id: 1,
    author_name: "Tess Tuser",
    author_email: "tess@fake.com",
    packs: [],
    stats: {
      system_time_p50: null,
      system_time_p95: null,
      user_time_p50: null,
      user_time_p95: null,
      total_executions: 0,
    },
    discard_data: false,
  },
];

const testRawTeamQueries: ISchedulableQuery[] = [
  {
    created_at: "2024-04-25T04:16:09Z",
    updated_at: "2024-04-25T04:16:09Z",
    id: 3,
    team_id: 1,
    interval: 3600,
    platform: "",
    min_osquery_version: "",
    automations_enabled: false,
    logging: "snapshot",
    name: "Team query 1",
    description: "",
    query: "SELECT * FROM osquery_info;",
    saved: true,
    observer_can_run: false,
    author_id: 1,
    author_name: "Tess Tuser",
    author_email: "tess@fake.com",
    packs: [],
    stats: {
      system_time_p50: null,
      system_time_p95: null,
      user_time_p50: null,
      user_time_p95: null,
      total_executions: 0,
    },
    discard_data: false,
  },
  {
    created_at: "2024-04-25T04:16:09Z",
    updated_at: "2024-04-25T04:16:09Z",
    id: 4,
    team_id: 1,
    interval: 3600,
    platform: "",
    min_osquery_version: "",
    automations_enabled: false,
    logging: "snapshot",
    name: "Team query 2",
    description: "",
    query: "SELECT * FROM osquery_info;",
    saved: true,
    observer_can_run: true,
    author_id: 1,
    author_name: "Tess Tuser",
    author_email: "tess@fake.com",
    packs: [],
    stats: {
      system_time_p50: null,
      system_time_p95: null,
      user_time_p50: null,
      user_time_p95: null,
      total_executions: 0,
    },
    discard_data: false,
  },
];

const testGlobalQueries = testRawGlobalQueries.map(enhanceQuery);
const testTeamQueries = testRawTeamQueries.map(enhanceQuery);

const renderAsPremiumGlobalAdmin = createCustomRenderer({
  context: {
    app: {
      isPremiumTier: true,
      isGlobalAdmin: true,
      currentUser: createMockUser(),
    },
  },
});
describe("QueriesTable", () => {
  it("Renders the page-wide empty state when no queries are present", () => {
    const testData: IQueriesTableProps[] = [
      {
        queries: [],
        totalQueriesCount: 0,
        hasNextResults: false,
        onlyInheritedQueries: false,
        isLoading: false,
        onDeleteQueryClick: jest.fn(),
        onCreateQueryClick: jest.fn(),
        isOnlyObserver: false,
        isObserverPlus: false,
        isAnyTeamObserverPlus: false,
        currentTeamId: undefined,
      },
    ];

    testData.forEach((tableProps) => {
      renderAsPremiumGlobalAdmin(<QueriesTable {...tableProps} />);
      expect(
        screen.getByText("You don't have any queries")
      ).toBeInTheDocument();
      expect(screen.queryByText("Frequency")).toBeNull();
    });
  });
  it("Renders inherited global queries and team queries when viewing a team, then renders the 'no-matching' empty state when a search string is entered that matches no queries", async () => {
    const testData: IQueriesTableProps[] = [
      {
        queries: [...testGlobalQueries, ...testTeamQueries],
        totalQueriesCount: 4,
        hasNextResults: false,
        onlyInheritedQueries: false,
        isLoading: false,
        onDeleteQueryClick: jest.fn(),
        onCreateQueryClick: jest.fn(),
        isOnlyObserver: false,
        isObserverPlus: false,
        isAnyTeamObserverPlus: false,
        currentTeamId: 1,
      },
    ];
    const dataStrings = [
      "Global query 1",
      "Global query 2",
      "Inherited",
      "Frequency",
      "Team query 1",
      "Team query 2",
    ];

    testData.forEach(async (tableProps) => {
      // will have no context to get current user from
      const { user } = renderAsPremiumGlobalAdmin(
        <QueriesTable {...tableProps} />
      );
      dataStrings.forEach((val) => {
        expect(screen.getAllByText(val)[0]).toBeInTheDocument();
      });

      await user.type(
        screen.getByPlaceholderText("Search by name"),
        "shouldn't match anything"
      );
      expect(screen.getByText("No matching queries")).toBeInTheDocument();
      dataStrings.forEach((val) => {
        expect(screen.getAllByText(val)).toHaveLength(0);
      });
    });
  });

  it("Renders an observer can run badge and tooltip for a observer can run query", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const testObserverCanRunQuery = [
      createMockQuery({
        observer_can_run: true,
      }),
    ];
    const testQueries = testObserverCanRunQuery.map(enhanceQuery);

    const { user } = render(
      <QueriesTable
        queries={testQueries}
        totalQueriesCount={1}
        hasNextResults={false}
        onlyInheritedQueries={false}
        isLoading={false}
        onDeleteQueryClick={jest.fn()}
        onCreateQueryClick={jest.fn()}
        isOnlyObserver={false}
        isObserverPlus={false}
        isAnyTeamObserverPlus={false}
        currentTeamId={1}
      />
    );

    await waitFor(() => {
      waitFor(() => {
        user.hover(screen.getByTestId("query-icon"));
      });

      expect(
        screen.getByText("Observers can run this query.")
      ).toBeInTheDocument();
    });
  });

  it("Renders an inherited badge and tooltip for inherited query on a team's queries page", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const testInheritedQuery = [createMockQuery()];

    const testQueries = testInheritedQuery.map(enhanceQuery);

    const { user } = render(
      <QueriesTable
        queries={testQueries}
        totalQueriesCount={1}
        hasNextResults={false}
        onlyInheritedQueries={false}
        isLoading={false}
        onDeleteQueryClick={jest.fn()}
        onCreateQueryClick={jest.fn()}
        isOnlyObserver={false}
        isObserverPlus={false}
        isAnyTeamObserverPlus={false}
        currentTeamId={1}
      />
    );

    await waitFor(() => {
      waitFor(() => {
        user.hover(screen.getByText("Inherited"));
      });

      expect(
        screen.getByText("This query runs on all hosts.")
      ).toBeInTheDocument();
    });
  });

  it("Does not render an inherited badge and tooltip for global query on the All team's queries page", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const testGlobalQuery = [createMockQuery()];
    const testQueries = testGlobalQuery.map(enhanceQuery);

    render(
      <QueriesTable
        queries={testQueries}
        totalQueriesCount={1}
        hasNextResults={false}
        onlyInheritedQueries={false}
        isLoading={false}
        onDeleteQueryClick={jest.fn()}
        onCreateQueryClick={jest.fn()}
        isOnlyObserver={false}
        isObserverPlus={false}
        isAnyTeamObserverPlus={false}
        currentTeamId={undefined}
      />
    );

    expect(screen.queryByText("Inherited")).not.toBeInTheDocument();
  });

  it("Renders the correct number of checkboxes for team queries and not inherited queries on a team's queries page and can check select all box", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const { container, user } = render(
      <QueriesTable
        queries={[...testTeamQueries, ...testGlobalQueries]}
        totalQueriesCount={4}
        hasNextResults={false}
        onlyInheritedQueries={false}
        isLoading={false}
        onDeleteQueryClick={jest.fn()}
        onCreateQueryClick={jest.fn()}
        isOnlyObserver={false}
        isObserverPlus={false}
        isAnyTeamObserverPlus={false}
        currentTeamId={1}
      />
    );

    const numberOfCheckboxes = container.querySelectorAll(
      "input[type='checkbox']"
    ).length;

    expect(numberOfCheckboxes).toBe(
      testTeamQueries.length + 1 // +1 for Select all checkbox
    );

    const checkbox = container.querySelectorAll(
      "input[type='checkbox']"
    )[0] as HTMLInputElement;

    await waitFor(() => {
      waitFor(() => {
        user.click(checkbox);
      });

      expect(checkbox.checked).toBe(true);
    });
  });
});
