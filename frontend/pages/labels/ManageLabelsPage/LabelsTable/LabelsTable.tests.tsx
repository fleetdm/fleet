import React from "react";

import { screen } from "@testing-library/react";
import { noop } from "lodash";

import createMockUser from "__mocks__/userMock";
import { createMockLabel } from "__mocks__/labelsMock";
import { createCustomRenderer } from "test/test-utils";
import LabelsTable from "./LabelsTable";

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
        label_membership_type: "dynamic",
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
        label_membership_type: "manual",
      }),
      createMockLabel({
        id: 5,
        name: "Custom label 3",
        label_type: "regular",
        description: "Third custom label",
        label_membership_type: "host_vitals",
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

    // Custom labels should be visible, each with the regular copy and the full name in a tooltip
    expect(screen.queryAllByText("Custom label 1")).toHaveLength(2);
    expect(screen.queryAllByText("First custom label")).toHaveLength(2);
    expect(screen.queryAllByText("Dynamic")).toHaveLength(1);

    expect(screen.queryAllByText("Custom label 2")).toHaveLength(2);
    expect(screen.queryAllByText("Second custom label")).toHaveLength(2);
    expect(screen.queryAllByText("Manual")).toHaveLength(1);

    expect(screen.queryAllByText("Custom label 3")).toHaveLength(2);
    expect(screen.queryAllByText("Third custom label")).toHaveLength(2);
    expect(screen.queryAllByText("Host vitals")).toHaveLength(1);

    // Builtin labels should not be visible
    expect(screen.queryByText("All hosts")).not.toBeInTheDocument();
    expect(screen.queryByText("macOS")).not.toBeInTheDocument();

    // Table headers should be visible
    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Description")).toBeInTheDocument();
    expect(screen.getByText("Type")).toBeInTheDocument();
  });
});
