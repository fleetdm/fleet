import { screen, waitFor, within } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import createMockUser from "__mocks__/userMock";
import { notify } from "components/ToastNotification";
import usersAPI from "services/entities/users";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import AccountPage from "./AccountPage";

jest.mock("services/entities/users");
jest.mock("services/entities/version");
jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const mockUsersAPI = usersAPI as jest.Mocked<typeof usersAPI>;

const currentUser = createMockUser({ email: "me@example.com" });
const smtpConfiguredConfig = createMockConfig();
smtpConfiguredConfig.smtp_settings = {
  ...smtpConfiguredConfig.smtp_settings,
  configured: true,
} as typeof smtpConfiguredConfig.smtp_settings;

describe("AccountPage", () => {
  const render = createCustomRenderer({
    context: {
      app: {
        currentUser,
        config: smtpConfiguredConfig,
      },
    },
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it("shows a wrong current password on the Current password field", async () => {
    mockUsersAPI.changePassword.mockRejectedValue({
      status: 422,
      data: {
        errors: [
          { name: "old_password", reason: "old password does not match" },
        ],
      },
    });
    const { user } = render(<AccountPage router={createMockRouter()} />);

    await user.click(screen.getByRole("button", { name: "Change password" }));
    await user.type(screen.getByLabelText("Current password"), "wrong-pass");
    await user.type(screen.getByLabelText("New password"), "N3w-passw0rd!!");
    await user.type(
      screen.getByLabelText("Confirm password"),
      "N3w-passw0rd!!"
    );
    await user.click(
      within(
        screen.getByLabelText("Current password").closest("form") as HTMLElement
      ).getByRole("button", { name: "Change password" })
    );

    // The server's reason, with its first letter capitalized.
    expect(
      await screen.findByText("Old password does not match")
    ).toBeInTheDocument();
    // Shown inline and toasted exactly once; the generic fallback must not fire.
    expect(notify.error).toHaveBeenCalledTimes(1);
    expect(notify.error).toHaveBeenCalledWith("Old password does not match");
  });

  it("does not replay an old password error when Change password is reopened", async () => {
    mockUsersAPI.changePassword.mockRejectedValueOnce({
      status: 422,
      data: {
        errors: [
          { name: "old_password", reason: "old password does not match" },
        ],
      },
    });
    mockUsersAPI.changePassword.mockResolvedValueOnce({} as never);
    const { user } = render(<AccountPage router={createMockRouter()} />);

    const submit = async (current: string) => {
      await user.click(screen.getByRole("button", { name: "Change password" }));
      await user.type(screen.getByLabelText("Current password"), current);
      await user.type(screen.getByLabelText("New password"), "N3w-passw0rd!!");
      await user.type(
        screen.getByLabelText("Confirm password"),
        "N3w-passw0rd!!"
      );
      await user.click(
        within(
          screen
            .getByLabelText("Current password")
            .closest("form") as HTMLElement
        ).getByRole("button", { name: "Change password" })
      );
    };

    await submit("wrong-pass");
    expect(
      await screen.findByText("Old password does not match")
    ).toBeInTheDocument();

    // Succeed on the second try; the modal closes. (With an error showing, the
    // field's label is replaced by the error text, so find it by value.)
    const currentInput = screen.getByDisplayValue("wrong-pass");
    await user.clear(currentInput);
    await user.type(currentInput, "right-pass");
    await user.click(
      within(currentInput.closest("form") as HTMLElement).getByRole("button", {
        name: "Change password",
      })
    );
    await waitFor(() =>
      expect(screen.queryByLabelText("Current password")).toBeNull()
    );

    jest.clearAllMocks();
    await user.click(screen.getByRole("button", { name: "Change password" }));
    expect(
      await screen.findByLabelText("Current password")
    ).toBeInTheDocument();
    expect(screen.queryByText("Old password does not match")).toBeNull();
    expect(notify.error).not.toHaveBeenCalled();
  });

  it("falls back to the toast for a password error with no field", async () => {
    mockUsersAPI.changePassword.mockRejectedValue({
      status: 500,
      data: { errors: [{ name: "base", reason: "boom" }] },
    });
    const { user } = render(<AccountPage router={createMockRouter()} />);

    await user.click(screen.getByRole("button", { name: "Change password" }));
    await user.type(screen.getByLabelText("Current password"), "wrong-pass");
    await user.type(screen.getByLabelText("New password"), "N3w-passw0rd!!");
    await user.type(
      screen.getByLabelText("Confirm password"),
      "N3w-passw0rd!!"
    );
    await user.click(
      within(
        screen.getByLabelText("Current password").closest("form") as HTMLElement
      ).getByRole("button", { name: "Change password" })
    );

    await waitFor(() =>
      expect(notify.error).toHaveBeenCalledWith(
        "Could not update. Please try again.",
        expect.anything()
      )
    );
  });

  it("tells the user the password is wrong when Confirm email update gets a bare 403", async () => {
    mockUsersAPI.update.mockRejectedValue({
      status: 403,
      data: { message: "Permission Denied" },
    });
    const { user } = render(<AccountPage router={createMockRouter()} />);

    const emailField = screen.getByLabelText("Email");
    await user.clear(emailField);
    await user.type(emailField, "new@example.com");
    await user.click(screen.getByRole("button", { name: "Update" }));

    await user.type(await screen.findByLabelText("Password"), "wrong-pass");
    await user.click(screen.getByRole("button", { name: "Submit" }));

    expect(await screen.findByText("Incorrect password")).toBeInTheDocument();
    // The modal stays open so the user can retry.
    expect(screen.getByText("Confirm email update")).toBeInTheDocument();
    // One toast, not one from the modal and another from the form behind it.
    expect(notify.error).toHaveBeenCalledTimes(1);
    expect(notify.error).toHaveBeenCalledWith("Incorrect password");
  });
});
