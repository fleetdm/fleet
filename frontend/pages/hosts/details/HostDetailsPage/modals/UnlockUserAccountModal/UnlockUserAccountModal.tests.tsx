import React from "react";

import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import hostAPI from "services/entities/hosts";

import UnlockUserAccountModal from "./UnlockUserAccountModal";

jest.mock("services/entities/hosts");

describe("UnlockUserAccountModal", () => {
  const render = createCustomRenderer({ withBackendMock: true });

  beforeEach(() => {
    jest.resetAllMocks();
  });

  it("requires a non-whitespace username and submits the trimmed value", async () => {
    (hostAPI.unlockUserAccount as jest.Mock).mockResolvedValue({});
    const onExit = jest.fn();
    const onSuccess = jest.fn();
    const { user } = render(
      <UnlockUserAccountModal
        id={7}
        hostName="Anna's Mac"
        onExit={onExit}
        onSuccess={onSuccess}
      />
    );

    const submit = screen.getByRole("button", { name: "Unlock user" });
    const username = screen.getByLabelText("Username");
    expect(submit).toBeDisabled();

    await user.type(username, "   ");
    expect(submit).toBeDisabled();

    await user.type(username, "anna  ");
    await user.click(submit);

    await waitFor(() =>
      expect(hostAPI.unlockUserAccount).toHaveBeenCalledWith(7, "anna")
    );
    expect(onSuccess).toHaveBeenCalledTimes(1);
    expect(onExit).toHaveBeenCalledTimes(1);
  });

  it("keeps the modal open when the request fails", async () => {
    (hostAPI.unlockUserAccount as jest.Mock).mockRejectedValue(
      new Error("request failed")
    );
    const onExit = jest.fn();
    const { user } = render(
      <UnlockUserAccountModal id={7} hostName="Anna's Mac" onExit={onExit} />
    );

    await user.type(screen.getByLabelText("Username"), "anna");
    await user.click(screen.getByRole("button", { name: "Unlock user" }));

    await waitFor(() => expect(hostAPI.unlockUserAccount).toHaveBeenCalled());
    expect(onExit).not.toHaveBeenCalled();
    expect(screen.getByText("Unlock user account")).toBeVisible();
  });
});
