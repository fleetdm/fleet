import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";
import userEvent from "@testing-library/user-event";

import createMockPolicy from "__mocks__/policyMock";
import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";
import { createMockTeamSummary } from "__mocks__/teamMock";

import { ILabelSummary } from "interfaces/label";
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
    router: createMockRouter(),
    teamIdForApi: 3,
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
    currentAutomatedPolicies: [],
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
            lastEditedQueryLabelsIncludeAll: [],
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
          router={createMockRouter()}
          teamIdForApi={3}
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
          currentAutomatedPolicies={[]}
        />
      );

      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    // Regression test for #38348: clicking Save with an empty query body must
    // not submit, even before the debounced SQL validator has populated
    // errors.query. The synchronous guard in promptSavePolicy enforces this.
    it("does not call onUpdate when Save is clicked with an empty query body", async () => {
      const onUpdate = jest.fn();
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          policy: {
            policyTeamId: undefined,
            lastEditedQueryId: mockPolicy.id,
            lastEditedQueryName: mockPolicy.name,
            lastEditedQueryDescription: mockPolicy.description,
            lastEditedQueryBody: "", // empty query body
            lastEditedQueryResolution: mockPolicy.resolution,
            lastEditedQueryCritical: mockPolicy.critical,
            lastEditedQueryPlatform: mockPolicy.platform,
            lastEditedQueryLabelsIncludeAny: [],
            lastEditedQueryLabelsIncludeAll: [],
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

      const { user } = render(
        <PolicyForm
          router={createMockRouter()}
          teamIdForApi={3}
          policyIdForEdit={mockPolicy.id}
          showOpenSchemaActionText={false}
          storedPolicy={createMockPolicy({ query: "" })}
          isStoredPolicyLoading={false}
          isTeamObserver={false}
          isUpdatingPolicy={false}
          onCreatePolicy={jest.fn()}
          onOsqueryTableSelect={jest.fn()}
          goToSelectTargets={jest.fn()}
          onUpdate={onUpdate}
          onOpenSchemaSidebar={jest.fn()}
          renderLiveQueryWarning={jest.fn()}
          backendValidators={{}}
          onClickAutofillDescription={jest.fn()}
          onClickAutofillResolution={jest.fn()}
          isFetchingAutofillDescription={false}
          isFetchingAutofillResolution={false}
          resetAiAutofillData={jest.fn()}
          currentAutomatedPolicies={[]}
        />
      );

      // On initial render `errors.query` is not yet set (the SQL validator is
      // debounced 500ms), so Save is enabled. This reproduces the race the
      // synchronous guard exists to handle: the user can click Save before
      // the debounce fires.
      const saveButton = screen.getByRole("button", { name: "Save" });
      expect(saveButton).toBeEnabled();
      await user.click(saveButton);

      expect(onUpdate).not.toHaveBeenCalled();
      // The synchronous guard sets errors.query = EMPTY_QUERY_ERR, which
      // SQLEditor surfaces as its label text.
      expect(
        screen.getByText("Query text must be present")
      ).toBeInTheDocument();
    });

    // Regression test for #38348: a policy with a non-empty but syntactically
    // invalid query must be savable. Only empty queries block Save.
    it("allows saving a policy whose query has a SQL syntax error", async () => {
      const onUpdate = jest.fn();
      const invalidSQL = "SELEKT * FROM bogus";
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          policy: {
            policyTeamId: undefined,
            lastEditedQueryId: mockPolicy.id,
            lastEditedQueryName: mockPolicy.name,
            lastEditedQueryDescription: mockPolicy.description,
            lastEditedQueryBody: invalidSQL,
            lastEditedQueryResolution: mockPolicy.resolution,
            lastEditedQueryCritical: mockPolicy.critical,
            lastEditedQueryPlatform: mockPolicy.platform,
            lastEditedQueryLabelsIncludeAny: [],
            lastEditedQueryLabelsIncludeAll: [],
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

      const { user } = render(
        <PolicyForm
          router={createMockRouter()}
          teamIdForApi={3}
          policyIdForEdit={mockPolicy.id}
          showOpenSchemaActionText={false}
          storedPolicy={createMockPolicy({ query: invalidSQL })}
          isStoredPolicyLoading={false}
          isTeamObserver={false}
          isUpdatingPolicy={false}
          onCreatePolicy={jest.fn()}
          onOsqueryTableSelect={jest.fn()}
          goToSelectTargets={jest.fn()}
          onUpdate={onUpdate}
          onOpenSchemaSidebar={jest.fn()}
          renderLiveQueryWarning={jest.fn()}
          backendValidators={{}}
          onClickAutofillDescription={jest.fn()}
          onClickAutofillResolution={jest.fn()}
          isFetchingAutofillDescription={false}
          isFetchingAutofillResolution={false}
          resetAiAutofillData={jest.fn()}
          currentAutomatedPolicies={[]}
        />
      );

      // Wait past the 500ms debounce so the SQL validator runs and flags the
      // syntax error. The error surfaces as SQLEditor's label text.
      await waitFor(() => {
        expect(
          screen.getByText("Syntax error. Please review before saving.")
        ).toBeInTheDocument();
      });

      const saveButton = screen.getByRole("button", { name: "Save" });
      expect(saveButton).toBeEnabled();
      await user.click(saveButton);

      expect(onUpdate).toHaveBeenCalledTimes(1);
      expect(onUpdate.mock.calls[0][0].query).toBe(invalidSQL);
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
            lastEditedQueryLabelsIncludeAll: [],
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

      const { user } = render(
        <PolicyForm
          router={createMockRouter()}
          teamIdForApi={3}
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
          currentAutomatedPolicies={[]}
        />
      );

      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
      expect(screen.getByRole("button", { name: "Run" })).toBeDisabled();
      await user.hover(screen.getByRole("button", { name: "Save" }));

      await waitFor(() => {
        expect(
          screen.getByText(/to save or run the policy/i)
        ).toBeInTheDocument();
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
            lastEditedQueryLabelsIncludeAll: [],
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
          router={createMockRouter()}
          teamIdForApi={3}
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
          currentAutomatedPolicies={[]}
        />
      );

      expect(screen.getByRole("button", { name: "Run" })).toBeDisabled();

      await waitFor(() => {
        waitFor(() => {
          user.hover(screen.getByRole("button", { name: "Run" }));
        });

        expect(
          screen.getByText(/live reports are disabled/i)
        ).toBeInTheDocument();
      });
    });

    describe("target selector", () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: createMockUser(),
            currentTeam: createMockTeamSummary(),
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
            lastEditedQueryLabelsIncludeAll: [],
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

        const funButton = await screen.findByRole("checkbox", {
          name: "Fun",
        });
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
        await userEvent.click(
          await screen.findByRole("checkbox", {
            name: "Fun",
          })
        );

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
        await userEvent.click(
          await screen.findByRole("checkbox", {
            name: "Fun",
          })
        );

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
        expect(onUpdate.mock.calls[0][0].labels_include_all).toEqual([]);
      });

      it("should set labels_include_all when picking the Include all option", async () => {
        const onUpdate = jest.fn();
        const props = { ...defaultProps, onUpdate };
        render(<PolicyForm {...props} />);
        await waitFor(() => {
          expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
        });

        await userEvent.click(screen.getByLabelText("Custom"));
        await userEvent.click(
          await screen.findByRole("checkbox", {
            name: "Fun",
          })
        );

        // Open the scope dropdown and pick "Include all".
        await userEvent.click(
          screen.getByRole("option", { name: "Include any" })
        );
        let includeAllOption: unknown;
        await waitFor(() => {
          includeAllOption = screen.getByRole("option", {
            name: "Include all",
          });
        });
        await userEvent.click(includeAllOption as Element);

        const saveButton = screen.getByRole("button", { name: "Save" });
        expect(saveButton).toBeEnabled();
        await userEvent.click(saveButton);

        expect(onUpdate.mock.calls[0][0].labels_include_all).toEqual(["Fun"]);
        expect(onUpdate.mock.calls[0][0].labels_include_any).toEqual([]);
        expect(onUpdate.mock.calls[0][0].labels_exclude_any).toEqual([]);
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
        await userEvent.click(
          await screen.findByRole("checkbox", {
            name: "Fun",
          })
        );

        await userEvent.click(screen.getByLabelText("All hosts"));

        const saveButton = screen.getByRole("button", { name: "Save" });
        expect(saveButton).toBeEnabled();
        await userEvent.click(saveButton);

        expect(onUpdate.mock.calls[0][0].labels_include_any).toEqual([]);
        expect(onUpdate.mock.calls[0][0].labels_exclude_any).toEqual([]);
        expect(onUpdate.mock.calls[0][0].labels_include_all).toEqual([]);
      });
    });

    describe("patch policy behavior", () => {
      const patchPolicy = createMockPolicy({
        type: "patch",
        platform: "darwin",
        patch_software: { name: "Firefox", software_title_id: 42 },
        install_software: undefined,
      });

      const patchPolicyProps = {
        ...defaultProps,
        storedPolicy: patchPolicy,
      };

      const renderPatchPolicy = createCustomRenderer({
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
            lastEditedQueryId: patchPolicy.id,
            lastEditedQueryName: patchPolicy.name,
            lastEditedQueryDescription: patchPolicy.description,
            lastEditedQueryBody: patchPolicy.query,
            lastEditedQueryResolution: patchPolicy.resolution,
            lastEditedQueryCritical: patchPolicy.critical,
            lastEditedQueryPlatform: patchPolicy.platform,
            lastEditedQueryLabelsIncludeAny: [],
            lastEditedQueryLabelsIncludeAll: [],
            lastEditedQueryLabelsExcludeAny: [],
            defaultPolicy: false,
            setLastEditedQueryName: jest.fn(),
            setLastEditedQueryDescription: jest.fn(),
            setLastEditedQueryBody: jest.fn(),
            setLastEditedQueryResolution: jest.fn(),
            setLastEditedQueryCritical: jest.fn(),
            setLastEditedQueryPlatform: jest.fn(),
          },
        },
      });

      it("hides platform selector", () => {
        renderPatchPolicy(<PolicyForm {...patchPolicyProps} />);
        expect(screen.queryByLabelText("macOS")).not.toBeInTheDocument();
        expect(screen.queryByLabelText("Windows")).not.toBeInTheDocument();
        expect(screen.queryByLabelText("Linux")).not.toBeInTheDocument();
      });

      it("hides target label selector", () => {
        renderPatchPolicy(<PolicyForm {...patchPolicyProps} />);
        expect(screen.queryByLabelText("All hosts")).not.toBeInTheDocument();
        expect(screen.queryByLabelText("Custom")).not.toBeInTheDocument();
      });

      it("submits only editable fields on save", async () => {
        const onUpdate = jest.fn();
        renderPatchPolicy(
          <PolicyForm {...patchPolicyProps} onUpdate={onUpdate} />
        );

        const saveButton = await screen.findByRole("button", { name: "Save" });
        await userEvent.click(saveButton);

        expect(onUpdate).toHaveBeenCalledTimes(1);
        const payload = onUpdate.mock.calls[0][0];
        expect(payload).toHaveProperty("name");
        expect(payload).toHaveProperty("description");
        expect(payload).toHaveProperty("resolution");
        expect(payload).toHaveProperty("critical");
        expect(payload).not.toHaveProperty("query");
        expect(payload).not.toHaveProperty("platform");
        expect(payload).not.toHaveProperty("labels_include_any");
      });

      it("shows 'Add automation' CTA when patch policy has no install_software", async () => {
        renderPatchPolicy(<PolicyForm {...patchPolicyProps} />);
        await waitFor(() => {
          expect(
            screen.getByText(/Automatically patch Firefox/)
          ).toBeInTheDocument();
          expect(screen.getByText(/Add automation/)).toBeInTheDocument();
        });
      });

      it("hides 'Add automation' CTA when automation already exists", async () => {
        const automatedPatchPolicy = createMockPolicy({
          type: "patch",
          platform: "darwin",
          patch_software: { name: "Firefox", software_title_id: 42 },
          install_software: { name: "Firefox", software_title_id: 42 },
        });
        renderPatchPolicy(
          <PolicyForm
            {...patchPolicyProps}
            storedPolicy={automatedPatchPolicy}
          />
        );

        // Wait for the component to fully render, then assert CTA is absent
        await waitFor(() => {
          expect(
            screen.getByRole("button", { name: "Save" })
          ).toBeInTheDocument();
        });
        expect(screen.queryByText(/Add automation/)).not.toBeInTheDocument();
      });
    });
  });

  describe("renderPolicyFleetName", () => {
    it("does not render anything on free tier", () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: createMockUser(),
            currentTeam: createMockTeamSummary(),
            isGlobalObserver: false,
            isGlobalAdmin: true,
            isGlobalMaintainer: false,
            isOnGlobalTeam: true,
            isPremiumTier: false,
            isSandboxMode: false,
            isFreeTier: true,
            config: createMockConfig(),
          },
          policy: {
            policyTeamId: undefined,
            lastEditedQueryId: mockPolicy.id,
            lastEditedQueryName: mockPolicy.name,
            lastEditedQueryDescription: mockPolicy.description,
            lastEditedQueryBody: mockPolicy.query,
            lastEditedQueryResolution: mockPolicy.resolution,
            lastEditedQueryCritical: mockPolicy.critical,
            lastEditedQueryPlatform: mockPolicy.platform,
            lastEditedQueryLabelsIncludeAny: [],
            lastEditedQueryLabelsIncludeAll: [],
            lastEditedQueryLabelsExcludeAny: [],
            defaultPolicy: false,
            setLastEditedQueryName: jest.fn(),
            setLastEditedQueryDescription: jest.fn(),
            setLastEditedQueryBody: jest.fn(),
            setLastEditedQueryResolution: jest.fn(),
            setLastEditedQueryCritical: jest.fn(),
            setLastEditedQueryPlatform: jest.fn(),
          },
        },
      });

      render(<PolicyForm {...defaultProps} />);

      expect(screen.queryByText(/policy for/i)).not.toBeInTheDocument();
    });

    it("shows 'Editing policy' when existing policy and user has save permissions", () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: createMockUser(),
            currentTeam: createMockTeamSummary(),
            isGlobalObserver: false,
            isGlobalAdmin: true, // has save perms
            isGlobalMaintainer: false,
            isTeamMaintainerOrTeamAdmin: false,
            isOnGlobalTeam: true,
            isPremiumTier: true,
            isSandboxMode: false,
            isFreeTier: false,
            config: createMockConfig(),
          },
          policy: {
            policyTeamId: undefined,
            lastEditedQueryId: mockPolicy.id,
            lastEditedQueryName: mockPolicy.name,
            lastEditedQueryDescription: mockPolicy.description,
            lastEditedQueryBody: mockPolicy.query,
            lastEditedQueryResolution: mockPolicy.resolution,
            lastEditedQueryCritical: mockPolicy.critical,
            lastEditedQueryPlatform: mockPolicy.platform,
            lastEditedQueryLabelsIncludeAny: [],
            lastEditedQueryLabelsIncludeAll: [],
            lastEditedQueryLabelsExcludeAny: [],
            defaultPolicy: false,
            setLastEditedQueryName: jest.fn(),
            setLastEditedQueryDescription: jest.fn(),
            setLastEditedQueryBody: jest.fn(),
            setLastEditedQueryResolution: jest.fn(),
            setLastEditedQueryCritical: jest.fn(),
            setLastEditedQueryPlatform: jest.fn(),
          },
        },
      });

      render(<PolicyForm {...defaultProps} />);

      expect(screen.getByText(/Editing policy for/i)).toBeInTheDocument();
      expect(
        screen.getByText(createMockTeamSummary().name)
      ).toBeInTheDocument();
    });

    it("shows 'Creating a new policy' when there is no existing policy", () => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            currentUser: createMockUser(),
            currentTeam: createMockTeamSummary(),
            isGlobalObserver: false,
            isGlobalAdmin: true,
            isGlobalMaintainer: false,
            isTeamMaintainerOrTeamAdmin: false,
            isOnGlobalTeam: true,
            isPremiumTier: true,
            isSandboxMode: false,
            isFreeTier: false,
            config: createMockConfig(),
          },
          policy: {
            policyTeamId: undefined,
            lastEditedQueryId: null,
            lastEditedQueryName: "",
            lastEditedQueryDescription: "",
            lastEditedQueryBody: "",
            lastEditedQueryResolution: "",
            lastEditedQueryCritical: false,
            lastEditedQueryPlatform: undefined,
            lastEditedQueryLabelsIncludeAny: [],
            lastEditedQueryLabelsIncludeAll: [],
            lastEditedQueryLabelsExcludeAny: [],
            defaultPolicy: false,
            setLastEditedQueryName: jest.fn(),
            setLastEditedQueryDescription: jest.fn(),
            setLastEditedQueryBody: jest.fn(),
            setLastEditedQueryResolution: jest.fn(),
            setLastEditedQueryCritical: jest.fn(),
            setLastEditedQueryPlatform: jest.fn(),
          },
        },
      });

      render(
        <PolicyForm
          {...defaultProps}
          policyIdForEdit={null}
          storedPolicy={undefined}
        />
      );

      expect(
        screen.getByText(/Creating a new policy for/i)
      ).toBeInTheDocument();
      expect(
        screen.getByText(createMockTeamSummary().name)
      ).toBeInTheDocument();
    });
  });
});
