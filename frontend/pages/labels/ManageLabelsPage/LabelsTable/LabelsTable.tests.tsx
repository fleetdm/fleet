import React from "react";

import RoutingProvider from "context/routing";

import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";

import createMockUser from "__mocks__/userMock";
import { createMockLabel } from "__mocks__/labelsMock";
import { createCustomRenderer } from "test/test-utils";
import LabelsTable from "./LabelsTable";
import { Router } from "react-router";
import { AppWrapper } from "router";

describe("LabelsTable", () => {
  it("Renders empty state when only builtin labels are provided", () => {
    const builtinLabels = [
      createMockLabel({ id: 1, name: "All hosts", label_type: "builtin" }),
      createMockLabel({ id: 2, name: "macOS", label_type: "builtin" }),
      createMockLabel({ id: 3, name: "Ubuntu", label_type: "builtin" }),
    ];

    const mockUser = createMockUser();

    const render = createCustomRenderer();
    render(
      <LabelsTable
        labels={builtinLabels}
        onClickAction={noop}
        currentUser={mockUser}
      />
    );

    expect(screen.getByText("No labels")).toBeInTheDocument();
    expect(
      screen.getByText("Labels you create will appear here.")
    ).toBeInTheDocument();
    expect(screen.queryByText("All hosts")).not.toBeInTheDocument();
    expect(screen.queryByText("macOS")).not.toBeInTheDocument();
    expect(screen.queryByText("Ubuntu")).not.toBeInTheDocument();
  });

  it("Renders only custom labels when custom and builtin labels are provided", () => {
    const labels = [
      createMockLabel({
        id: 1,
        name: "All hosts",
        label_type: "builtin",
      }),
      createMockLabel({
        id: 2,
        name: "Custom label 1",
        label_type: "regular",
        description: "First custom label",
      }),
      createMockLabel({
        id: 3,
        name: "macOS",
        label_type: "builtin",
      }),
      createMockLabel({
        id: 4,
        name: "Custom label 2",
        label_type: "regular",
        description: "Second custom label",
      }),
    ];

    const mockUser = createMockUser();

    const render = createCustomRenderer();
    render(
      <LabelsTable
        labels={labels}
        onClickAction={noop}
        currentUser={mockUser}
      />
    );

    // Custom labels should be visible
    expect(screen.getByText("Custom label 1")).toBeInTheDocument();
    expect(screen.getByText("Custom label 2")).toBeInTheDocument();
    expect(screen.getByText("First custom label")).toBeInTheDocument();
    expect(screen.getByText("Second custom label")).toBeInTheDocument();

    // Builtin labels should not be visible
    expect(screen.queryByText("All hosts")).not.toBeInTheDocument();
    expect(screen.queryByText("macOS")).not.toBeInTheDocument();

    // Table headers should be visible
    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Description")).toBeInTheDocument();
    expect(screen.getByText("Type")).toBeInTheDocument();
  });

  it.only("Includes edit and delete actions for global admins", async () => {
    const customLabel = createMockLabel({
      id: 1,
      name: "Custom label",
      label_type: "regular",
      author_id: 999, // Different from the admin user
    });

    const globalAdminUser = createMockUser({
      id: 1,
      global_role: "admin",
    });

    const render = createCustomRenderer();
    const { user } = render(
      <LabelsTable
        labels={[customLabel]}
        onClickAction={noop}
        currentUser={globalAdminUser}
      />
    );

    const row = screen.getByText("Custom label");
    await user.hover(row);

    await waitFor(() => {
      expect(screen.getByText("Actions")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Actions")).catch(() => {});

    await waitFor(() => {
      expect(screen.getByText("View all hosts")).toBeInTheDocument();
      expect(screen.getByText("Edit")).toBeInTheDocument();
      expect(screen.getByText("Delete")).toBeInTheDocument();
    });
  });

  it("Includes edit and delete actions for a team admin on a label they authored, but not on a label they did not", async () => {
    const teamAdminUser = createMockUser({
      id: 5,
      global_role: null,
      teams: [
        {
          id: 1,
          name: "Team 1",
          role: "admin",
          description: "",
          agent_options: undefined,
          user_count: 1,
          host_count: 1,
          secrets: [],
        },
      ],
    });

    const authoredLabel = createMockLabel({
      id: 1,
      name: "My label",
      label_type: "regular",
      author_id: 5, // Same as team admin user
    });

    const notAuthoredLabel = createMockLabel({
      id: 2,
      name: "Someone else's label",
      label_type: "regular",
      author_id: 999, // Different from team admin user
    });

    const render = createCustomRenderer();
    const { user, rerender } = render(
      <LabelsTable
        labels={[authoredLabel]}
        onClickAction={noop}
        currentUser={teamAdminUser}
      />
    );

    // Test authored label - should have edit and delete actions
    let actionsButton = screen.getByText("Actions");
    await user.click(actionsButton);

    // Wait for dropdown menu to appear
    await waitFor(() => {
      expect(screen.getByText("View all hosts")).toBeInTheDocument();
      expect(screen.getByText("Edit")).toBeInTheDocument();
      expect(screen.getByText("Delete")).toBeInTheDocument();
    });

    // Close dropdown by clicking outside
    await user.click(document.body);

    // Wait for dropdown to close
    await waitFor(() => {
      expect(screen.queryByText("View all hosts")).not.toBeInTheDocument();
    });

    // Re-render with not authored label
    rerender(
      <LabelsTable
        labels={[notAuthoredLabel]}
        onClickAction={noop}
        currentUser={teamAdminUser}
      />
    );

    // Test not authored label - should only have view action
    actionsButton = screen.getByText("Actions");
    await user.click(actionsButton);

    // Wait for dropdown menu to appear
    await waitFor(() => {
      expect(screen.getByText("View all hosts")).toBeInTheDocument();
      expect(screen.queryByText("Edit")).not.toBeInTheDocument();
      expect(screen.queryByText("Delete")).not.toBeInTheDocument();
    });
  });
});
