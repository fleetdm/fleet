import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { noop } from "lodash";

import createMockUser from "__mocks__/userMock";

import SiteTopNav from "./SiteTopNav";

describe("SiteTopNav", () => {
  it("renders the correct nav items for global admins", () => {
    const render = createCustomRenderer({
      context: {
        app: { isGlobalAdmin: true },
      },
    });

    render(
      <SiteTopNav
        onLogoutUser={noop}
        onNavItemClick={noop}
        pathname="/test"
        currentUser={createMockUser()}
        config={{
          org_info: { org_logo_url: "http://test.com", org_name: "test" },
        }}
      />
    );

    expect(screen.getByText("Hosts")).toBeInTheDocument();
    expect(screen.getByText("Software")).toBeInTheDocument();
    expect(screen.getByText("Queries")).toBeInTheDocument();
    expect(screen.getByText("Schedule")).toBeInTheDocument();
    expect(screen.getByText("Policies")).toBeInTheDocument();
  });

  it("shows settings and manage users for global admin", async () => {
    const render = createCustomRenderer({
      context: {
        app: { isGlobalAdmin: true },
      },
    });

    const { user } = render(
      <SiteTopNav
        onLogoutUser={noop}
        onNavItemClick={noop}
        pathname="/test"
        currentUser={createMockUser()}
        config={{
          org_info: { org_logo_url: "http://test.com", org_name: "test" },
        }}
      />
    );

    await user.click(screen.getAllByRole("button")[0]);

    expect(
      screen.getByRole("button", { name: "Settings" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Manage users" })
    ).toBeInTheDocument();
  });

  it("shows only 'My Account' for observers", async () => {
    const render = createCustomRenderer({
      context: {
        app: { isGlobalObserver: true },
      },
    });

    const { user } = render(
      <SiteTopNav
        onLogoutUser={noop}
        onNavItemClick={noop}
        pathname="/test"
        currentUser={createMockUser()}
        config={{
          org_info: { org_logo_url: "http://test.com", org_name: "test" },
        }}
      />
    );

    await user.click(screen.getAllByRole("button")[0]);

    expect(screen.queryByRole("button", { name: "Settings" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Manage users" })).toBeNull();
    expect(
      screen.getByRole("button", { name: "My account" })
    ).toBeInTheDocument();
  });
});
