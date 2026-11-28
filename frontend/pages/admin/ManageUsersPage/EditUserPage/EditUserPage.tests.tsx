import { screen, waitFor } from "@testing-library/react";
import React from "react";

import { createMockConfig } from "__mocks__/configMock";
import createMockUser from "__mocks__/userMock";
import usersAPI from "services/entities/users";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import EditUserPage from "./EditUserPage";

jest.mock("services/entities/users");

const mockUsersAPI = usersAPI as jest.Mocked<typeof usersAPI>;

const API_USER = createMockUser({
  id: 42,
  name: "gitops-bot",
  api_only: true,
  global_role: "admin",
  fleets: [],
});
const BOUND = { issuer: "github", subject: "repo:acme/mdm" };

const renderPage = () => {
  const router = createMockRouter();
  const render = createCustomRenderer({
    withBackendMock: true,
    context: { app: { isPremiumTier: false, config: createMockConfig() } },
  });
  return {
    router,
    ...render(
      <EditUserPage router={router} params={{ user_id: "42" }} location={{}} />
    ),
  };
};

describe("EditUserPage - API-only user external identities", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUsersAPI.getUserById.mockResolvedValue(API_USER);
    mockUsersAPI.updateApiOnlyUser.mockResolvedValue(API_USER);
    mockUsersAPI.getApiOnlyUserExternalIdentities.mockResolvedValue({
      external_identities: [BOUND],
      issuers: ["github"],
    });
    mockUsersAPI.updateApiOnlyUserExternalIdentities.mockResolvedValue({
      external_identities: [],
    });
  });

  it("shows the bound identities and doesn't rewrite them when unchanged", async () => {
    const { user, router } = renderPage();

    expect(await screen.findByDisplayValue(BOUND.subject)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(router.push).toHaveBeenCalled());
    expect(
      mockUsersAPI.updateApiOnlyUserExternalIdentities
    ).not.toHaveBeenCalled();
  });

  it("saves changed identities after the user", async () => {
    const { user, router } = renderPage();

    await user.click(
      await screen.findByRole("button", { name: "Remove external identity 1" })
    );
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(
        mockUsersAPI.updateApiOnlyUserExternalIdentities
      ).toHaveBeenCalledWith(42, [])
    );
    expect(mockUsersAPI.updateApiOnlyUser).toHaveBeenCalled();
    await waitFor(() => expect(router.push).toHaveBeenCalled());
  });

  it("stays on the page when the identities fail to save", async () => {
    mockUsersAPI.updateApiOnlyUserExternalIdentities.mockRejectedValue({
      status: 409,
      data: { message: "already bound to another user" },
    });
    const { user, router } = renderPage();

    await user.click(
      await screen.findByRole("button", { name: "Remove external identity 1" })
    );
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(
        mockUsersAPI.updateApiOnlyUserExternalIdentities
      ).toHaveBeenCalled()
    );
    expect(router.push).not.toHaveBeenCalled();
  });

  it("hides the section when the server has no issuers configured", async () => {
    mockUsersAPI.getApiOnlyUserExternalIdentities.mockResolvedValue({
      external_identities: [],
    });
    renderPage();

    expect(await screen.findByText("Edit API-only user")).toBeInTheDocument();
    expect(screen.queryByText("External identities")).not.toBeInTheDocument();
  });
});
