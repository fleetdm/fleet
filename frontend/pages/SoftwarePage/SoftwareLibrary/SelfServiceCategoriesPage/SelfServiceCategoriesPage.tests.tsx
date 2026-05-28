import React from "react";
import { screen, waitFor, within } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mockServer from "test/mock-server";
import createMockUser from "__mocks__/userMock";
import { createMockTeamSummary } from "__mocks__/teamMock";
import {
  addSelfServiceCategoryConflictHandler,
  addSelfServiceCategoryHandler,
  deleteSelfServiceCategoryHandler,
  editSelfServiceCategoryConflictHandler,
  editSelfServiceCategoryHandler,
  emptySelfServiceCategoriesHandler,
  listSelfServiceCategoriesHandler,
} from "test/handlers/self-service-categories-handlers";

import SelfServiceCategoriesPage from "./SelfServiceCategoriesPage";

const baseProps = {
  router: createMockRouter(),
  location: {
    pathname: "/software/library/categories",
    search: "?fleet_id=1",
    query: { fleet_id: "1" },
    hash: "",
  },
};

const renderFlash = jest.fn();

const mockTeam = createMockTeamSummary({ id: 1, name: "Workstations" });

const premiumAdminContext = {
  app: {
    isPremiumTier: true,
    isGlobalAdmin: true,
    currentUser: createMockUser({ global_role: "admin" }),
    availableTeams: [mockTeam],
    setCurrentTeam: jest.fn(),
  },
  notification: { renderFlash, hideFlash: jest.fn() },
};

// Returns the currently open modal element scoped for `within(...)` queries.
// Modal renders its title as a span (not a heading) so role-based queries
// can't locate it; falling back to the modal-container class since only one
// modal is open at a time in these tests.
const MODAL_SELECTOR = ".modal__modal_container";
const getOpenModal = async () => {
  await waitFor(() => {
    if (!document.querySelector(MODAL_SELECTOR)) {
      throw new Error("Modal not yet rendered");
    }
  });
  return document.querySelector(MODAL_SELECTOR) as HTMLElement;
};

