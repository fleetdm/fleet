import React from "react";
import { screen, waitFor, within } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";
import userEvent from "@testing-library/user-event";

import createMockQuery from "__mocks__/queryMock";
import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";

import queryAPI from "services/entities/queries";
import EditQueryForm from "./EditQueryForm";

jest.mock("services/entities/queries");

const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

const mockLabels = [
  {
    id: 1,
    name: "Fun",
    description: "Computers that like to have a good time",
    label_type: "regular",
  },
  {
    id: 2,
    name: "Fresh",
    description: "Laptops with dirty mouths",
    label_type: "regular",
  },
];

const labelSummariesHandler = http.get(baseUrl("/labels/summary"), () => {
  return HttpResponse.json({
    labels: mockLabels,
  });
});

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
      withBackendMock: true,
      context: {
        query: {
          lastEditedQueryId: mockQuery.id,
          lastEditedQueryName: "", // missing query name
          lastEditedQueryDescription: mockQuery.description,
          lastEditedQueryBody: mockQuery.query,
          lastEditedQueryObserverCanRun: mockQuery.observer_can_run,
          lastEditedQueryFrequency: mockQuery.interval,
          lastEditedQueryAutomationsEnabled: mockQuery.automations_enabled,
          lastEditedQueryPlatforms: mockQuery.platform,
          lastEditedQueryMinOsqueryVersion: mockQuery.min_osquery_version,
          lastEditedQueryLoggingType: mockQuery.logging,
          setLastEditedQueryName: jest.fn(),
          setLastEditedQueryDescription: jest.fn(),
          setLastEditedQueryBody: jest.fn(),
          setLastEditedQueryObserverCanRun: jest.fn(),
          setLastEditedQueryFrequency: jest.fn(),
          setLastEditedQueryAutomationsEnabled: jest.fn(),
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
          isPremiumTier: false,
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
      withBackendMock: true,
      context: {
        query: {
          lastEditedQueryId: mockQuery.id,
          lastEditedQueryName: "", // missing query name
          lastEditedQueryDescription: mockQuery.description,
          lastEditedQueryBody: mockQuery.query,
          lastEditedQueryObserverCanRun: mockQuery.observer_can_run,
          lastEditedQueryFrequency: mockQuery.interval,
          lastEditedQueryAutomationsEnabled: mockQuery.automations_enabled,
          lastEditedQueryPlatforms: mockQuery.platform,
          lastEditedQueryMinOsqueryVersion: mockQuery.min_osquery_version,
          lastEditedQueryLoggingType: mockQuery.logging,
          setLastEditedQueryName: jest.fn(),
          setLastEditedQueryDescription: jest.fn(),
          setLastEditedQueryBody: jest.fn(),
          setLastEditedQueryObserverCanRun: jest.fn(),
          setLastEditedQueryFrequency: jest.fn(),
          setLastEditedQueryAutomationsEnabled: jest.fn(),
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
          isPremiumTier: false,
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

  it("shows automations warning icon when query frequency is set to 0", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        query: {
          lastEditedQueryId: mockQuery.id,
          lastEditedQueryName: "Test Query",
          lastEditedQueryDescription: mockQuery.description,
          lastEditedQueryBody: mockQuery.query,
          lastEditedQueryObserverCanRun: mockQuery.observer_can_run,
          lastEditedQueryFrequency: 0, // Set frequency to 0
          lastEditedQueryAutomationsEnabled: true, // Enable automations
          lastEditedQueryPlatforms: mockQuery.platform,
          lastEditedQueryMinOsqueryVersion: mockQuery.min_osquery_version,
          lastEditedQueryLoggingType: mockQuery.logging,
          setLastEditedQueryName: jest.fn(),
          setLastEditedQueryDescription: jest.fn(),
          setLastEditedQueryBody: jest.fn(),
          setLastEditedQueryObserverCanRun: jest.fn(),
          setLastEditedQueryAutomationsEnabled: jest.fn(),
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
          isPremiumTier: false,
          isSandboxMode: false,
          config: createMockConfig(),
        },
      },
    });

    const { user } = render(
      <EditQueryForm
        router={mockRouter}
        queryIdForEdit={1}
        apiTeamIdForQuery={1}
        teamNameForQuery="Apples"
        showOpenSchemaActionText
        storedQuery={createMockQuery({ interval: 0 })}
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

    // Find the interval dropdown
    const intervalDropdown = screen
      .getByText("Interval")
      .closest(".form-field--dropdown") as HTMLElement;
    expect(intervalDropdown).toBeInTheDocument();

    // Check if the interval is set to "Never"
    const selectedInterval = within(intervalDropdown).getByText("Never");
    expect(selectedInterval).toBeInTheDocument();

    // Find the automations slider
    const automationsSlider = screen
      .getByText("Automations on")
      .closest(".fleet-slider__wrapper") as HTMLElement;
    expect(automationsSlider).toBeInTheDocument();

    // Check if the automations are enabled
    const automationsButton = within(automationsSlider).getByRole("switch");
    expect(automationsButton).toHaveClass("fleet-slider--active");

    // Check if the warning icon is present
    const warningIcon = within(automationsSlider).getByTestId("warning-icon");
    expect(warningIcon).toBeInTheDocument();
  });

  it("should not show the target selector in the free tier", async () => {
    mockServer.use(labelSummariesHandler);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        query: {
          lastEditedQueryId: mockQuery.id,
          lastEditedQueryName: "", // missing query name
          lastEditedQueryDescription: mockQuery.description,
          lastEditedQueryBody: mockQuery.query,
          lastEditedQueryObserverCanRun: mockQuery.observer_can_run,
          lastEditedQueryFrequency: mockQuery.interval,
          lastEditedQueryAutomationsEnabled: mockQuery.automations_enabled,
          lastEditedQueryPlatforms: mockQuery.platform,
          lastEditedQueryMinOsqueryVersion: mockQuery.min_osquery_version,
          lastEditedQueryLoggingType: mockQuery.logging,
          setLastEditedQueryName: jest.fn(),
          setLastEditedQueryDescription: jest.fn(),
          setLastEditedQueryBody: jest.fn(),
          setLastEditedQueryObserverCanRun: jest.fn(),
          setLastEditedQueryFrequency: jest.fn(),
          setLastEditedQueryAutomationsEnabled: jest.fn(),
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
          isPremiumTier: false,
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

    // Wait for any queries (that should not be happening) to finish.
    await new Promise((resolve) => setTimeout(resolve, 500));

    // Check that the target selector is not present.
    expect(screen.queryByText("All hosts")).not.toBeInTheDocument();
  });

  // TODO: Consider testing save button is disabled for a sql error
  // Trickiness is in modifying react-ace using react-testing library

  describe("in premium tier", () => {
    const onUpdate = jest.fn();
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        query: {
          lastEditedQueryId: mockQuery.id,
          lastEditedQueryName: "Some query", // missing query name
          lastEditedQueryDescription: mockQuery.description,
          lastEditedQueryBody: mockQuery.query,
          lastEditedQueryObserverCanRun: mockQuery.observer_can_run,
          lastEditedQueryFrequency: mockQuery.interval,
          lastEditedQueryAutomationsEnabled: mockQuery.automations_enabled,
          lastEditedQueryPlatforms: mockQuery.platform,
          lastEditedQueryMinOsqueryVersion: mockQuery.min_osquery_version,
          lastEditedQueryLoggingType: mockQuery.logging,
          setLastEditedQueryName: jest.fn(),
          setLastEditedQueryDescription: jest.fn(),
          setLastEditedQueryBody: jest.fn(),
          setLastEditedQueryObserverCanRun: jest.fn(),
          setLastEditedQueryFrequency: jest.fn(),
          setLastEditedQueryAutomationsEnabled: jest.fn(),
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

    const props = {
      router: mockRouter,
      queryIdForEdit: 1,
      apiTeamIdForQuery: 1,
      teamNameForQuery: "Apples",
      showOpenSchemaActionText: true,
      storedQuery: createMockQuery(),
      isStoredQueryLoading: false,
      isQuerySaving: false,
      isQueryUpdating: false,
      onSubmitNewQuery: jest.fn(),
      onOsqueryTableSelect: jest.fn(),
      onUpdate,
      onOpenSchemaSidebar: jest.fn(),
      renderLiveQueryWarning: jest.fn(),
      backendValidators: {},
      showConfirmSaveChangesModal: false,
      setShowConfirmSaveChangesModal: jest.fn(),
    };

    beforeEach(() => {
      onUpdate.mockClear();
      mockServer.use(labelSummariesHandler);
    });

    it("should show the target selector in All hosts target mode when the query has no labels", async () => {
      render(<EditQueryForm {...props} />);
      await waitFor(() => {
        expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
        expect(screen.getByLabelText("Custom")).toBeInTheDocument();
        expect(screen.getByLabelText("All hosts")).toBeChecked();
      });
    });

    it("should show the target selector in Custom target mode when the query has labels", async () => {
      const testProps = {
        ...props,
        storedQuery: createMockQuery({
          name: "Some query",
          labels_include_any: [{ name: "Fun", id: 1 }],
        }),
      };
      render(<EditQueryForm {...testProps} />);
      await waitFor(() => {
        expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
        expect(screen.getByLabelText("Custom")).toBeInTheDocument();
        expect(screen.getByLabelText("Custom")).toBeChecked();
        expect(screen.getByLabelText("Fun")).toBeChecked();
        expect(screen.getByLabelText("Fresh")).not.toBeChecked();
        expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
      });
    });

    it("should disable the save button in Custom target mode when no labels are selected", async () => {
      const testProps = {
        ...props,
        storedQuery: createMockQuery({
          labels_include_any: [{ name: "Fun", id: 1 }],
        }),
      };
      render(<EditQueryForm {...testProps} />);
      let saveButton;
      let funButton;
      await waitFor(() => {
        expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
        expect(screen.getByLabelText("Custom")).toBeInTheDocument();
        expect(screen.getByLabelText("Custom")).toBeChecked();
        funButton = screen.getByLabelText("Fun");
        expect(funButton).toBeChecked();
        expect(screen.getByLabelText("Fresh")).not.toBeChecked();
        saveButton = screen.getByRole("button", { name: "Save" });
        expect(saveButton).toBeEnabled();
      });

      // Unchecking the only selected label should disable the save button.
      funButton && (await userEvent.click(funButton));
      expect(saveButton).toBeDisabled();

      // Re-checking it should enable it.
      funButton && (await userEvent.click(funButton));
      expect(saveButton).toBeEnabled();
    });

    it("should send labels when updating a query in Custom target mode", async () => {
      const testProps = {
        ...props,
        storedQuery: createMockQuery({
          labels_include_any: [{ name: "Fun", id: 1 }],
        }),
      };
      render(<EditQueryForm {...testProps} />);
      await waitFor(() => {
        expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
      });

      await userEvent.click(screen.getByRole("button", { name: "Save" }));

      expect(onUpdate.mock.calls[0][0].labels_include_any).toEqual(["Fun"]);
    });

    it("should clear labels when updating a query in All hosts target mode", async () => {
      const testProps = {
        ...props,
        storedQuery: createMockQuery({
          labels_include_any: [{ name: "Fun", id: 1 }],
        }),
      };
      render(<EditQueryForm {...testProps} />);
      let allHosts;
      await waitFor(() => {
        allHosts = screen.getByLabelText("All hosts");
        expect(allHosts).toBeInTheDocument();
      });
      allHosts && (await userEvent.click(allHosts));
      await userEvent.click(screen.getByRole("button", { name: "Save" }));

      expect(onUpdate.mock.calls[0][0].labels_include_any).toEqual([]);
    });

    it("should send labels when saving a new query in Custom target mode", async () => {
      // Mock the create query API with a never-returning promise, so we can just
      // spy on the request without having to mock anything else.
      const createFn = jest
        .spyOn(queryAPI, "create")
        .mockImplementation(() => new Promise(jest.fn()));
      const testProps = {
        ...props,
        storedQuery: createMockQuery({
          labels_include_any: [{ name: "Fun", id: 1 }],
        }),
      };
      render(<EditQueryForm {...testProps} />);
      await waitFor(() => {
        expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
      });

      await userEvent.click(
        screen.getByRole("button", { name: "Save as new" })
      );

      expect(createFn.mock.calls[0][0].labels_include_any).toEqual(["Fun"]);
    });

    it("should clear labels when saving a new query in All hosts target mode", async () => {
      // Mock the create query API with a never-returning promise, so we can just
      // spy on the request without having to mock anything else.
      const createFn = jest
        .spyOn(queryAPI, "create")
        .mockImplementation(() => new Promise(jest.fn()));
      const testProps = {
        ...props,
        storedQuery: createMockQuery({
          labels_include_any: [{ name: "Fun", id: 1 }],
        }),
      };
      render(<EditQueryForm {...testProps} />);
      let allHosts;
      await waitFor(() => {
        allHosts = screen.getByLabelText("All hosts");
        expect(allHosts).toBeInTheDocument();
      });
      allHosts && (await userEvent.click(allHosts));

      await userEvent.click(
        screen.getByRole("button", { name: "Save as new" })
      );

      expect(createFn.mock.calls[0][0].labels_include_any).toEqual([]);
    });
  });
});
