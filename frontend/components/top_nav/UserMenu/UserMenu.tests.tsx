import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import { noop } from "lodash";

import createMockUser from "__mocks__/userMock";
import createMockTeam, { createMockTeamSummary } from "__mocks__/teamMock";

import UserMenu from ".";

describe("UserMenu - component", () => {
  it("renders correct menu items for a global admin on the free tier", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: false,
          isSandboxMode: false,
        },
      },
    });

    const { user } = render(
      <UserMenu
        onLogout={noop}
        onUserMenuItemClick={noop}
        isGlobalAdmin
        isAnyTeamAdmin={false}
        currentUser={createMockUser()}
        currentTeam={undefined}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(
      screen.getByRole("menuitem", { name: /labels/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /organization settings/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /integrations/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /users/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /my account/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /documentation/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /sign out/i })
    ).toBeInTheDocument();

    expect(
      screen.queryByRole("menuitem", { name: /fleets/i })
    ).not.toBeInTheDocument();
  });

  it("renders correct menu items for a global admin on the premium tier", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isSandboxMode: false,
        },
      },
    });

    const { user } = render(
      <UserMenu
        onLogout={noop}
        onUserMenuItemClick={noop}
        isGlobalAdmin
        isAnyTeamAdmin={false}
        currentUser={createMockUser()}
        currentTeam={undefined}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(
      screen.getByRole("menuitem", { name: /labels/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /organization settings/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /integrations/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /users/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /fleets/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /my account/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /documentation/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /sign out/i })
    ).toBeInTheDocument();
  });

  it("renders correct menu items for a fleet-level admin", async () => {
    const mockTeam = createMockTeam({ role: "admin" });

    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isSandboxMode: false,
        },
      },
    });

    const { user } = render(
      <UserMenu
        onLogout={noop}
        onUserMenuItemClick={noop}
        isGlobalAdmin={false}
        isAnyTeamAdmin
        currentUser={createMockUser({ global_role: null, teams: [mockTeam] })}
        currentTeam={createMockTeamSummary({ id: mockTeam.id })}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(
      screen.getByRole("menuitem", { name: /labels/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /users/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /agent options/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /settings/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /my account/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /documentation/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /sign out/i })
    ).toBeInTheDocument();

    expect(
      screen.queryByRole("menuitem", { name: /organization settings/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("menuitem", { name: /integrations/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("menuitem", { name: /fleets/i })
    ).not.toBeInTheDocument();
  });

  it("renders correct menu items for a global admin in sandbox mode", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isSandboxMode: true,
        },
      },
    });

    const { user } = render(
      <UserMenu
        onLogout={noop}
        onUserMenuItemClick={noop}
        isGlobalAdmin
        isAnyTeamAdmin={false}
        currentUser={createMockUser()}
        currentTeam={undefined}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(
      screen.getByRole("menuitem", { name: /labels/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /integrations/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /my account/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /documentation/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /sign out/i })
    ).toBeInTheDocument();

    expect(
      screen.queryByRole("menuitem", { name: /organization settings/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("menuitem", { name: /users/i })
    ).not.toBeInTheDocument();
  });

  it("renders correct menu items for a non-admin", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isSandboxMode: false,
        },
      },
    });

    const { user } = render(
      <UserMenu
        onLogout={noop}
        onUserMenuItemClick={noop}
        isGlobalAdmin={false}
        isAnyTeamAdmin={false}
        currentUser={createMockUser({ global_role: "observer" })}
        currentTeam={undefined}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(
      screen.getByRole("menuitem", { name: /labels/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /my account/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /documentation/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /sign out/i })
    ).toBeInTheDocument();

    expect(
      screen.queryByRole("menuitem", { name: /organization settings/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("menuitem", { name: /integrations/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("menuitem", { name: /users/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("menuitem", { name: /agent options/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("menuitem", { name: /fleets/i })
    ).not.toBeInTheDocument();
  });
});
