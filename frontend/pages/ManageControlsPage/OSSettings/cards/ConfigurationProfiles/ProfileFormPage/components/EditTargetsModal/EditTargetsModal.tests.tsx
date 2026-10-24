import { screen } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { createMockLabel } from "__mocks__/labelsMock";
import labelsAPI from "services/entities/labels";
import { createCustomRenderer } from "test/test-utils";

import EditTargetsModal, { IProfileTargets } from "./EditTargetsModal";

const allHosts: IProfileTargets = {
  targetType: "All hosts",
  includeMode: "any",
  includeLabels: {},
  excludeLabels: {},
};

const makeRenderer = (gitOpsModeEnabled = false) =>
  createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        config: createMockConfig({
          gitops: {
            ...createMockConfig().gitops,
            gitops_mode_enabled: gitOpsModeEnabled,
            repository_url: gitOpsModeEnabled ? "https://example.com/repo" : "",
          },
        }),
      },
    },
  });

const renderModal = (
  props: Partial<React.ComponentProps<typeof EditTargetsModal>> = {},
  render = makeRenderer()
) => {
  const onDone = jest.fn();
  const onExit = jest.fn();
  const results = render(
    <EditTargetsModal
      teamId={1}
      targets={allHosts}
      onAddLabel={jest.fn()}
      onDone={onDone}
      onExit={onExit}
      {...props}
    />
  );
  return { ...results, onDone, onExit };
};

describe("EditTargetsModal", () => {
  beforeEach(() => {
    jest.spyOn(labelsAPI, "summary").mockResolvedValue({
      labels: [
        createMockLabel({ id: 1, name: "Apple Silicon macOS hosts" }),
        createMockLabel({ id: 2, name: "Engineering" }),
        createMockLabel({ id: 3, name: "macOS", label_type: "builtin" }),
      ],
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("returns the chosen labels on Done", async () => {
    const { user, onDone } = renderModal();

    await user.click(screen.getByLabelText("Custom"));
    await user.click(
      await screen.findByRole("checkbox", { name: "Apple Silicon macOS hosts" })
    );
    await user.click(screen.getByLabelText("All"));
    await user.click(screen.getByRole("button", { name: "Done" }));

    expect(labelsAPI.summary).toHaveBeenCalledWith(1);
    expect(onDone).toHaveBeenCalledWith({
      targetType: "Custom",
      includeMode: "all",
      includeLabels: { "Apple Silicon macOS hosts": true },
      excludeLabels: {},
    });
  });

  it("only filters the labels when Enter is pressed in the search", async () => {
    const { user, onDone } = renderModal();

    await user.click(screen.getByLabelText("Custom"));
    await user.click(
      await screen.findByRole("checkbox", { name: "Apple Silicon macOS hosts" })
    );
    await user.type(
      screen.getByPlaceholderText("Search labels"),
      "Engineering{Enter}"
    );

    expect(onDone).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
  });

  it("discards changes on Cancel", async () => {
    const { user, onDone, onExit } = renderModal();

    await user.click(screen.getByLabelText("Custom"));
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(onExit).toHaveBeenCalledTimes(1);
    expect(onDone).not.toHaveBeenCalled();
  });

  it("asks for a label before a custom target is done", async () => {
    const { user, onDone } = renderModal();

    await user.click(screen.getByLabelText("Custom"));
    await user.click(screen.getByRole("button", { name: "Done" }));
    expect(screen.getByText("Select at least one label")).toBeInTheDocument();
    expect(onDone).not.toHaveBeenCalled();

    // going back to All hosts makes the error irrelevant
    await user.click(screen.getByLabelText("All hosts"));
    expect(
      screen.queryByText("Select at least one label")
    ).not.toBeInTheDocument();
  });

  it("starts from the current targets", async () => {
    renderModal({
      targets: {
        targetType: "Custom",
        includeMode: "any",
        includeLabels: {},
        excludeLabels: { Engineering: true },
      },
    });

    expect(screen.getByLabelText("Custom")).toBeChecked();
    // the include tab shows the exclude tab's labels as unavailable
    expect(
      await screen.findByRole("checkbox", { name: "Engineering" })
    ).toHaveAttribute("aria-disabled", "true");
  });

  it("shows the targets read-only in GitOps mode", async () => {
    renderModal({}, makeRenderer(true));

    expect(screen.getByLabelText("Custom")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Done" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeEnabled();
  });

  it("shows the targets read-only when disabled", () => {
    renderModal({ disabled: true });

    expect(screen.getByLabelText("All hosts")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Done" })).toBeDisabled();
  });
});
