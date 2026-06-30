import React from "react";
import { screen } from "@testing-library/react";

import { createMockConfig } from "__mocks__/configMock";
import { createCustomRenderer } from "test/test-utils";

import GoogleWorkspaceSection from "./GoogleWorkspaceSection";

// Premium gating now lives in the parent IdentityProviders component, so this
// section always renders its form.
const renderWith = () =>
  createCustomRenderer({
    withBackendMock: true,
    context: { app: { config: createMockConfig() } },
  });

describe("GoogleWorkspaceSection", () => {
  it("renders the connect form", () => {
    const render = renderWith();
    render(<GoogleWorkspaceSection appConfig={createMockConfig()} />);

    expect(screen.getByText("Google Workspace")).toBeInTheDocument();
    expect(screen.getByLabelText("API key JSON")).toBeInTheDocument();
    expect(screen.getByLabelText("Primary domain")).toBeInTheDocument();
    expect(
      screen.getByLabelText("Admin email to impersonate")
    ).toBeInTheDocument();
    // Mutual-exclusion messaging.
    expect(screen.getByText(/SCIM provisioning/i)).toBeInTheDocument();
  });

  it("pre-fills the form and masks the API key from existing config", () => {
    const config = createMockConfig();
    config.integrations.google_workspace = [
      {
        domain: "example.com",
        impersonated_user_email: "admin@example.com",
        api_key_json: { client_email: "********", private_key: "********" },
      },
    ];

    const render = renderWith();
    render(<GoogleWorkspaceSection appConfig={config} />);

    expect(screen.getByDisplayValue("example.com")).toBeInTheDocument();
    expect(screen.getByDisplayValue("admin@example.com")).toBeInTheDocument();
    // Masked key shown as the unchanged-password placeholder.
    expect(screen.getByDisplayValue("********")).toBeInTheDocument();
  });
});
