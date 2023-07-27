import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockPolicy from "__mocks__/policyMock";

import PolicyForm from "./PolicyForm";
import createMockUser from "__mocks__/userMock";

const [validEmail, invalidEmail] = ["hi@thegnar.co", "invalid-email"];
const password = "p@ssw0rd";

const mockPolicy = createMockPolicy();

describe("PolicyForm - component", () => {
  it("disables save button for invalid sql", async () => {
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

    const { user } = render(
      <PolicyForm
        policyIdForEdit={mockPolicy.id}
        showOpenSchemaActionText={false}
        storedPolicy={mockPolicy}
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

    await user.click(screen.getByTestId("user-menu"));
  });
});
