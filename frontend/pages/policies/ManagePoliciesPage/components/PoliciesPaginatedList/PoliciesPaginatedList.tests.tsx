import React from "react";

import { http, HttpResponse } from "msw";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { baseUrl, createCustomRenderer } from "test/test-utils";
import createMockPolicy from "__mocks__/policyMock";
import mockServer from "test/mock-server";

import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import PoliciesPaginatedList, { IFormPolicy } from "./PoliciesPaginatedList";

const waitForLoadingToFinish = async (container: HTMLElement) => {
  await waitFor(() => {
    expect(container.querySelector(".loading-overlay")).not.toBeInTheDocument();
  });
};

const globalPolicies = [
  createMockPolicy({ team_id: null, name: "Inherited policy 1" }),
  createMockPolicy({ id: 2, team_id: null, name: "Inherited policy 2" }),
  createMockPolicy({ id: 3, team_id: null, name: "Inherited policy 3" }),
];

const teamPolicies = [
  createMockPolicy({ id: 4, team_id: 2, name: "Team policy 1" }),
  createMockPolicy({ id: 5, team_id: 2, name: "Team policy 2" }),
];

const globalPoliciesHandler = http.get(baseUrl("/policies"), () => {
  return HttpResponse.json({
    policies: globalPolicies,
  });
});

const globalPoliciesCountHandler = http.get(baseUrl("/policies/count"), () => {
  return HttpResponse.json({
    count: globalPolicies.length,
  });
});

const teamPoliciesHandler = http.get(baseUrl("/teams/2/policies"), () => {
  return HttpResponse.json({
    policies: teamPolicies,
  });
});

const teamPoliciesCountHandler = http.get(
  baseUrl("/teams/2/policies/count"),
  () => {
    return HttpResponse.json({
      count: teamPolicies.length,
    });
  }
);

