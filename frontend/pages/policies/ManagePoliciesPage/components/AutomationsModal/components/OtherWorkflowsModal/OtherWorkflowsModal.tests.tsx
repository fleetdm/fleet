import React from "react";
import { screen } from "@testing-library/react";
import { renderWithSetup, createMockRouter } from "test/test-utils";

import { IAutomationsConfig } from "interfaces/config";
import { IGlobalIntegrations } from "interfaces/integration";
import createMockConfig from "__mocks__/configMock";

import OtherWorkflowsModal from "./OtherWorkflowsModal";

const INVALID_URL_ERROR = "Destination URL is not a valid URL";
const REQUIRED_URL_ERROR = "Please add a destination URL";
const URL_PLACEHOLDER = "https://server.com/example";

const baseConfig = createMockConfig();

// Webhook automations enabled + empty URL so the Destination URL field renders
// and is editable.
const automationsConfig = ({
  webhook_settings: {
    ...baseConfig.webhook_settings,
    failing_policies_webhook: {
      ...baseConfig.webhook_settings.failing_policies_webhook,
      enable_failing_policies_webhook: true,
      destination_url: "",
    },
  },
  integrations: { jira: [], zendesk: [], google_calendar: null },
} as unknown) as IAutomationsConfig;

const availableIntegrations = ({
  jira: [],
  zendesk: [],
} as unknown) as IGlobalIntegrations;

const renderModal = () =>
  renderWithSetup(
    <OtherWorkflowsModal
      router={createMockRouter()}
      automationsConfig={automationsConfig}
      availableIntegrations={availableIntegrations}
    />
  );

describe("OtherWorkflowsModal - Destination URL validation", () => {
  it("does not show a validation error on open", () => {
    renderModal();

    expect(screen.getByPlaceholderText(URL_PLACEHOLDER)).toBeInTheDocument();
    expect(screen.queryByText(INVALID_URL_ERROR)).not.toBeInTheDocument();
    expect(screen.queryByText(REQUIRED_URL_ERROR)).not.toBeInTheDocument();
  });

  it("shows an error when blurred with an invalid URL", async () => {
    const { user } = renderModal();

    await user.type(
      screen.getByPlaceholderText(URL_PLACEHOLDER),
      "not-a-valid-url"
    );
    await user.tab();

    expect(await screen.findByText(INVALID_URL_ERROR)).toBeInTheDocument();
  });

  it("shows a required error when blurred while empty", async () => {
    const { user } = renderModal();

    await user.click(screen.getByPlaceholderText(URL_PLACEHOLDER));
    await user.tab();

    expect(await screen.findByText(REQUIRED_URL_ERROR)).toBeInTheDocument();
  });

  it("clears the error once the user edits the field", async () => {
    const { user } = renderModal();

    const urlInput = screen.getByPlaceholderText(URL_PLACEHOLDER);
    await user.type(urlInput, "not-a-valid-url");
    await user.tab();
    expect(await screen.findByText(INVALID_URL_ERROR)).toBeInTheDocument();

    await user.type(urlInput, "a");

    expect(screen.queryByText(INVALID_URL_ERROR)).not.toBeInTheDocument();
  });

  it("shows no error when blurred with a valid URL", async () => {
    const { user } = renderModal();

    await user.type(
      screen.getByPlaceholderText(URL_PLACEHOLDER),
      "https://example.com/webhook"
    );
    await user.tab();

    expect(screen.queryByText(INVALID_URL_ERROR)).not.toBeInTheDocument();
    expect(screen.queryByText(REQUIRED_URL_ERROR)).not.toBeInTheDocument();
  });
});
