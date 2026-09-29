import { screen, waitFor } from "@testing-library/react";
import React from "react";
import { QueryClient, QueryClientProvider } from "react-query";

import { notify } from "components/ToastNotification";
import hostAPI from "services/entities/hosts";
import { createCustomRenderer, renderWithSetup } from "test/test-utils";

import DiskEncryptionKeyModal, {
  ESCROW_OFF_TOOLTIP,
  ROTATION_PENDING_TOOLTIP,
} from "./DiskEncryptionKeyModal";

jest.mock("services/entities/hosts");
jest.mock("components/ToastNotification", () => ({
  notify: { success: jest.fn(), error: jest.fn() },
}));

const keyResponse = (rotationPending = false) => ({
  host_id: 7,
  encryption_key: {
    key: "AAAA-BBBB-CCCC",
    updated_at: "2026-09-20T13:00:00Z",
    rotation_pending: rotationPending,
  },
});

const apiError = (reason: string) => ({
  data: { message: "Validation Failed", errors: [{ name: "base", reason }] },
});

describe("DiskEncryptionKeyModal", () => {
  const render = createCustomRenderer({ withBackendMock: true });

  beforeEach(() => {
    jest.resetAllMocks();
    (hostAPI.getEncryptionKey as jest.Mock).mockResolvedValue(keyResponse());
  });

  const renderModal = (
    props: Partial<React.ComponentProps<typeof DiskEncryptionKeyModal>> = {}
  ) =>
    render(
      <DiskEncryptionKeyModal
        platform="darwin"
        hostId={7}
        canRotateKey
        isEscrowEnabled
        onCancel={jest.fn()}
        {...props}
      />
    );

  it("hides Rotate key when the viewer can't rotate", async () => {
    renderModal({ canRotateKey: false });
    await waitFor(() => expect(screen.getByText("Close")).toBeVisible());
    expect(screen.queryByText("Rotate key")).not.toBeInTheDocument();
  });

  it("hides Rotate key for hosts other than macOS", async () => {
    renderModal({ platform: "windows" });
    await waitFor(() => expect(screen.getByText("Close")).toBeVisible());
    expect(screen.queryByText("Rotate key")).not.toBeInTheDocument();
  });

  it("enables Rotate key for an eligible macOS host", async () => {
    renderModal();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Rotate key" })).toBeEnabled()
    );
  });

  it("disables Rotate key with a tooltip when escrow is off", async () => {
    const { user } = renderModal({ isEscrowEnabled: false });
    const button = await screen.findByRole("button", { name: "Rotate key" });
    expect(button).toBeDisabled();
    await user.hover(button);
    expect(await screen.findByText(ESCROW_OFF_TOOLTIP)).toBeInTheDocument();
  });

  it("shows a pending rotation as Rotating...", async () => {
    (hostAPI.getEncryptionKey as jest.Mock).mockResolvedValue(
      keyResponse(true)
    );
    const { user } = renderModal();
    const button = await screen.findByRole("button", { name: "Rotating..." });
    expect(button).toBeDisabled();
    await user.hover(button);
    expect(
      await screen.findByText(ROTATION_PENDING_TOOLTIP)
    ).toBeInTheDocument();
  });

  it("closes with a success toast once the request is sent", async () => {
    (hostAPI.rotateDiskEncryptionKey as jest.Mock).mockResolvedValue(undefined);
    const onCancel = jest.fn();
    const { user } = renderModal({ onCancel });
    await user.click(await screen.findByRole("button", { name: "Rotate key" }));

    await waitFor(() => expect(onCancel).toHaveBeenCalled());
    expect(hostAPI.rotateDiskEncryptionKey).toHaveBeenCalledWith(7);
    expect(notify.success).toHaveBeenCalledWith(
      "Successfully sent request to rotate disk encryption key."
    );
  });

  it("shows a rotation someone else started once the request conflicts", async () => {
    const err = apiError(
      "Disk encryption key rotation is already in progress for this host."
    );
    (hostAPI.rotateDiskEncryptionKey as jest.Mock).mockRejectedValue(err);
    const onCancel = jest.fn();
    const { user } = renderModal({ onCancel });
    await user.click(await screen.findByRole("button", { name: "Rotate key" }));

    await waitFor(() =>
      expect(
        notify.error
      ).toHaveBeenCalledWith(
        "Disk encryption key rotation is already in progress for this host.",
        { response: err }
      )
    );
    expect(onCancel).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Rotating..." })).toBeDisabled();
  });

  // The app's QueryClient keeps the key cached after the modal closes; the shared
  // test renderer's cacheTime: 0 would hide both the cache write and the reuse.
  it("still shows the rotation on reopen without fetching the key again", async () => {
    (hostAPI.rotateDiskEncryptionKey as jest.Mock).mockResolvedValue(undefined);
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const modal = (
      <QueryClientProvider client={client}>
        <DiskEncryptionKeyModal
          platform="darwin"
          hostId={7}
          canRotateKey
          isEscrowEnabled
          onCancel={jest.fn()}
        />
      </QueryClientProvider>
    );

    const { user, unmount } = renderWithSetup(modal);
    await user.click(await screen.findByRole("button", { name: "Rotate key" }));
    await waitFor(() => expect(notify.success).toHaveBeenCalled());
    unmount();

    renderWithSetup(modal);
    expect(
      await screen.findByRole("button", { name: "Rotating..." })
    ).toBeDisabled();
    expect(hostAPI.getEncryptionKey).toHaveBeenCalledTimes(1);
    client.clear();
  });

  it.each([
    [
      "Couldn't rotate disk encryption key. The current key is not decryptable.",
      "Couldn't rotate disk encryption key. The current key is not decryptable.",
    ],
    [
      "something else",
      "Couldn't send request to rotate disk encryption key. Please try again.",
    ],
  ])(
    "keeps the modal open with an error toast for %s",
    async (reason, toast) => {
      const err = apiError(reason);
      (hostAPI.rotateDiskEncryptionKey as jest.Mock).mockRejectedValue(err);
      const onCancel = jest.fn();
      const { user } = renderModal({ onCancel });
      await user.click(
        await screen.findByRole("button", { name: "Rotate key" })
      );

      await waitFor(() =>
        expect(notify.error).toHaveBeenCalledWith(toast, { response: err })
      );
      expect(onCancel).not.toHaveBeenCalled();
      expect(screen.getByRole("button", { name: "Rotate key" })).toBeEnabled();
    }
  );
});
