import { screen, waitFor } from "@testing-library/react";
import React from "react";

import createMockUser from "__mocks__/userMock";
import PATHS from "router/paths";
import usersAPI from "services/entities/users";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import ResetPasswordPage from "./ResetPasswordPage";

jest.mock("services/entities/users");

describe("ResetPasswordPage", () => {
  it("sends a user with no role to the no access page", async () => {
    (usersAPI.performRequiredPasswordReset as jest.Mock).mockRejectedValue({
      data: { message: "either global role or fleet role needs to be defined" },
    });
    const router = createMockRouter();
    const render = createCustomRenderer({
      context: {
        app: {
          currentUser: createMockUser({ global_role: "admin" }),
          setConfig: jest.fn(),
        },
      },
    });
    const { user } = render(
      <ResetPasswordPage location={{ query: {} }} router={router} />
    );

    const password = "new-password-123!";
    await user.type(screen.getByPlaceholderText("New password"), password);
    await user.type(screen.getByPlaceholderText("Confirm password"), password);
    await user.click(screen.getByRole("button", { name: "Reset password" }));

    await waitFor(() =>
      expect(router.push).toHaveBeenCalledWith(PATHS.NO_ACCESS)
    );
  });
});
