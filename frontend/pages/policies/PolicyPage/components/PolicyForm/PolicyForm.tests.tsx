import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";
import userEvent from "@testing-library/user-event";

import createMockPolicy from "__mocks__/policyMock";
import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";

import { ILabelSummary } from "interfaces/label";
import PolicyProvider from "context/policy";
import PolicyForm from "./PolicyForm";

const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

const mockPolicy = createMockPolicy();

const mockLabels: ILabelSummary[] = [
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

describe("PolicyForm - component", () => {
  const defaultProps = {
    policyIdForEdit: mockPolicy.id,
    showOpenSchemaActionText: false,
    storedPolicy: createMockPolicy({ name: "Foo" }),
    isStoredPolicyLoading: false,
    isTeamObserver: false,
    isUpdatingPolicy: false,
    onCreatePolicy: jest.fn(),
    onOsqueryTableSelect: jest.fn(),
    goToSelectTargets: jest.fn(),
    onUpdate: jest.fn(),
    onOpenSchemaSidebar: jest.fn(),
    renderLiveQueryWarning: jest.fn(),
    backendValidators: {},
    onClickAutofillDescription: jest.fn(),
    onClickAutofillResolution: jest.fn(),
    isFetchingAutofillDescription: false,
    isFetchingAutofillResolution: false,
    resetAiAutofillData: jest.fn(),
  };

  it("should not show the target selector in the free tier", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          currentUser: createMockUser(),
          config: createMockConfig(),
          isPremiumTier: false,
        },
      },
    });

    render(<PolicyForm {...defaultProps} />);

    // Wait for any queries (that should not be happening) to finish.
    await new Promise((resolve) => setTimeout(resolve, 500));

    // Check that the target selector is not present.
    expect(screen.queryByText("All hosts")).not.toBeInTheDocument();
  });

  describe("in premium tier", () => {
    beforeEach(() => {
      mockServer.use(labelSummariesHandler);
    });

    it("disables save button for missing policy name", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          policy: {
            policyTeamId: undefined,
            lastEditedQueryId: mockPolicy.id,
            lastEditedQueryName: "", // missing policy name
            lastEditedQueryDescription: mockPolicy.description,
            lastEditedQueryBody: mockPolicy.query,
            lastEditedQueryResolution: mockPolicy.resolution,
            lastEditedQueryCritical: mockPolicy.critical,
            lastEditedQueryPlatform: mockPolicy.platform,
            lastEditedQueryLabelsIncludeAny: [],
            lastEditedQueryLabelsExcludeAny: [],
            defaultPolicy: false,
            setLastEditedQueryName: jest.fn(),
            setLastEditedQueryDescription: jest.fn(),
            setLastEditedQueryBody: jest.fn(),
            setLastEditedQueryResolution: jest.fn(),
            setLastEditedQueryCritical: jest.fn(),
            setLastEditedQueryPlatform: jest.fn(),
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
        <PolicyForm
          policyIdForEdit={mockPolicy.id}
          showOpenSchemaActionText={false}
          storedPolicy={createMockPolicy({ name: "" })}
          isStoredPolicyLoading={false}
          isTeamObserver={false}
          isUpdatingPolicy={false}
          onCreatePolicy={jest.fn()}
          onOsqueryTableSelect={jest.fn()}
          goToSelectTargets={jest.fn()}
          onUpdate={jest.fn()}
          onOpenSchemaSidebar={jest.fn()}
          renderLiveQueryWarning={jest.fn()}
          backendValidators={{}}
          onClickAutofillDescription={jest.fn()}
          onClickAutofillResolution={jest.fn()}
          isFetchingAutofillDescription={false}
          isFetchingAutofillResolution={false}
          resetAiAutofillData={jest.fn()}
        />
      );

      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    it("disables save and run button with tooltip for missing policy platforms", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          policy: {
            policyTeamId: undefined,
            lastEditedQueryId: mockPolicy.id,
            lastEditedQueryName: mockPolicy.name,
            lastEditedQueryDescription: mockPolicy.description,
            lastEditedQueryBody: mockPolicy.query,
            lastEditedQueryResolution: mockPolicy.resolution,
            lastEditedQueryCritical: mockPolicy.critical,
            lastEditedQueryPlatform: undefined, // missing policy platforms
            lastEditedQueryLabelsIncludeAny: [],
            lastEditedQueryLabelsExcludeAny: [],
            defaultPolicy: false,
            setLastEditedQueryName: jest.fn(),
            setLastEditedQueryDescription: jest.fn(),
            setLastEditedQueryBody: jest.fn(),
            setLastEditedQueryResolution: jest.fn(),
            setLastEditedQueryCritical: jest.fn(),
            setLastEditedQueryPlatform: jest.fn(),
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

      const { container, user } = render(
        <PolicyForm
          policyIdForEdit={mockPolicy.id}
          showOpenSchemaActionText={false}
          storedPolicy={createMockPolicy({ platform: undefined })}
          isStoredPolicyLoading={false}
          isTeamObserver={false}
          isUpdatingPolicy={false}
          onCreatePolicy={jest.fn()}
          onOsqueryTableSelect={jest.fn()}
          goToSelectTargets={jest.fn()}
          onUpdate={jest.fn()}
          onOpenSchemaSidebar={jest.fn()}
          renderLiveQueryWarning={jest.fn()}
          backendValidators={{}}
          onClickAutofillDescription={jest.fn()}
          onClickAutofillResolution={jest.fn()}
          isFetchingAutofillDescription={false}
          isFetchingAutofillResolution={false}
          resetAiAutofillData={jest.fn()}
        />
      );

      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
      expect(screen.getByRole("button", { name: "Run" })).toBeDisabled();

      await waitFor(() => {
        waitFor(() => {
          user.hover(screen.getByRole("button", { name: "Save" }));
        });

        expect(
          container.querySelector("#save-policy-button")
        ).toHaveTextContent(/to save or run the policy/i);
      });
    });

    it("disables run button with tooltip when live queries are globally disabled", async () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          policy: {
            policyTeamId: undefined,
            lastEditedQueryId: mockPolicy.id,
            lastEditedQueryName: mockPolicy.name,
            lastEditedQueryDescription: mockPolicy.description,
            lastEditedQueryBody: mockPolicy.query,
            lastEditedQueryResolution: mockPolicy.resolution,
            lastEditedQueryCritical: mockPolicy.critical,
            lastEditedQueryPlatform: undefined, // missing policy platforms
            lastEditedQueryLabelsIncludeAny: [],
            lastEditedQueryLabelsExcludeAny: [],
            defaultPolicy: false,
            setLastEditedQueryName: jest.fn(),
            setLastEditedQueryDescription: jest.fn(),
            setLastEditedQueryBody: jest.fn(),
            setLastEditedQueryResolution: jest.fn(),
            setLastEditedQueryCritical: jest.fn(),
            setLastEditedQueryPlatform: jest.fn(),
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

      const { user } = render(
        <PolicyForm
          policyIdForEdit={mockPolicy.id}
          showOpenSchemaActionText={false}
          storedPolicy={createMockPolicy()}
          isStoredPolicyLoading={false}
          isTeamObserver={false}
          isUpdatingPolicy={false}
          onCreatePolicy={jest.fn()}
          onOsqueryTableSelect={jest.fn()}
          goToSelectTargets={jest.fn()}
          onUpdate={jest.fn()}
          onOpenSchemaSidebar={jest.fn()}
          renderLiveQueryWarning={jest.fn()}
          backendValidators={{}}
          onClickAutofillDescription={jest.fn()}
          onClickAutofillResolution={jest.fn()}
          isFetchingAutofillDescription={false}
          isFetchingAutofillResolution={false}
          resetAiAutofillData={jest.fn()}
        />
      );

      expect(screen.getByRole("button", { name: "Run" })).toBeDisabled();

      await waitFor(() => {
        waitFor(() => {
          user.hover(screen.getByRole("button", { name: "Run" }));
        });

        expect(
          screen.getByText(/live queries are disabled/i)
        ).toBeInTheDocument();
      });
    });

    describe("target selector", () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
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
          policy: {
            policyTeamId: undefined,
            lastEditedQueryId: mockPolicy.id,
            lastEditedQueryName: "sumthin sumthin",
            lastEditedQueryDescription: mockPolicy.description,
            lastEditedQueryBody: mockPolicy.query,
            lastEditedQueryResolution: mockPolicy.resolution,
            lastEditedQueryCritical: mockPolicy.critical,
            lastEditedQueryPlatform: "linux",
            lastEditedQueryLabelsIncludeAny: [],
            lastEditedQueryLabelsExcludeAny: [],
            setLastEditedQueryName: jest.fn(),
            setLastEditedQueryDescription: jest.fn(),
            setLastEditedQueryBody: jest.fn(),
            setLastEditedQueryResolution: jest.fn(),
            setLastEditedQueryCritical: jest.fn(),
            setLastEditedQueryPlatform: jest.fn(),
          },
        },
      });

      it("should show the target selector in All hosts target mode when the query has no labels", async () => {
        render(<PolicyForm {...defaultProps} />);
        await waitFor(() => {
          expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
          expect(screen.getByLabelText("Custom")).toBeInTheDocument();
          expect(screen.getByLabelText("All hosts")).toBeChecked();
        });
      });

      it("should disable the save button in Custom target mode when no labels are selected, and enable it once labels are selected", async () => {
        render(<PolicyForm {...defaultProps} />);
        let allHosts;
        let custom;
        await waitFor(() => {
          allHosts = screen.getByLabelText("All hosts");
          custom = screen.getByLabelText("Custom");
          expect(allHosts).toBeInTheDocument();
          expect(custom).toBeInTheDocument();
        });
        custom && (await userEvent.click(custom));
        const saveButton = screen.getByRole("button", { name: "Save" });
        expect(saveButton).toBeDisabled();

        const funButton = screen.getByLabelText("Fun");
        expect(funButton).not.toBeChecked();
        await userEvent.click(funButton);
        await waitFor(() => {
          expect(saveButton).toBeEnabled();
        });
      });

      it("should send labels when saving a new query in Custom target mode (include any)", async () => {
        const onUpdate = jest.fn();
        const props = { ...defaultProps, onUpdate };
        render(<PolicyForm {...props} />);
        await waitFor(() => {
          expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
        });

        // Set a label.
        await userEvent.click(screen.getByLabelText("Custom"));
        await userEvent.click(screen.getByLabelText("Fun"));

        const saveButton = screen.getByRole("button", { name: "Save" });
        expect(saveButton).toBeEnabled();
        await userEvent.click(saveButton);

        expect(onUpdate.mock.calls[0][0].labels_include_any).toEqual(["Fun"]);
        expect(onUpdate.mock.calls[0][0].labels_exclude_any).toEqual([]);
      });

      it("should send labels when saving a new query in Custom target mode (exclude any)", async () => {
        const onUpdate = jest.fn();
        const props = { ...defaultProps, onUpdate };
        render(<PolicyForm {...props} />);
        await waitFor(() => {
          expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
        });

        // Set a label.
        await userEvent.click(screen.getByLabelText("Custom"));
        await userEvent.click(screen.getByLabelText("Fun"));

        // Click "Include any" to open the dropdown.
        const includeAnyOption = screen.getByRole("option", {
          name: "Include any",
        });
        await userEvent.click(includeAnyOption);

        // Click "Exclude any" to select it.
        let excludeAnyOption: unknown;
        await waitFor(() => {
          excludeAnyOption = screen.getByRole("option", {
            name: "Exclude any",
          });
        });
        await userEvent.click(excludeAnyOption as Element);

        const saveButton = screen.getByRole("button", { name: "Save" });
        expect(saveButton).toBeEnabled();
        await userEvent.click(saveButton);

        expect(onUpdate.mock.calls[0][0].labels_exclude_any).toEqual(["Fun"]);
        expect(onUpdate.mock.calls[0][0].labels_include_any).toEqual([]);
      });

      it("should clear labels when saving a new query in All hosts target mode", async () => {
        const onUpdate = jest.fn();
        const props = { ...defaultProps, onUpdate };
        render(<PolicyForm {...props} />);
        await waitFor(() => {
          expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
        });

        // Set a label.
        await userEvent.click(screen.getByLabelText("Custom"));
        await userEvent.click(screen.getByLabelText("Fun"));

        await userEvent.click(screen.getByLabelText("All hosts"));

        const saveButton = screen.getByRole("button", { name: "Save" });
        expect(saveButton).toBeEnabled();
        await userEvent.click(saveButton);

        expect(onUpdate.mock.calls[0][0].labels_include_any).toEqual([]);
        expect(onUpdate.mock.calls[0][0].labels_exclude_any).toEqual([]);
      });
    });
  });
  // TODO: Consider testing save button is disabled for a sql error
  // Trickiness is in modifying react-ace using react-testing library
});
