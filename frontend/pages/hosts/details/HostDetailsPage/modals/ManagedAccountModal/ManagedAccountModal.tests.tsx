import React from "react";

import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import hostAPI from "services/entities/hosts";

import ManagedAccountModal from "./ManagedAccountModal";

jest.mock("services/entities/hosts");

const mockPasswordResponse = {
  host_id: 7,
  managed_account_password: {
    username: "_fleetadmin",
    password: "supersecret",
    updated_at: "2026-04-30T13:00:00Z",
  },
};

describe("ManagedAccountModal", () => {
  const render = createCustomRenderer({ withBackendMock: true });

  beforeEach(() => {
    jest.resetAllMocks();
    (hostAPI.getManagedAccountPassword as jest.Mock).mockResolvedValue(
      mockPasswordResponse
    );
  });

  it("renders username and password masked input", async () => {
    render(
      <ManagedAccountModal
        hostId={7}
        canRotatePassword
        onCancel={jest.fn()}
        onRotate={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(screen.getByText("_fleetadmin")).toBeVisible();
    });
    expect(screen.getByText("Username")).toBeVisible();
  });

  it("shows the auto-rotate banner when auto_rotate_at is in the response", async () => {
    (hostAPI.getManagedAccountPassword as jest.Mock).mockResolvedValue({
      ...mockPasswordResponse,
      managed_account_password: {
        ...mockPasswordResponse.managed_account_password,
        auto_rotate_at: "2026-04-30T14:35:00Z",
      },
    });

    render(
      <ManagedAccountModal
        hostId={7}
        canRotatePassword
        onCancel={jest.fn()}
        onRotate={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(
        screen.getByText(/Password rotates automatically after/i)
      ).toBeVisible();
    });
  });

  it("does not show the banner when auto_rotate_at is missing from the response", async () => {
    render(
      <ManagedAccountModal
        hostId={7}
        canRotatePassword
        onCancel={jest.fn()}
        onRotate={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(screen.getByText("_fleetadmin")).toBeVisible();
    });
    expect(
      screen.queryByText(/Password rotates automatically after/i)
    ).not.toBeInTheDocument();
  });

  it("hides the rotate button when canRotatePassword is false", async () => {
    render(
      <ManagedAccountModal
        hostId={7}
        canRotatePassword={false}
        onCancel={jest.fn()}
        onRotate={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(screen.getByText("_fleetadmin")).toBeVisible();
    });
    expect(screen.queryByText("Rotate password")).not.toBeInTheDocument();
  });

  it("shows the rotate button when canRotatePassword is true", async () => {
    render(
      <ManagedAccountModal
        hostId={7}
        canRotatePassword
        onCancel={jest.fn()}
        onRotate={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(screen.getByText("Rotate password")).toBeVisible();
    });
  });

  it("calls rotate API and onRotate on success", async () => {
    (hostAPI.rotateManagedLocalAccountPassword as jest.Mock).mockResolvedValue(
      undefined
    );
    const onRotate = jest.fn();

    const { user } = render(
      <ManagedAccountModal
        hostId={7}
        canRotatePassword
        onCancel={jest.fn()}
        onRotate={onRotate}
      />
    );

    const button = await screen.findByText("Rotate password");
    await user.click(button);

    await waitFor(() => {
      expect(hostAPI.rotateManagedLocalAccountPassword).toHaveBeenCalledWith(7);
    });
    await waitFor(() => {
      expect(onRotate).toHaveBeenCalled();
    });
  });

  it("shows the pending-rotation banner when pending_rotation is in the response", async () => {
    (hostAPI.getManagedAccountPassword as jest.Mock).mockResolvedValue({
      ...mockPasswordResponse,
      managed_account_password: {
        ...mockPasswordResponse.managed_account_password,
        pending_rotation: true,
      },
    });

    render(
      <ManagedAccountModal
        hostId={7}
        canRotatePassword
        onCancel={jest.fn()}
        onRotate={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(
        screen.getByText(
          "Password will rotate once the host acknowledges the request."
        )
      ).toBeVisible();
    });
    expect(
      screen.queryByText(/Password rotates automatically after/i)
    ).not.toBeInTheDocument();
  });

  it("shows the pending-rotation banner after a successful rotate", async () => {
    (hostAPI.rotateManagedLocalAccountPassword as jest.Mock).mockResolvedValue(
      undefined
    );
    // Initial fetch returns auto_rotate_at; we expect the just-rotated state to
    // override the banner regardless of what the refetch returns.
    (hostAPI.getManagedAccountPassword as jest.Mock).mockResolvedValue({
      ...mockPasswordResponse,
      managed_account_password: {
        ...mockPasswordResponse.managed_account_password,
        auto_rotate_at: "2026-04-30T14:35:00Z",
      },
    });

    const { user } = render(
      <ManagedAccountModal
        hostId={7}
        canRotatePassword
        onCancel={jest.fn()}
        onRotate={jest.fn()}
      />
    );

    const button = await screen.findByText("Rotate password");
    await user.click(button);

    await waitFor(() => {
      expect(
        screen.getByText(
          "Password will rotate once the host acknowledges the request."
        )
      ).toBeVisible();
    });
    // The auto-rotate banner is replaced by the pending-rotation banner.
    expect(
      screen.queryByText(/Password rotates automatically after/i)
    ).not.toBeInTheDocument();
  });

  it("does not call onRotate when rotate API errors", async () => {
    (hostAPI.rotateManagedLocalAccountPassword as jest.Mock).mockRejectedValue(
      new Error("boom")
    );
    const onRotate = jest.fn();

    const { user } = render(
      <ManagedAccountModal
        hostId={7}
        canRotatePassword
        onCancel={jest.fn()}
        onRotate={onRotate}
      />
    );

    const button = await screen.findByText("Rotate password");
    await user.click(button);

    await waitFor(() => {
      expect(hostAPI.rotateManagedLocalAccountPassword).toHaveBeenCalledWith(7);
    });
    expect(onRotate).not.toHaveBeenCalled();
  });
});
