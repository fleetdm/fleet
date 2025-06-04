import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import { noop } from "lodash";

import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";
import createMockTeam from "__mocks__/teamMock";

import SiteTopNav from ".";

const urlLocation = {
  pathname: "queries/manage",
  search: "",
  query: {},
};

describe("SiteTopNav - component", () => {
  it("renders correct navigation for free global admin", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: false,
          isGlobalAdmin: true,
        },
      },
    });

    const { user } = render(
      <SiteTopNav
        config={createMockConfig()}
        currentUser={createMockUser()}
        location={urlLocation}
        onLogoutUser={noop}
        onUserMenuItemClick={noop}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/controls/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/queries/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /settings/i })
    ).toBeInTheDocument();

    expect(
      screen.getByRole("menuitem", { name: /manage users/i })
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
  it("renders correct navigation for free global maintainer", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: false,
          isGlobalMaintainer: true,
        },
      },
    });

    const { user } = render(
      <SiteTopNav
        config={createMockConfig()}
        currentUser={createMockUser({
          role: "maintainer",
          global_role: "maintainer",
        })}
        location={urlLocation}
        onLogoutUser={noop}
        onUserMenuItemClick={noop}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/controls/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/queries/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /my account/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /documentation/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /sign out/i })
    ).toBeInTheDocument();

    expect(screen.queryByText(/settings/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/manage users/i)).not.toBeInTheDocument();
  });
  it("renders correct navigation for free global observer", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: false,
        },
      },
    });

    const { user } = render(
      <SiteTopNav
        config={createMockConfig()}
        currentUser={createMockUser({
          role: "observer",
          global_role: "observer",
        })}
        location={urlLocation}
        onLogoutUser={noop}
        onUserMenuItemClick={noop}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/queries/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /my account/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /documentation/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /sign out/i })
    ).toBeInTheDocument();

    expect(screen.queryByText(/controls/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/settings/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/manage users/i)).not.toBeInTheDocument();
  });
  it("renders correct navigation for premium global admin", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
        },
      },
    });

    const { user } = render(
      <SiteTopNav
        config={createMockConfig()}
        currentUser={createMockUser()}
        location={urlLocation}
        onLogoutUser={noop}
        onUserMenuItemClick={noop}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/controls/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/queries/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /settings/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /manage users/i })
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
  it("renders correct navigation for premium global maintainer", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isGlobalMaintainer: true,
        },
      },
    });

    const { user } = render(
      <SiteTopNav
        config={createMockConfig()}
        currentUser={createMockUser({
          role: "maintainer",
          global_role: "maintainer",
        })}
        location={urlLocation}
        onLogoutUser={noop}
        onUserMenuItemClick={noop}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/controls/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/queries/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /my account/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /documentation/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /sign out/i })
    ).toBeInTheDocument();

    expect(screen.queryByText(/settings/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/manage users/i)).not.toBeInTheDocument();
  });
  it("renders correct navigation for premium global observer", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
        },
      },
    });

    const { user } = render(
      <SiteTopNav
        config={createMockConfig()}
        currentUser={createMockUser({
          role: "observer",
          global_role: "observer",
        })}
        location={urlLocation}
        onLogoutUser={noop}
        onUserMenuItemClick={noop}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/queries/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /my account/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /documentation/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /sign out/i })
    ).toBeInTheDocument();

    expect(screen.queryByText(/controls/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/settings/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/manage users/i)).not.toBeInTheDocument();
  });
  it("renders correct navigation for premium team admin", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isAnyTeamAdmin: true,
        },
      },
    });

    const { user } = render(
      <SiteTopNav
        config={createMockConfig()}
        currentUser={createMockUser({
          global_role: "",
          teams: [createMockTeam({ role: "admin" })],
        })}
        location={urlLocation}
        onLogoutUser={noop}
        onUserMenuItemClick={noop}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/controls/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/queries/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
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

    expect(screen.queryByText(/manage users/i)).not.toBeInTheDocument();
  });
  it("renders correct navigation for premium team maintainer", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isAnyTeamMaintainer: true,
        },
      },
    });

    const { user } = render(
      <SiteTopNav
        config={createMockConfig()}
        currentUser={createMockUser({
          role: "maintainer",
          global_role: "",
        })}
        location={urlLocation}
        onLogoutUser={noop}
        onUserMenuItemClick={noop}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/controls/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/queries/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /my account/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /documentation/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /sign out/i })
    ).toBeInTheDocument();

    expect(screen.queryByText(/settings/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/manage users/i)).not.toBeInTheDocument();
  });
  it("renders correct navigation for premium team observer", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
        },
      },
    });

    const { user } = render(
      <SiteTopNav
        config={createMockConfig()}
        currentUser={createMockUser({
          role: "observer",
          global_role: "",
        })}
        location={urlLocation}
        onLogoutUser={noop}
        onUserMenuItemClick={noop}
      />
    );

    await user.click(screen.getByTestId("user-avatar"));

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/queries/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();

    expect(
      screen.getByRole("menuitem", { name: /my account/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /documentation/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: /sign out/i })
    ).toBeInTheDocument();

    expect(screen.queryByText(/controls/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/settings/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/manage users/i)).not.toBeInTheDocument();
  });
});