describe("SelfServiceCategoriesPage", () => {
  beforeEach(() => {
    renderFlash.mockClear();
  });

  it("renders the premium gate on Fleet Free", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: false,
          isGlobalAdmin: true,
          currentUser: createMockUser({ global_role: "admin" }),
        },
        notification: { renderFlash, hideFlash: jest.fn() },
      },
    });

    render(<SelfServiceCategoriesPage {...baseProps} />);

    expect(
      screen.getByText("This feature is included in Fleet Premium.")
    ).toBeInTheDocument();
  });

  it("renders the empty state with Add button when canManage", async () => {
    mockServer.use(emptySelfServiceCategoriesHandler);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    render(<SelfServiceCategoriesPage {...baseProps} />);

    expect(
      await screen.findByText("No self-service categories")
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Add category to group your software and scripts in self-service."
      )
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Add category" })
    ).toBeInTheDocument();
  });

  it("renders the empty state without Add button for non-managers", async () => {
    mockServer.use(emptySelfServiceCategoriesHandler);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: false,
          isGlobalMaintainer: false,
          isTeamAdmin: false,
          isTeamMaintainer: false,
          currentUser: createMockUser({ global_role: "observer" }),
          availableTeams: [mockTeam],
          setCurrentTeam: jest.fn(),
        },
        notification: { renderFlash, hideFlash: jest.fn() },
      },
    });

    render(<SelfServiceCategoriesPage {...baseProps} />);

    expect(
      await screen.findByText("No self-service categories")
    ).toBeInTheDocument();
    expect(
      screen.getByText("No self-service categories are available.")
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Add category" })
    ).not.toBeInTheDocument();
  });

  it("renders the populated list with category names", async () => {
    mockServer.use(listSelfServiceCategoriesHandler());
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    render(<SelfServiceCategoriesPage {...baseProps} />);

    expect(await screen.findByText("🌎 Browsers")).toBeInTheDocument();
    expect(screen.getByText("👬 Communication")).toBeInTheDocument();
    expect(screen.getByText("🧰 Developer tools")).toBeInTheDocument();
  });

  it("hides edit/delete actions for observers", async () => {
    mockServer.use(listSelfServiceCategoriesHandler());
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: false,
          isGlobalMaintainer: false,
          isTeamAdmin: false,
          isTeamMaintainer: false,
          currentUser: createMockUser({ global_role: "observer" }),
          availableTeams: [mockTeam],
          setCurrentTeam: jest.fn(),
        },
        notification: { renderFlash, hideFlash: jest.fn() },
      },
    });

    render(<SelfServiceCategoriesPage {...baseProps} />);

    await screen.findByText("🌎 Browsers");
    expect(
      screen.queryByRole("button", { name: /^Edit / })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /^Delete / })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Add category/ })
    ).not.toBeInTheDocument();
  });

  it("creates a category on Add submit", async () => {
    mockServer.use(
      emptySelfServiceCategoriesHandler,
      addSelfServiceCategoryHandler
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SelfServiceCategoriesPage {...baseProps} />);

    await user.click(
      await screen.findByRole("button", { name: "Add category" })
    );
    const modal = await getOpenModal();

    await user.type(within(modal).getByLabelText("Name"), "🌎 Browsers");
    await user.click(within(modal).getByRole("button", { name: /^Add$/ }));

    await waitFor(() => {
      expect(renderFlash).toHaveBeenCalledWith(
        "success",
        "Successfully added self-service category."
      );
    });
  });

  it("shows inline 409 error on duplicate name", async () => {
    mockServer.use(
      emptySelfServiceCategoriesHandler,
      addSelfServiceCategoryConflictHandler
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SelfServiceCategoriesPage {...baseProps} />);

    await user.click(
      await screen.findByRole("button", { name: "Add category" })
    );
    const modal = await getOpenModal();

    await user.type(within(modal).getByLabelText("Name"), "🌎 Browsers");
    await user.click(within(modal).getByRole("button", { name: /^Add$/ }));

    expect(
      await within(modal).findByText(
        "A self-service category with this name already exists in this fleet."
      )
    ).toBeInTheDocument();
    expect(renderFlash).not.toHaveBeenCalled();
  });

  it("shows inline 409 error on duplicate name when editing", async () => {
    mockServer.use(
      listSelfServiceCategoriesHandler([{ id: 1, name: "🌎 Browsers" }]),
      editSelfServiceCategoryConflictHandler
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SelfServiceCategoriesPage {...baseProps} />);

    await screen.findByText("🌎 Browsers");
    await user.click(screen.getByRole("button", { name: "Edit 🌎 Browsers" }));

    const modal = await getOpenModal();
    const input = within(modal).getByLabelText("Name");
    await user.clear(input);
    await user.type(input, "👬 Communication");
    await user.click(within(modal).getByRole("button", { name: /Save/ }));

    expect(
      await within(modal).findByText(
        "A self-service category with this name already exists in this fleet."
      )
    ).toBeInTheDocument();
    expect(renderFlash).not.toHaveBeenCalled();
  });

  it("closes the Add modal when Cancel is clicked", async () => {
    mockServer.use(emptySelfServiceCategoriesHandler);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SelfServiceCategoriesPage {...baseProps} />);

    await user.click(
      await screen.findByRole("button", { name: "Add category" })
    );
    const modal = await getOpenModal();

    await user.click(within(modal).getByRole("button", { name: /Cancel/ }));
    await waitFor(() => {
      expect(document.querySelector(".modal__modal_container")).toBeNull();
    });
  });

  it("closes the Edit modal when Cancel is clicked", async () => {
    mockServer.use(
      listSelfServiceCategoriesHandler([{ id: 1, name: "🌎 Browsers" }])
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SelfServiceCategoriesPage {...baseProps} />);

    await screen.findByText("🌎 Browsers");
    await user.click(screen.getByRole("button", { name: "Edit 🌎 Browsers" }));
    const modal = await getOpenModal();

    await user.click(within(modal).getByRole("button", { name: /Cancel/ }));
    await waitFor(() => {
      expect(document.querySelector(".modal__modal_container")).toBeNull();
    });
  });

  it("closes the Delete modal when Cancel is clicked", async () => {
    mockServer.use(
      listSelfServiceCategoriesHandler([{ id: 1, name: "🛟 Support" }])
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SelfServiceCategoriesPage {...baseProps} />);

    await screen.findByText("🛟 Support");
    await user.click(screen.getByRole("button", { name: "Delete 🛟 Support" }));
    const modal = await getOpenModal();

    await user.click(within(modal).getByRole("button", { name: /Cancel/ }));
    await waitFor(() => {
      expect(document.querySelector(".modal__modal_container")).toBeNull();
    });
  });

  it("edits a category on Save", async () => {
    mockServer.use(
      listSelfServiceCategoriesHandler([{ id: 1, name: "🌎 Browsers" }]),
      editSelfServiceCategoryHandler
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SelfServiceCategoriesPage {...baseProps} />);

    await screen.findByText("🌎 Browsers");
    await user.click(screen.getByRole("button", { name: "Edit 🌎 Browsers" }));

    const modal = await getOpenModal();
    const input = within(modal).getByLabelText("Name");
    await user.clear(input);
    await user.type(input, "🌍 Browsers (EU)");
    await user.click(within(modal).getByRole("button", { name: /Save/ }));

    await waitFor(() => {
      expect(renderFlash).toHaveBeenCalledWith(
        "success",
        "Successfully updated self-service category."
      );
    });
  });

  it("deletes a category on confirm", async () => {
    mockServer.use(
      listSelfServiceCategoriesHandler([{ id: 1, name: "🛟 Support" }]),
      deleteSelfServiceCategoryHandler
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SelfServiceCategoriesPage {...baseProps} />);

    await screen.findByText("🛟 Support");
    await user.click(screen.getByRole("button", { name: "Delete 🛟 Support" }));

    const modal = await getOpenModal();
    expect(
      within(modal).getByText(
        "The category will be removed from all associated software."
      )
    ).toBeInTheDocument();

    await user.click(within(modal).getByRole("button", { name: /^Delete$/ }));

    await waitFor(() => {
      expect(renderFlash).toHaveBeenCalledWith(
        "success",
        "Successfully deleted self-service category."
      );
    });
  });
});
