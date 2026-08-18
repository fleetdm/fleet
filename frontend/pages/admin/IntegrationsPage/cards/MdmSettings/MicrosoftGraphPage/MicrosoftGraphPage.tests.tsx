import React from "react";
import { screen, waitFor, within } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import { createMockConfig } from "__mocks__/configMock";
import { IMicrosoftGraphCredential } from "interfaces/microsoft_graph_credential";
import microsoftGraphCredentialsAPI from "services/entities/microsoft_graph_credentials";

import MicrosoftGraphPage from "./MicrosoftGraphPage";

jest.mock("services/entities/microsoft_graph_credentials");
jest.mock("components/ToastNotification", () => ({
  notify: { success: jest.fn(), error: jest.fn() },
}));

const mockedAPI = microsoftGraphCredentialsAPI as jest.Mocked<
  typeof microsoftGraphCredentialsAPI
>;

const createMockCredential = (
  overrides: Partial<IMicrosoftGraphCredential> = {}
): IMicrosoftGraphCredential => ({
  tenant_id: "tenant-guid",
  client_id: "client-guid",
  credential_invalid: false,
  last_synced_at: null,
  last_sync_error: null,
  ...overrides,
});

const renderPage = (
  credentials: IMicrosoftGraphCredential[] = [],
  { isPremiumTier = true, gitOpsModeEnabled = false } = {}
) => {
  mockedAPI.getCredentials.mockResolvedValue({
    microsoft_graph_credentials: credentials,
  });

  const config = createMockConfig();
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier,
        config: {
          ...config,
          gitops: { ...config.gitops, gitops_mode_enabled: gitOpsModeEnabled },
        },
      },
    },
  });

  return render(<MicrosoftGraphPage />);
};

describe("MicrosoftGraphPage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedAPI.applyCredentials.mockResolvedValue(undefined);
    mockedAPI.deleteCredentials.mockResolvedValue(undefined);
  });

  it("paywalls the credential fields on Fleet Free", async () => {
    renderPage([], { isPremiumTier: false });

    expect(
      await screen.findByText("This feature is included in Fleet Premium.")
    ).toBeInTheDocument();
    expect(screen.queryByLabelText("Tenant ID")).not.toBeInTheDocument();
    // The endpoint is premium-only, so Free should not even ask.
    expect(mockedAPI.getCredentials).not.toHaveBeenCalled();
  });

  it("renders empty fields and no delete action when no credential is stored", async () => {
    renderPage([]);

    expect(await screen.findByLabelText("Tenant ID")).toHaveValue("");
    expect(screen.getByLabelText("Client ID")).toHaveValue("");
    expect(screen.getByLabelText("Client secret")).toHaveValue("");
    expect(
      screen.queryByRole("button", { name: "Delete" })
    ).not.toBeInTheDocument();
  });

  it("seeds the form from a stored credential and masks the secret", async () => {
    renderPage([createMockCredential()]);

    await waitFor(() => {
      expect(screen.getByLabelText("Tenant ID")).toHaveValue("tenant-guid");
    });
    expect(screen.getByLabelText("Client ID")).toHaveValue("client-guid");
    // The API never returns the secret; the mask signals that one is stored.
    expect(screen.getByLabelText("Client secret")).toHaveValue("********");
    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
  });

  it("renders sync status, including the last error", async () => {
    renderPage([
      createMockCredential({
        last_synced_at: "2026-08-14T12:00:00Z",
        last_sync_error: "AADSTS7000215: Invalid client secret provided.",
      }),
    ]);

    expect(await screen.findByText("Last synced")).toBeInTheDocument();
    expect(
      screen.getByText("AADSTS7000215: Invalid client secret provided.")
    ).toBeInTheDocument();
  });

  it("reports never synced when the credential has not synced yet", async () => {
    renderPage([createMockCredential()]);

    expect(await screen.findByText("Last synced")).toBeInTheDocument();
    expect(screen.getByText("Never")).toBeInTheDocument();
  });

  it("keeps the stored secret when it is not changed", async () => {
    const { user } = renderPage([createMockCredential()]);

    await screen.findByLabelText("Tenant ID");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(mockedAPI.applyCredentials).toHaveBeenCalledWith([
        { tenant_id: "tenant-guid", client_id: "client-guid" },
      ]);
    });
  });

  it("sends the secret once the admin changes it", async () => {
    const { user } = renderPage([createMockCredential()]);

    const secretField = await screen.findByLabelText("Client secret");
    await user.clear(secretField);
    await user.type(secretField, "new-secret");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(mockedAPI.applyCredentials).toHaveBeenCalledWith([
        {
          tenant_id: "tenant-guid",
          client_id: "client-guid",
          client_secret: "new-secret",
        },
      ]);
    });
  });

  it("validates on save rather than disabling the button", async () => {
    const { user } = renderPage([]);

    const saveButton = await screen.findByRole("button", { name: "Save" });
    expect(saveButton).toBeEnabled();

    await user.click(saveButton);

    expect(await screen.findByText("Enter a tenant ID")).toBeInTheDocument();
    expect(screen.getByText("Enter a client ID")).toBeInTheDocument();
    expect(screen.getByText("Enter a client secret")).toBeInTheDocument();
    expect(mockedAPI.applyCredentials).not.toHaveBeenCalled();
  });

  it("clears a field error when the field is focused", async () => {
    const { user } = renderPage([]);

    await user.click(await screen.findByRole("button", { name: "Save" }));
    expect(await screen.findByText("Enter a tenant ID")).toBeInTheDocument();

    // The error replaces the label, so target the input directly.
    await user.click(
      document.querySelector("input[name='tenantId']") as HTMLElement
    );
    await waitFor(() => {
      expect(screen.queryByText("Enter a tenant ID")).not.toBeInTheDocument();
    });
  });

  it("deletes the credential by sending an empty list", async () => {
    const { user } = renderPage([createMockCredential()]);

    await user.click(await screen.findByRole("button", { name: "Delete" }));

    // Confirm inside the modal, which has its own Delete button. Fleet's Modal sets no dialog role, so scope by class.
    const modal = await waitFor(() => {
      const el = document.querySelector(
        ".delete-microsoft-graph-credential-modal"
      );
      if (!el) throw new Error("delete modal not rendered");
      return el as HTMLElement;
    });
    await user.click(within(modal).getByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(mockedAPI.deleteCredentials).toHaveBeenCalled();
    });
  });

  it("disables the fields and save in GitOps mode", async () => {
    renderPage([createMockCredential()], { gitOpsModeEnabled: true });

    expect(await screen.findByLabelText("Tenant ID")).toBeDisabled();
    expect(screen.getByLabelText("Client ID")).toBeDisabled();
    expect(screen.getByLabelText("Client secret")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });
});
