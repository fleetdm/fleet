import React from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mockServer from "test/mock-server";
import { createGetConfigHandler } from "test/handlers/config-handlers";

import createMockConfig from "__mocks__/configMock";

import ManageSoftwareAutomationsModal from "./ManageSoftwareAutomationsModal";

const INVALID_URL_ERROR = "Destination URL is not a valid URL";
const REQUIRED_URL_ERROR = "Please add a destination URL";
const URL_PLACEHOLDER = "https://server.com/example";

// Start with the webhook workflow already enabled so the Destination URL field
// is rendered and editable, letting us exercise the on-blur validation directly.
const softwareConfig = createMockConfig({
  webhook_settings: {
    vulnerabilities_webhook: {
      enable_vulnerabilities_webhook: true,
      destination_url: "",
    },
  },
  integrations: { jira: [], zendesk: [] },
});

const defaultProps = {
  router: createMockRouter(),
  onCancel: jest.fn(),
  onCreateWebhookSubmit: jest.fn(),
  togglePreviewPayloadModal: jest.fn(),
  togglePreviewTicketModal: jest.fn(),
  showPreviewPayloadModal: false,
  showPreviewTicketModal: false,
  softwareConfig,
};

const renderModal = ({ gitOpsModeEnabled = false } = {}) => {
  mockServer.use(createGetConfigHandler());
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        config: createMockConfig({
          gitops: { gitops_mode_enabled: gitOpsModeEnabled },
        }),
        isFreeTier: false,
      },
    },
  });
  return render(<ManageSoftwareAutomationsModal {...defaultProps} />);
};

describe("ManageSoftwareAutomationsModal - Destination URL validation", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("does not show a validation error while typing an invalid URL", async () => {
    const { user } = renderModal();

    const urlInput = screen.getByPlaceholderText(URL_PLACEHOLDER);
    await user.type(urlInput, "not-a-valid-url");

    expect(screen.queryByText(INVALID_URL_ERROR)).not.toBeInTheDocument();
  });

  it("shows an error when the field is blurred with an invalid URL", async () => {
    const { user } = renderModal();

    const urlInput = screen.getByPlaceholderText(URL_PLACEHOLDER);
    await user.type(urlInput, "not-a-valid-url");
    await user.tab();

    expect(await screen.findByText(INVALID_URL_ERROR)).toBeInTheDocument();
  });

  it("clears the error once the user edits the field again", async () => {
    const { user } = renderModal();

    const urlInput = screen.getByPlaceholderText(URL_PLACEHOLDER);
    await user.type(urlInput, "not-a-valid-url");
    await user.tab();
    expect(await screen.findByText(INVALID_URL_ERROR)).toBeInTheDocument();

    await user.type(urlInput, "a");

    await waitFor(() => {
      expect(screen.queryByText(INVALID_URL_ERROR)).not.toBeInTheDocument();
    });
  });

  it("shows no error when the field is blurred with a valid URL", async () => {
    const { user } = renderModal();

    const urlInput = screen.getByPlaceholderText(URL_PLACEHOLDER);
    await user.type(urlInput, "https://example.com/webhook");
    await user.tab();

    expect(screen.queryByText(INVALID_URL_ERROR)).not.toBeInTheDocument();
    expect(screen.queryByText(REQUIRED_URL_ERROR)).not.toBeInTheDocument();
  });

  it("shows a required error when the field is blurred while empty", async () => {
    const { user } = renderModal();

    const urlInput = screen.getByPlaceholderText(URL_PLACEHOLDER);
    await user.click(urlInput);
    await user.tab();

    expect(await screen.findByText(REQUIRED_URL_ERROR)).toBeInTheDocument();
  });

  it("does not validate on blur when GitOps mode disables the field", () => {
    renderModal({ gitOpsModeEnabled: true });

    const urlInput = screen.getByPlaceholderText(URL_PLACEHOLDER);
    expect(urlInput).toBeDisabled();

    // The field is read-only in GitOps mode, so a blur must not surface an error.
    fireEvent.blur(urlInput);

    expect(screen.queryByText(REQUIRED_URL_ERROR)).not.toBeInTheDocument();
    expect(screen.queryByText(INVALID_URL_ERROR)).not.toBeInTheDocument();
  });
});
