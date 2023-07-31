import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockQuery from "__mocks__/queryMock";
import createMockUser from "__mocks__/userMock";

import QueryForm from "./QueryForm";

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

describe("QueryForm - component", () => {
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
        },
      },
    });

    render(
      <QueryForm
        router={mockRouter}
        queryIdForEdit={1}
        apiTeamIdForQuery={1}
        teamNameForQuery={"Apples"}
        showOpenSchemaActionText
        storedQuery={createMockQuery({ name: "" })} // empty name
        isStoredQueryLoading={false}
        isQuerySaving={false}
        isQueryUpdating={false}
        saveQuery={jest.fn()}
        onOsqueryTableSelect={jest.fn()}
        goToSelectTargets={jest.fn()}
        onUpdate={jest.fn()}
        onOpenSchemaSidebar={jest.fn()}
        renderLiveQueryWarning={jest.fn()}
        backendValidators={{}}
      />
    );

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("disables save for sql error", async () => {
    const render = createCustomRenderer({
      context: {
        query: {
          lastEditedQueryId: mockQuery.id,
          lastEditedQueryName: mockQuery.name,
          lastEditedQueryDescription: mockQuery.description,
          lastEditedQueryBody: "select ** from users;",
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
        },
      },
    });

    const { user } = render(
      <QueryForm
        router={mockRouter}
        queryIdForEdit={1}
        apiTeamIdForQuery={1}
        teamNameForQuery={"Apples"}
        showOpenSchemaActionText
        storedQuery={createMockQuery()}
        isStoredQueryLoading={false}
        isQuerySaving={false}
        isQueryUpdating={false}
        saveQuery={jest.fn()}
        onOsqueryTableSelect={jest.fn()}
        goToSelectTargets={jest.fn()}
        onUpdate={jest.fn()}
        onOpenSchemaSidebar={jest.fn()}
        renderLiveQueryWarning={jest.fn()}
        backendValidators={{}}
      />
    );

    // TODO: How to modify ace editor using react testing library
    // await user.type(screen.getByLabelText(/query/), "select ** from users;");

    // const aceTextareaPlaceholder = screen.getByText("SELECT * FROM users");
    // const parent = aceTextareaPlaceholder.parentElement?.parentElement?.querySelector(
    //   "textarea"
    // ) as HTMLTextAreaElement;
    // parent.focus();
    // user.paste("SELECT ** FROM users;");

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Run" })).toBeDisabled();
  });
});
