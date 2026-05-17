import React from "react";

import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import hostAPI from "services/entities/hosts";

import RecoveryLockPasswordModal from "./RecoveryLockPasswordModal";

jest.mock("services/entities/hosts");

const mockPasswordResponse = {
  host_id: 7,
  recovery_lock_password: {
    password: "supersecret",
    updated_at: "2026-04-30T13:00:00Z",
  },
};

describe("RecoveryLockPasswordModal", () => {
  const render = createCustomRenderer({ withBackendMock: true });

  beforeEach(() => {
    jest.resetAllMocks();
    (hostAPI.getRecoveryLockPassword as jest.Mock).mockResolvedValue(
      mockPasswordResponse
    );
  });

  it("hides the rotate button when canRotatePassword is false", async () => {
    render(
      <RecoveryLockPasswordModal
        hostId={7}
        canRotatePassword={false}
        onCancel={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(screen.getByText("Close")).toBeVisible();
    });
    expect(screen.queryByText("Rotate password")).not.toBeInTheDocument();
  });

  it("shows the rotate button when canRotatePassword is true", async () => {
    render(
      <RecoveryLockPasswordModal
        hostId={7}
        canRotatePassword
        onCancel={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(screen.getByText("Rotate password")).toBeVisible();
    });
  });
});
