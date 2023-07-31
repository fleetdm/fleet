import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockPolicy from "__mocks__/policyMock";
import createMockUser from "__mocks__/userMock";

import PolicyForm from "./PolicyForm";

const mockPolicy = createMockPolicy();

describe("PolicyForm - component", () => {
  it("disables save button for missing policy name", async () => {
    const render = createCustomRenderer({
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
        },
      },
    });

    render(
      <PolicyForm
        policyIdForEdit={mockPolicy.id}
        showOpenSchemaActionText={false}
        storedPolicy={createMockPolicy({ name: "" })}
        isStoredPolicyLoading={false}
        isTeamAdmin={false}
        isTeamMaintainer={false}
        isTeamObserver={false}
        isUpdatingPolicy={false}
        onCreatePolicy={jest.fn()}
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

  it("disables save and run button for missing policy platforms", async () => {
    const render = createCustomRenderer({
      context: {
        policy: {
          policyTeamId: undefined,
          lastEditedQueryId: mockPolicy.id,
          lastEditedQueryName: mockPolicy.name,
          lastEditedQueryDescription: mockPolicy.description,
          lastEditedQueryBody: mockPolicy.query,
          lastEditedQueryResolution: mockPolicy.resolution,
          lastEditedQueryCritical: mockPolicy.critical,
          lastEditedQueryPlatform: "", // missing policy platforms
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
        },
      },
    });

    const { user } = render(
      <PolicyForm
        policyIdForEdit={mockPolicy.id}
        showOpenSchemaActionText={false}
        storedPolicy={createMockPolicy({ platform: "" })}
        isStoredPolicyLoading={false}
        isTeamAdmin={false}
        isTeamMaintainer={false}
        isTeamObserver={false}
        isUpdatingPolicy={false}
        onCreatePolicy={jest.fn()}
        onOsqueryTableSelect={jest.fn()}
        goToSelectTargets={jest.fn()}
        onUpdate={jest.fn()}
        onOpenSchemaSidebar={jest.fn()}
        renderLiveQueryWarning={jest.fn()}
        backendValidators={{}}
      />
    );

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Run" })).toBeDisabled();

    await user.hover(screen.getByRole("button", { name: "Save" }));

    expect(screen.findByText(/to save or run the policy/g)).toBeInTheDocument(); // Line of tooltip
  });

  it("disables save for sql error", async () => {
    const render = createCustomRenderer({
      context: {
        policy: {
          policyTeamId: undefined,
          lastEditedQueryId: mockPolicy.id,
          lastEditedQueryName: mockPolicy.name,
          lastEditedQueryDescription: mockPolicy.description,
          lastEditedQueryBody: "select ** from users;",
          lastEditedQueryResolution: mockPolicy.resolution,
          lastEditedQueryCritical: mockPolicy.critical,
          lastEditedQueryPlatform: "", // missing policy platforms
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
        },
      },
    });

    render(
      <PolicyForm
        policyIdForEdit={mockPolicy.id}
        showOpenSchemaActionText={false}
        storedPolicy={createMockPolicy({ query: "select ** from users;" })} // sql error
        isStoredPolicyLoading={false}
        isTeamAdmin={false}
        isTeamMaintainer={false}
        isTeamObserver={false}
        isUpdatingPolicy={false}
        onCreatePolicy={jest.fn()}
        onOsqueryTableSelect={jest.fn()}
        goToSelectTargets={jest.fn()}
        onUpdate={jest.fn()}
        onOpenSchemaSidebar={jest.fn()}
        renderLiveQueryWarning={jest.fn()}
        backendValidators={{}}
      />
    );

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Run" })).toBeDisabled();
  });
});

// await user.hover(screen.getByRole("button", { name: "Save" }));

// expect(
//   await screen.findByText("Password must be present")
// ).toBeInTheDocument();
