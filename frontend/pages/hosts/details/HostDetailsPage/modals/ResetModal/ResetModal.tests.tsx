import { screen } from "@testing-library/react";
import React from "react";

import { notify } from "components/ToastNotification";
import hostAPI from "services/entities/hosts";
import { createCustomRenderer } from "test/test-utils";

import ResetModal from "./ResetModal";

jest.mock("services/entities/hosts");
jest.mock("components/ToastNotification", () => ({
  notify: { success: jest.fn(), error: jest.fn() },
}));

const MOCK_PROPS = {
  id: 7,
  hostName: "windows-host-1",
  onSuccess: jest.fn(),
  onClose: jest.fn(),
};

describe("ResetModal", () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  it("keeps Reset disabled until confirmed and while the request is in flight", async () => {
    jest
      .mocked(hostAPI.resetHost)
      .mockReturnValue(new Promise(() => undefined));
    const render = createCustomRenderer();
    const { user } = render(<ResetModal {...MOCK_PROPS} />);

    const resetButton = screen.getByRole("button", { name: "Reset" });
    expect(resetButton).toBeDisabled();

    await user.click(screen.getByRole("checkbox", { name: /windows-host-1/ }));
    expect(resetButton).toBeEnabled();

    await user.click(resetButton);
    expect(resetButton).toBeDisabled();
    await user.click(resetButton);
    expect(hostAPI.resetHost).toHaveBeenCalledTimes(1);
  });

  it("resets the host, confirms with a toast and closes", async () => {
    jest.mocked(hostAPI.resetHost).mockResolvedValue({});
    const render = createCustomRenderer();
    const { user } = render(<ResetModal {...MOCK_PROPS} />);

    await user.click(screen.getByRole("checkbox", { name: /windows-host-1/ }));
    await user.click(screen.getByRole("button", { name: "Reset" }));

    expect(hostAPI.resetHost).toHaveBeenCalledTimes(1);
    expect(hostAPI.resetHost).toHaveBeenCalledWith(7);
    expect(notify.success).toHaveBeenCalledTimes(1);
    expect(notify.success).toHaveBeenCalledWith(
      "Resetting host or will reset when the host comes online."
    );
    expect(MOCK_PROPS.onSuccess).toHaveBeenCalledTimes(1);
    expect(MOCK_PROPS.onClose).toHaveBeenCalledTimes(1);
  });

  it("shows the server's reason and closes without reporting success when the request fails", async () => {
    const error = {
      response: {
        data: {
          errors: [{ name: "host_id", reason: "Host is already wiped." }],
        },
      },
    };
    jest.mocked(hostAPI.resetHost).mockRejectedValue(error);
    const render = createCustomRenderer();
    const { user } = render(<ResetModal {...MOCK_PROPS} />);

    await user.click(screen.getByRole("checkbox", { name: /windows-host-1/ }));
    await user.click(screen.getByRole("button", { name: "Reset" }));

    expect(notify.error).toHaveBeenCalledTimes(1);
    expect(notify.error).toHaveBeenCalledWith("Host is already wiped.", {
      response: error,
    });
    expect(notify.success).not.toHaveBeenCalled();
    expect(MOCK_PROPS.onSuccess).not.toHaveBeenCalled();
    expect(MOCK_PROPS.onClose).toHaveBeenCalledTimes(1);
  });
});