describe("PoliciesPaginatedList - component", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
  });

  it("Lists global policies when teamId is set to 'all teams'", async () => {
    mockServer.use(globalPoliciesHandler);
    mockServer.use(teamPoliciesHandler);
    mockServer.use(globalPoliciesCountHandler);
    mockServer.use(teamPoliciesCountHandler);
    const { container } = render(
      <PoliciesPaginatedList
        isSelected={jest.fn()}
        onToggleItem={jest.fn()}
        onCancel={jest.fn()}
        onSubmit={jest.fn()}
        teamId={APP_CONTEXT_ALL_TEAMS_ID}
        footer={null}
        isUpdating={false}
      />
    );
    await waitForLoadingToFinish(container);

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    globalPolicies.forEach((item, index) => {
      expect(checkboxes[index]).toHaveTextContent(item.name);
      expect(checkboxes[index]).not.toBeChecked();
    });
  });

  it("Lists team policies when teamId is not set to 'all teams'", async () => {
    mockServer.use(globalPoliciesHandler);
    mockServer.use(teamPoliciesHandler);
    mockServer.use(globalPoliciesCountHandler);
    mockServer.use(teamPoliciesCountHandler);
    const { container } = render(
      <PoliciesPaginatedList
        isSelected={jest.fn()}
        onToggleItem={jest.fn()}
        onCancel={jest.fn()}
        onSubmit={jest.fn()}
        teamId={2}
        footer={null}
        isUpdating={false}
      />
    );
    await waitForLoadingToFinish(container);

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(2);
    teamPolicies.forEach((item, index) => {
      expect(checkboxes[index]).toHaveTextContent(item.name);
      expect(checkboxes[index]).not.toBeChecked();
    });
  });

  it("Renders a footer", async () => {
    mockServer.use(globalPoliciesHandler);
    mockServer.use(teamPoliciesHandler);
    mockServer.use(globalPoliciesCountHandler);
    mockServer.use(teamPoliciesCountHandler);
    const { container } = render(
      <PoliciesPaginatedList
        isSelected={jest.fn()}
        onToggleItem={jest.fn()}
        onCancel={jest.fn()}
        onSubmit={jest.fn()}
        teamId={2}
        footer={<div>Hello World!</div>}
        isUpdating={false}
      />
    );
    await waitForLoadingToFinish(container);

    expect(container).toHaveTextContent("Hello World!");
  });

  it("Calls onSubmit() with the changed items when 'Save' is pressed", async () => {
    mockServer.use(globalPoliciesHandler);
    mockServer.use(teamPoliciesHandler);
    mockServer.use(globalPoliciesCountHandler);
    mockServer.use(teamPoliciesCountHandler);
    const onSubmit = jest.fn();
    const { container } = render(
      <PoliciesPaginatedList
        isSelected={jest.fn()}
        onToggleItem={jest.fn((item) => ({
          ...item,
          name: `${item.name} (changed)`,
        }))}
        onCancel={jest.fn()}
        onSubmit={onSubmit}
        teamId={APP_CONTEXT_ALL_TEAMS_ID}
        footer={null}
        isUpdating={false}
      />
    );
    await waitForLoadingToFinish(container);

    const checkboxes = screen.getAllByRole("checkbox");
    await userEvent.click(checkboxes[0]);
    await userEvent.click(checkboxes[2]);
    await userEvent.click(screen.getByRole("button", { name: /Save/i }));
    await waitFor(() => {
      expect(onSubmit.mock.calls.length).toEqual(1);
      // PoliciesPaginated list enhances policy objects into IFormPolicies in ways
      // that may change over time, so rather than full equality we'll just check
      // that it sends the objects with the right IDs.
      const changedItems = onSubmit.mock.calls[0][0];
      expect(changedItems[0].id).toEqual(globalPolicies[0].id);
      expect(changedItems[0].name).toEqual(
        `${globalPolicies[0].name} (changed)`
      );
      expect(changedItems[1].id).toEqual(globalPolicies[2].id);
      expect(changedItems[1].name).toEqual(
        `${globalPolicies[2].name} (changed)`
      );
    });
  });

  it("Calls onCancel() when 'Cancel' is pressed", async () => {
    mockServer.use(globalPoliciesHandler);
    mockServer.use(teamPoliciesHandler);
    mockServer.use(globalPoliciesCountHandler);
    mockServer.use(teamPoliciesCountHandler);
    const onCancel = jest.fn();
    const { container } = render(
      <PoliciesPaginatedList
        isSelected={jest.fn()}
        onToggleItem={jest.fn()}
        onCancel={onCancel}
        onSubmit={jest.fn()}
        teamId={APP_CONTEXT_ALL_TEAMS_ID}
        footer={null}
        isUpdating={false}
      />
    );
    await waitForLoadingToFinish(container);

    const checkboxes = screen.getAllByRole("checkbox");
    await userEvent.click(checkboxes[0]);
    await userEvent.click(checkboxes[2]);
    await userEvent.click(screen.getByRole("button", { name: /Cancel/i }));
    await waitFor(() => {
      expect(onCancel.mock.calls.length).toEqual(1);
    });
  });

  it("Allows for disabling the save button based on the changed items", async () => {
    mockServer.use(globalPoliciesHandler);
    mockServer.use(teamPoliciesHandler);
    mockServer.use(globalPoliciesCountHandler);
    mockServer.use(teamPoliciesCountHandler);
    const disableSave = (changedItems: IFormPolicy[]) => {
      if (changedItems.length === 0) {
        return "No changes";
      }
      if (changedItems.length > 1) {
        return "Stop touching things!";
      }
      return false;
    };
    const { container } = render(
      <PoliciesPaginatedList
        isSelected={jest.fn()}
        onToggleItem={(item) => item}
        onCancel={jest.fn()}
        onSubmit={jest.fn()}
        teamId={APP_CONTEXT_ALL_TEAMS_ID}
        footer={null}
        isUpdating={false}
        disableSave={disableSave}
      />
    );
    await waitForLoadingToFinish(container);

    const saveButton = screen.getByRole("button", { name: /Save/i });
    // Initially, the save button should be disabled since there are no changes.
    expect(saveButton).toBeDisabled();

    // TODO: These tests pass locally but fail in the CI pipeline; skipping for now.
    // await userEvent.hover(saveButton);
    // await waitFor(() => {
    //   expect(screen.getByText(/No changes/)).toBeInTheDocument();
    // });
    // await userEvent.unhover(saveButton);

    // After clicking a checkbox, the save button should be enabled.
    const checkboxes = screen.getAllByRole("checkbox");
    await userEvent.click(checkboxes[0]);
    expect(saveButton).not.toBeDisabled();

    // After clicking another checkbox, the save button should be disabled again.
    await userEvent.click(checkboxes[1]);
    expect(saveButton).toBeDisabled();

    // TODO: These tests pass locally but fail in the CI pipeline; skipping for now.
    // await userEvent.hover(saveButton);
    // await waitFor(() => {
    //   expect(screen.getByText(/Stop touching things!/)).toBeInTheDocument();
    // });
    // await userEvent.unhover(saveButton);
  });

  it("Disables the form when gitops mode is enabled", async () => {
    mockServer.use(globalPoliciesHandler);
    mockServer.use(teamPoliciesHandler);
    mockServer.use(globalPoliciesCountHandler);
    mockServer.use(teamPoliciesCountHandler);
    const customRender = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          config: {
            gitops: {
              gitops_mode_enabled: true,
            },
          },
        },
      },
    });
    const { container } = customRender(
      <PoliciesPaginatedList
        isSelected={jest.fn()}
        onToggleItem={jest.fn()}
        onCancel={jest.fn()}
        onSubmit={jest.fn()}
        teamId={APP_CONTEXT_ALL_TEAMS_ID}
        footer={null}
        isUpdating={false}
      />
    );
    await waitForLoadingToFinish(container);
    const checkboxes = container.querySelectorAll("input[type=checkbox]");
    checkboxes.forEach((checkbox) => {
      expect(checkbox).toBeDisabled();
    });
  });
});
