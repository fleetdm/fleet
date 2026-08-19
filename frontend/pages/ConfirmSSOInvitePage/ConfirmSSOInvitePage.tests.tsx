import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "react-query";

import { createMockRouter, renderWithSetup } from "test/test-utils";
import inviteAPI from "services/entities/invites";
import usersAPI from "services/entities/users";
import sessionsAPI from "services/entities/sessions";
import { IUser } from "interfaces/user";

import ConfirmSSOInvitePage from "./ConfirmSSOInvitePage";

jest.mock("services/entities/invites");
jest.mock("services/entities/users");
jest.mock("services/entities/sessions");

const mockInviteAPI = inviteAPI as jest.Mocked<typeof inviteAPI>;
const mockUsersAPI = usersAPI as jest.Mocked<typeof usersAPI>;
const mockSessionsAPI = sessionsAPI as jest.Mocked<typeof sessionsAPI>;

const renderPage = (token = "abc") => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, cacheTime: 0 } },
  });
  return renderWithSetup(
    <QueryClientProvider client={client}>
      <ConfirmSSOInvitePage
        params={{ invite_token: token }}
        router={createMockRouter()}
      />
    </QueryClientProvider>
  );
};

describe("ConfirmSSOInvitePage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("calls usersAPI.create with the email resolved from the verified invite, then triggers SSO initialization", async () => {
    // The page eventually does `window.location.href = url` after
    // initializeSSO resolves, but JSDOM does not implement navigation.
    // Resolve usersAPI.create successfully (its payload is what we assert
    // on) and reject initializeSSO so the post-create code path
    // short-circuits before touching window.location. We still assert that
    // initializeSSO was called.
    mockInviteAPI.verify.mockResolvedValue({
      invite: {
        created_at: "2026-05-07T00:00:00Z",
        updated_at: "2026-05-07T00:00:00Z",
        id: 1,
        invited_by: 1,
        email: "invitee@example.com",
        name: "Invitee Name",
        sso_enabled: true,
        global_role: "observer",
        teams: [],
      },
    });
    mockUsersAPI.create.mockResolvedValue({} as IUser);
    mockSessionsAPI.initializeSSO.mockRejectedValue(
      new Error("redirect skipped")
    );

    const { user } = renderPage("token-xyz");

    expect(
      await screen.findByRole("textbox", { name: "Full name" })
    ).toBeInTheDocument();

    const emailInput = screen.getByLabelText("Email") as HTMLInputElement;
    expect(emailInput).toBeDisabled();
    expect(emailInput.value).toBe("invitee@example.com");

    await user.click(screen.getByRole("button", { name: "Submit" }));

    await waitFor(() => {
      expect(mockUsersAPI.create).toHaveBeenCalledWith({
        email: "invitee@example.com",
        invite_token: "token-xyz",
        name: "Invitee Name",
        sso_invite: true,
      });
    });

    await waitFor(() => {
      expect(mockSessionsAPI.initializeSSO).toHaveBeenCalled();
    });
  });

  it("renders the invalid invite token message when verification fails", async () => {
    // Reject with a 4xx-shaped error so DEFAULT_USE_QUERY_OPTIONS does not
    // trigger retries.
    mockInviteAPI.verify.mockRejectedValue({ status: 404, message: "invalid" });

    renderPage("bad-token");

    expect(
      await screen.findByText(/this invite token is invalid/i)
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("textbox", { name: "Full name" })
    ).not.toBeInTheDocument();
  });
});
