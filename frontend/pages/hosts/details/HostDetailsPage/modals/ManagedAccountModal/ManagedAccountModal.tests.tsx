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

  it("shows the auto-rotate banner when autoRotateAt is set", async () => {
    render(
      <ManagedAccountModal
        hostId={7}
        canRotatePassword
        autoRotateAt="2026-04-30T14:35:00Z"
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

  it("does not show the banner when autoRotateAt is undefined", async () => {
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
