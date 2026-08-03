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
  it("renders correct navigation for free global admin", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: false,
          isGlobalAdmin: true,
        },
      },
    });

    render(
      <SiteTopNav
        config={createMockConfig()}
        currentUser={createMockUser()}
        location={urlLocation}
        onLogoutUser={noop}
        onUserMenuItemClick={noop}
      />
    );

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/controls/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/reports/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
  });

  it("renders correct navigation for free global maintainer", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: false,
          isGlobalMaintainer: true,
        },
      },
    });

    render(
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

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/controls/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/reports/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
  });

  it("renders correct navigation for free global observer", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: false,
        },
      },
    });

    render(
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

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/reports/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();

    expect(screen.queryByText(/controls/i)).not.toBeInTheDocument();
  });

  it("renders correct navigation for premium global admin", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
        },
      },
    });

    render(
      <SiteTopNav
        config={createMockConfig()}
        currentUser={createMockUser()}
        location={urlLocation}
        onLogoutUser={noop}
        onUserMenuItemClick={noop}
      />
    );

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/controls/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/reports/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
  });

  it("renders correct navigation for premium global maintainer", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isGlobalMaintainer: true,
        },
      },
    });

    render(
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

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/controls/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/reports/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
  });

  it("renders correct navigation for premium global observer", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
        },
      },
    });

    render(
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

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/reports/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();

    expect(screen.queryByText(/controls/i)).not.toBeInTheDocument();
  });

  it("renders correct navigation for premium team admin", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isAnyTeamAdmin: true,
        },
      },
    });

    render(
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

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/controls/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/reports/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
  });

  it("renders correct navigation for premium team maintainer", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isAnyTeamMaintainer: true,
        },
      },
    });

    render(
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

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/controls/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/reports/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();
  });

  it("renders correct navigation for premium team observer", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
        },
      },
    });

    render(
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

    expect(screen.getByText(/hosts/i)).toBeInTheDocument();
    expect(screen.getByText(/software/i)).toBeInTheDocument();
    expect(screen.getByText(/reports/i)).toBeInTheDocument();
    expect(screen.getByText(/policies/i)).toBeInTheDocument();

    expect(screen.queryByText(/controls/i)).not.toBeInTheDocument();
  });
});
