import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockQuery from "__mocks__/queryMock";
import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";

import EditQueryForm from "./EditQueryForm";

const mockQuery = createMockQuery();
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

describe("EditQueryForm - component", () => {
  it("disables save button for missing query name", async () => {
    const render = createCustomRenderer({
      context: {
        query: {
          lastEditedQueryId: mockQuery.id,
          lastEditedQueryName: "", // missing query name
          lastEditedQueryDescription: mockQuery.description,
          lastEditedQueryBody: mockQuery.query,
          lastEditedQueryObserverCanRun: mockQuery.observer_can_run,
          lastEditedQueryFrequency: mockQuery.interval,
          lastEditedQueryPlatforms: mockQuery.platform,
          lastEditedQueryMinOsqueryVersion: mockQuery.min_osquery_version,
          lastEditedQueryLoggingType: mockQuery.logging,
          setLastEditedQueryName: jest.fn(),
          setLastEditedQueryDescription: jest.fn(),
          setLastEditedQueryBody: jest.fn(),
          setLastEditedQueryObserverCanRun: jest.fn(),
          setLastEditedQueryFrequency: jest.fn(),
          setLastEditedQueryPlatforms: jest.fn(),
          setLastEditedQueryMinOsqueryVersion: jest.fn(),
          setLastEditedQueryLoggingType: jest.fn(),
        },
        app: {
          currentUser: createMockUser(),
          isGlobalObserver: false,
          isGlobalAdmin: true,
          isGlobalMaintainer: false,
          isOnGlobalTeam: true,
          isPremiumTier: true,
          isSandboxMode: false,
          config: createMockConfig(),
        },
      },
    });

    render(
      <EditQueryForm
        router={mockRouter}
        queryIdForEdit={1}
        apiTeamIdForQuery={1}
        teamNameForQuery="Apples"
        showOpenSchemaActionText
        storedQuery={createMockQuery({ name: "" })} // empty name
        isStoredQueryLoading={false}
        isQuerySaving={false}
        isQueryUpdating={false}
        onSubmitNewQuery={jest.fn()}
        onOsqueryTableSelect={jest.fn()}
        onUpdate={jest.fn()}
        onOpenSchemaSidebar={jest.fn()}
        renderLiveQueryWarning={jest.fn()}
        backendValidators={{}}
        showConfirmSaveChangesModal={false}
        setShowConfirmSaveChangesModal={jest.fn()}
      />
    );

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("disables live query button for globally disabled live queries", async () => {
    const render = createCustomRenderer({
      context: {
        query: {
          lastEditedQueryId: mockQuery.id,
          lastEditedQueryName: "", // missing query name
          lastEditedQueryDescription: mockQuery.description,
          lastEditedQueryBody: mockQuery.query,
          lastEditedQueryObserverCanRun: mockQuery.observer_can_run,
          lastEditedQueryFrequency: mockQuery.interval,
          lastEditedQueryPlatforms: mockQuery.platform,
          lastEditedQueryMinOsqueryVersion: mockQuery.min_osquery_version,
          lastEditedQueryLoggingType: mockQuery.logging,
          setLastEditedQueryName: jest.fn(),
          setLastEditedQueryDescription: jest.fn(),
          setLastEditedQueryBody: jest.fn(),
          setLastEditedQueryObserverCanRun: jest.fn(),
          setLastEditedQueryFrequency: jest.fn(),
          setLastEditedQueryPlatforms: jest.fn(),
          setLastEditedQueryMinOsqueryVersion: jest.fn(),
          setLastEditedQueryLoggingType: jest.fn(),
        },
        app: {
          currentUser: createMockUser(),
          isGlobalObserver: false,
          isGlobalAdmin: true,
          isGlobalMaintainer: false,
          isOnGlobalTeam: true,
          isPremiumTier: true,
          isSandboxMode: false,
          config: createMockConfig({
            server_settings: {
              ...createMockConfig().server_settings,
              live_query_disabled: true, // Live query disabled
            },
          }),
        },
      },
    });

    const { container, user } = render(
      <EditQueryForm
        router={mockRouter}
        queryIdForEdit={1}
        apiTeamIdForQuery={1}
        teamNameForQuery="Apples"
        showOpenSchemaActionText
        storedQuery={createMockQuery({ name: "Mock query" })}
        isStoredQueryLoading={false}
        isQuerySaving={false}
        isQueryUpdating={false}
        onSubmitNewQuery={jest.fn()}
        onOsqueryTableSelect={jest.fn()}
        onUpdate={jest.fn()}
        onOpenSchemaSidebar={jest.fn()}
        renderLiveQueryWarning={jest.fn()}
        backendValidators={{}}
        showConfirmSaveChangesModal={false}
        setShowConfirmSaveChangesModal={jest.fn()}
      />
    );

    expect(screen.getByRole("button", { name: "Live query" })).toBeDisabled();

    await user.hover(screen.getByRole("button", { name: "Live query" }));

    expect(container.querySelector("#live-query-button")).toHaveTextContent(
      /live queries are disabled/i
    );
  });
  // TODO: Consider testing save button is disabled for a sql error
  // Trickiness is in modifying react-ace using react-testing library
});
