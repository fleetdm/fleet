import { screen, waitFor } from "@testing-library/react";
import React from "react";

import { createMockTeamSummary } from "__mocks__/teamMock";
import createMockUser from "__mocks__/userMock";
import { emptySelfServiceCategoriesHandler } from "test/handlers/self-service-categories-handlers";
import mockServer from "test/mock-server";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import SoftwareLibrarySettingsPage from "./SoftwareLibrarySettingsPage";

const mockTeam = createMockTeamSummary({ id: 1, name: "Workstations" });

const premiumAdminContext = {
  app: {
    isPremiumTier: true,
    isGlobalAdmin: true,
    currentUser: createMockUser({ global_role: "admin" }),
    availableTeams: [mockTeam],
    setCurrentTeam: jest.fn(),
  },
};

interface ILocationOpts {
  section?: string;
  extraQuery?: string;
}

const buildLocation = ({ section, extraQuery = "" }: ILocationOpts = {}) => {
  const basePath = section
    ? `/software/library/settings/${section}`
    : "/software/library/settings";
  const search = `?fleet_id=1${extraQuery ? `&${extraQuery}` : ""}`;
  return {
    pathname: basePath,
    search,
    query: { fleet_id: "1" },
    hash: "",
  };
};

describe("SoftwareLibrarySettingsPage", () => {
  beforeEach(() => jest.clearAllMocks());

  it("renders the premium gate on Fleet Free", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: false,
          isGlobalAdmin: true,
          currentUser: createMockUser({ global_role: "admin" }),
        },
      },
    });

    const { container } = render(
      <SoftwareLibrarySettingsPage
        router={createMockRouter()}
        location={buildLocation()}
        params={{}}
      />
    );

    expect(
      screen.getByText("This feature is included in Fleet Premium.")
    ).toBeInTheDocument();
    // Fleet Free has no concept of fleets. The dropdown must be hidden, and a
    // static page title takes its place.
    expect(
      container.querySelector(".fleet-dropdown-wrapper")
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("heading", {
        level: 1,
        name: "Software library settings",
      })
    ).toBeInTheDocument();
  });

  it("redirects to the categories section when no :section is in the URL", async () => {
    mockServer.use(emptySelfServiceCategoriesHandler);
    const router = createMockRouter();
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    render(
      <SoftwareLibrarySettingsPage
        router={router}
        location={buildLocation()}
        params={{}}
      />
    );

    await waitFor(() => {
      expect(router.replace).toHaveBeenCalledWith({
        pathname: "/software/library/settings/categories",
        search: "?fleet_id=1",
      });
    });
  });

  it("redirects an unknown :section back to categories, preserving query", async () => {
    mockServer.use(emptySelfServiceCategoriesHandler);
    const router = createMockRouter();
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    render(
      <SoftwareLibrarySettingsPage
        router={router}
        location={buildLocation({
          section: "nope",
          extraQuery: "add_category=1",
        })}
        params={{ section: "nope" }}
      />
    );

    await waitFor(() => {
      expect(router.replace).toHaveBeenCalledWith({
        pathname: "/software/library/settings/categories",
        search: "?fleet_id=1&add_category=1",
      });
    });
  });

  it("renders the categories section when :section=categories", async () => {
    mockServer.use(emptySelfServiceCategoriesHandler);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    render(
      <SoftwareLibrarySettingsPage
        router={createMockRouter()}
        location={buildLocation({ section: "categories" })}
        params={{ section: "categories" }}
      />
    );

    expect(
      await screen.findByText("No self-service categories")
    ).toBeInTheDocument();
  });

  it("renders both side-nav items with the current section marked active", async () => {
    mockServer.use(emptySelfServiceCategoriesHandler);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { container } = render(
      <SoftwareLibrarySettingsPage
        router={createMockRouter()}
        location={buildLocation({ section: "categories" })}
        params={{ section: "categories" }}
      />
    );

    // The nav item and the section header both read "Self-service categories",
    // so match the h2 specifically to confirm the section rendered.
    expect(
      await screen.findByRole("heading", {
        level: 2,
        name: "Self-service categories",
      })
    ).toBeInTheDocument();
    expect(screen.getByText("Patch deadline")).toBeInTheDocument();

    // Fleet's SideNavItem applies `--active` to the <li> for the current
    // section. react-router <Link> doesn't render href without a Router in
    // Jest, so the active class is the stable signal here.
    const activeItems = container.querySelectorAll(".side-nav-item--active");
    expect(activeItems).toHaveLength(1);
    expect(activeItems[0]).toHaveTextContent("Self-service categories");
  });
});
