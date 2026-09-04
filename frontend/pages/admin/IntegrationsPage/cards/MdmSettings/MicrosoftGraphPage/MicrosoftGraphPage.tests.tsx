import React from "react";
import { screen, waitFor, within } from "@testing-library/react";

import { createCustomRenderer, getOpenModal } from "test/test-utils";
import { createMockConfig } from "__mocks__/configMock";
import { IMicrosoftGraphCredential } from "interfaces/microsoft_graph_credential";
import microsoftGraphCredentialsAPI from "services/entities/microsoft_graph_credentials";

import { notify } from "components/ToastNotification";

import MicrosoftGraphPage from "./MicrosoftGraphPage";

jest.mock("services/entities/microsoft_graph_credentials");
jest.mock("components/ToastNotification", () => ({
  notify: { success: jest.fn(), error: jest.fn(), batch: jest.fn() },
}));

const mockedAPI = microsoftGraphCredentialsAPI as jest.Mocked<
  typeof microsoftGraphCredentialsAPI
>;

const createMockCredential = (
  overrides: Partial<IMicrosoftGraphCredential> = {}
): IMicrosoftGraphCredential => ({
  tenant_id: "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4",
  client_id: "7f6b1665-51f5-48de-a9b6-ac17539583fb",
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
      expect(screen.getByLabelText("Tenant ID")).toHaveValue(
        "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4"
      );
    });
    expect(screen.getByLabelText("Client ID")).toHaveValue(
      "7f6b1665-51f5-48de-a9b6-ac17539583fb"
    );
    // The API never returns the secret; the mask signals that one is stored.
    const secretField = screen.getByLabelText("Client secret");
    expect(secretField).toHaveValue("********");
    expect(secretField).toHaveAttribute("autocomplete", "new-password");
    expect(secretField).toHaveAttribute("data-1p-ignore");
    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
  });

  it("renders the last sync error in a tooltip on the error indicator", async () => {
    const { user } = renderPage([
      createMockCredential({
        last_synced_at: "2026-08-14T12:00:00Z",
        last_sync_error: "Microsoft Graph is temporarily unavailable (503).",
      }),
    ]);

    expect(await screen.findByText("Last synced:")).toBeInTheDocument();
    await user.hover(screen.getByTestId("error-icon"));

    // Scoped to the tooltip.
    await waitFor(() => {
      expect(
        within(screen.getByRole("tooltip")).getByText(
          "Microsoft Graph is temporarily unavailable (503)."
        )
      ).toBeInTheDocument();
    });
  });

  it("exposes the sync error to screen readers without a hover", async () => {
    renderPage([
      createMockCredential({
        last_synced_at: "2026-08-14T12:00:00Z",
        last_sync_error: "Microsoft Graph is temporarily unavailable (503).",
      }),
    ]);

    // The message is visible only on hover, which a screen reader cannot perform, so it is also rendered as
    // visually-hidden text that stays in the accessibility tree.
    expect(
      await screen.findByText(
        "Microsoft Graph is temporarily unavailable (503)."
      )
    ).toBeInTheDocument();
  });

  it("hides the error indicator when the credential was rejected", async () => {
    renderPage([
      createMockCredential({
        credential_invalid: true,
        last_synced_at: "2026-08-14T12:00:00Z",
        last_sync_error:
          "Microsoft Graph rejected the credential. Check the tenant ID, client ID, and client secret.",
      }),
    ]);

    // A rejected credential already raises the app-wide banner, so repeating it here would be the same news twice.
    expect(await screen.findByText("Last synced:")).toBeInTheDocument();
    expect(screen.queryByTestId("error-icon")).not.toBeInTheDocument();
  });

  it("reports never synced when the credential has not synced yet", async () => {
    renderPage([createMockCredential()]);

    expect(await screen.findByText("Last synced:")).toBeInTheDocument();
    expect(screen.getByText("Never")).toBeInTheDocument();
  });

  it("keeps the stored secret when it is not changed", async () => {
    const { user } = renderPage([createMockCredential()]);

    await screen.findByLabelText("Tenant ID");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(mockedAPI.applyCredentials).toHaveBeenCalledWith([
        {
          tenant_id: "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4",
          client_id: "7f6b1665-51f5-48de-a9b6-ac17539583fb",
        },
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
          tenant_id: "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4",
          client_id: "7f6b1665-51f5-48de-a9b6-ac17539583fb",
          client_secret: "new-secret",
        },
      ]);
    });
  });

  it("requires a fresh secret when the client ID changes", async () => {
    const { user } = renderPage([createMockCredential()]);

    const clientIdField = await screen.findByLabelText("Client ID");
    await user.clear(clientIdField);
    await user.type(clientIdField, "9a2e4d10-3b77-4c58-8e21-1f0c5d6a7b88");

    // The stored secret belongs to the old app registration, so the mask is cleared and re-entry is required.
    await waitFor(() => {
      expect(screen.getByLabelText("Client secret")).toHaveValue("");
    });

    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(
      await screen.findByText("Enter a client secret")
    ).toBeInTheDocument();
    expect(mockedAPI.applyCredentials).not.toHaveBeenCalled();
  });

  it("rejects IDs that are not in Entra's GUID format", async () => {
    const { user } = renderPage([]);

    await user.type(await screen.findByLabelText("Tenant ID"), "not-a-guid");
    await user.type(screen.getByLabelText("Client ID"), "also-not-a-guid");
    await user.type(screen.getByLabelText("Client secret"), "a-secret");
    await user.click(screen.getByRole("button", { name: "Save" }));

    // The API rejects a malformed GUID with a 422, so catching it inline avoids the round trip.
    expect(
      await screen.findByText("Enter a tenant ID in GUID format")
    ).toBeInTheDocument();
    expect(
      screen.getByText("Enter a client ID in GUID format")
    ).toBeInTheDocument();
    expect(mockedAPI.applyCredentials).not.toHaveBeenCalled();
  });

  it("treats a differently-cased ID as the same app registration", async () => {
    const { user } = renderPage([createMockCredential()]);

    const clientIdField = await screen.findByLabelText("Client ID");
    await user.clear(clientIdField);
    await user.type(clientIdField, "7F6B1665-51F5-48DE-A9B6-AC17539583FB");

    // The API lower-cases both IDs before comparing, so re-pasting the same ID in upper case is not an identity
    // change and must not force the admin to re-enter a secret that did not change.
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(screen.queryByText("Enter a client secret")).not.toBeInTheDocument();
    await waitFor(() => {
      expect(mockedAPI.applyCredentials).toHaveBeenCalledWith([
        {
          tenant_id: "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4",
          client_id: "7F6B1665-51F5-48DE-A9B6-AC17539583FB",
        },
      ]);
    });
  });

  it("keeps typed input when a background refetch only moves sync state", async () => {
    const { user } = renderPage([createMockCredential()]);

    const tenantField = await screen.findByLabelText("Tenant ID");
    await user.clear(tenantField);
    await user.type(tenantField, "11111111-2222-3333-4444-555555555555");

    // The sync cron rewrites last_synced_at every 5 minutes, and refetchOnWindowFocus is on, so returning to the tab
    // hands the form a new credential object. Re-seeding on that would discard what the admin is typing.
    mockedAPI.getCredentials.mockResolvedValue({
      microsoft_graph_credentials: [
        createMockCredential({ last_synced_at: "2026-08-19T16:10:37Z" }),
      ],
    });
    window.dispatchEvent(new Event("focus"));

    await waitFor(() => {
      expect(screen.getByText("Last synced:")).toBeInTheDocument();
    });
    expect(screen.getByLabelText("Tenant ID")).toHaveValue(
      "11111111-2222-3333-4444-555555555555"
    );
  });

  it("shows a field error on blur once the field is dirty", async () => {
    const { user } = renderPage([]);

    const tenantField = await screen.findByLabelText("Tenant ID");
    await user.type(tenantField, "not-a-guid");
    await user.tab();

    expect(
      await screen.findByText("Enter a tenant ID in GUID format")
    ).toBeInTheDocument();
    // Blur validates the blurred field only.
    expect(screen.queryByText("Enter a client ID")).not.toBeInTheDocument();
  });

  it("stays silent on blur of a field the admin never typed in", async () => {
    const { user } = renderPage([]);

    await user.click(await screen.findByLabelText("Tenant ID"));
    await user.tab();
    await user.tab();

    expect(screen.queryByText("Enter a tenant ID")).not.toBeInTheDocument();
    expect(screen.queryByText("Enter a client ID")).not.toBeInTheDocument();
  });

  it("renders a server field error inline and toasts it", async () => {
    mockedAPI.applyCredentials.mockRejectedValue({
      data: {
        errors: [
          {
            name: "microsoft_graph_credentials.client_secret",
            reason: "Microsoft Graph rejected the credential.",
          },
        ],
      },
    });
    const { user } = renderPage([createMockCredential()]);

    const secretField = await screen.findByLabelText("Client secret");
    await user.clear(secretField);
    await user.type(secretField, "wrong-secret");
    await user.click(screen.getByRole("button", { name: "Save" }));

    // The field can be scrolled off-screen after submit, so the error surfaces inline and as a toast.
    expect(
      await screen.findByText("Microsoft Graph rejected the credential.")
    ).toBeInTheDocument();
    expect(notify.batch).toHaveBeenCalledWith([
      { variant: "error", message: "Microsoft Graph rejected the credential." },
    ]);
  });

  // A whole-credential failure (verification, licensing, missing private key) is reported without a per-field `name`, so
  // it can't attach to a field. The toast then shows a static main line and puts the full server response in the
  // expandable panel, so a long reason (e.g. AADSTS) doesn't render on both lines.
  it("toasts the whole-credential failure with a static main line and the full response in the panel", async () => {
    const rejection = {
      data: {
        errors: [
          {
            name: "microsoft_graph_credentials",
            reason:
              "Microsoft Graph returned an error (400): AADSTS90002: Tenant not found.",
          },
        ],
      },
    };
    mockedAPI.applyCredentials.mockRejectedValue(rejection);
    const { user } = renderPage([createMockCredential()]);

    const secretField = await screen.findByLabelText("Client secret");
    await user.clear(secretField);
    await user.type(secretField, "some-secret");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledWith(
        "Couldn't save Microsoft Graph credential.",
        {
          response: rejection,
        }
      );
    });
    // A whole-credential failure has no field, so the per-field batch toast path stays quiet.
    expect(notify.batch).not.toHaveBeenCalled();
  });

  it("clears the secret error when the identity change that required it is reverted", async () => {
    const { user } = renderPage([createMockCredential()]);

    const clientIdField = await screen.findByLabelText("Client ID");
    await user.clear(clientIdField);
    await user.type(clientIdField, "9a2e4d10-3b77-4c58-8e21-1f0c5d6a7b88");
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(
      await screen.findByText("Enter a client secret")
    ).toBeInTheDocument();

    // Putting the stored client ID back means the app registration is unchanged, so the secret is no longer required
    // and its error no longer applies.
    await user.clear(clientIdField);
    await user.type(clientIdField, "7f6b1665-51f5-48de-a9b6-ac17539583fb");

    await waitFor(() => {
      expect(
        screen.queryByText("Enter a client secret")
      ).not.toBeInTheDocument();
    });
  });

  it("restores the secret mask when the identity change that cleared it is reverted", async () => {
    const { user } = renderPage([createMockCredential()]);

    const clientIdField = await screen.findByLabelText("Client ID");
    await user.clear(clientIdField);
    await user.type(clientIdField, "9a2e4d10-3b77-4c58-8e21-1f0c5d6a7b88");
    await waitFor(() => {
      expect(screen.getByLabelText("Client secret")).toHaveValue("");
    });

    // Putting the stored client ID back means the stored secret applies again, so the field must not read as though
    // no secret is configured.
    await user.clear(clientIdField);
    await user.type(clientIdField, "7f6b1665-51f5-48de-a9b6-ac17539583fb");

    await waitFor(() => {
      expect(screen.getByLabelText("Client secret")).toHaveValue("********");
    });
  });

  it("leaves a secret field the admin cleared themselves alone during ID edits", async () => {
    const { user } = renderPage([createMockCredential()]);

    const secretField = await screen.findByLabelText("Client secret");
    await waitFor(() => {
      expect(secretField).toHaveValue("********");
    });
    await user.clear(secretField);

    // A net no-op ID edit leaves the identity matching the stored credential, but the admin emptied the secret field
    // on purpose, so the mask must not reappear under them.
    const tenantIdField = screen.getByLabelText("Tenant ID");
    await user.type(tenantIdField, "x");
    await user.type(tenantIdField, "{backspace}");

    expect(screen.getByLabelText("Client secret")).toHaveValue("");
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

    // Captured before submitting: the error text replaces the label, so findByLabelText cannot reach the field once
    // the error is showing. It is the same node either way.
    const tenantField = await screen.findByLabelText("Tenant ID");
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByText("Enter a tenant ID")).toBeInTheDocument();

    await user.click(tenantField);
    await waitFor(() => {
      expect(screen.queryByText("Enter a tenant ID")).not.toBeInTheDocument();
    });
  });

  it("deletes the credential by sending an empty list", async () => {
    const { user } = renderPage([createMockCredential()]);

    await user.click(await screen.findByRole("button", { name: "Delete" }));

    // Confirm inside the modal, which has its own Delete button.
    const modal = await getOpenModal();
    await user.click(within(modal).getByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(mockedAPI.deleteCredentials).toHaveBeenCalled();
    });
  });

  it("locks the form while a save is in flight", async () => {
    // Definite assignment: a Promise executor runs synchronously, so this is set before the test can use it.
    let finishSave!: () => void;
    mockedAPI.applyCredentials.mockReturnValue(
      new Promise<void>((resolve) => {
        finishSave = resolve;
      })
    );
    const { user } = renderPage([createMockCredential()]);

    const tenantField = await screen.findByLabelText("Tenant ID");
    const deleteButton = screen.getByRole("button", { name: "Delete" });
    await user.click(screen.getByRole("button", { name: "Save" }));

    // The request is built from the values at submit time, so editing mid-flight would let the admin believe they saved
    // something the request never carried.
    await waitFor(() => expect(tenantField).toBeDisabled());
    expect(screen.getByLabelText("Client ID")).toBeDisabled();
    expect(screen.getByLabelText("Client secret")).toBeDisabled();
    expect(deleteButton).toBeDisabled();

    finishSave();
    await waitFor(() => expect(tenantField).toBeEnabled());
  });

  it("disables the fields and save in GitOps mode", async () => {
    renderPage([createMockCredential()], { gitOpsModeEnabled: true });

    expect(await screen.findByLabelText("Tenant ID")).toBeDisabled();
    expect(screen.getByLabelText("Client ID")).toBeDisabled();
    expect(screen.getByLabelText("Client secret")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Delete" })).toBeDisabled();
  });
});
