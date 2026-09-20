import { screen, within } from "@testing-library/react";
import React from "react";

import { IGitOpsExceptions } from "interfaces/config";
import { createCustomRenderer } from "test/test-utils";

import ActionButtons, { IActionButtonProps } from "./ActionButtons";

const renderInGitOpsMode = (exceptions: IGitOpsExceptions) =>
  createCustomRenderer({
    context: {
      app: {
        isGlobalAdmin: true,
        config: {
          gitops: {
            gitops_mode_enabled: true,
            repository_url: "a.b.cc",
            exceptions,
          },
        },
      },
    },
  });

const manageSecrets: IActionButtonProps = {
  type: "secondary",
  label: "Manage enroll secrets",
  buttonVariant: "secondary",
  onClick: jest.fn(),
  gitOpsModeCompatible: true,
  entityType: "secrets",
};

const renameFleet: IActionButtonProps = {
  type: "secondary",
  label: "Rename fleet",
  buttonVariant: "secondary",
  onClick: jest.fn(),
  gitOpsModeCompatible: true,
};

// The same actions are rendered twice: as buttons for wide viewports and as
// "More options" dropdown items for narrow ones. Only the buttons are under test.
const getSecondaryButton = (container: HTMLElement, label: string) =>
  within(
    container.querySelector(".action-buttons__secondary-buttons") as HTMLElement
  ).getByRole("button", { name: label });

describe("ActionButtons", () => {
  it("keeps a GitOps-compatible action enabled when its entity type is excepted", () => {
    const render = renderInGitOpsMode({
      labels: false,
      software: false,
      secrets: true,
    });

    const { container } = render(
      <ActionButtons baseClass="test" actions={[manageSecrets]} />
    );

    expect(
      getSecondaryButton(container, "Manage enroll secrets")
    ).toBeEnabled();
  });

  it("disables a GitOps-compatible action when its entity type is not excepted", () => {
    const render = renderInGitOpsMode({
      labels: true,
      software: true,
      secrets: false,
    });

    const { container } = render(
      <ActionButtons baseClass="test" actions={[manageSecrets]} />
    );

    expect(
      getSecondaryButton(container, "Manage enroll secrets")
    ).toBeDisabled();
  });

  it("disables a GitOps-compatible action that names no entity type", () => {
    const render = renderInGitOpsMode({
      labels: true,
      software: true,
      secrets: true,
    });

    const { container } = render(
      <ActionButtons baseClass="test" actions={[renameFleet]} />
    );

    expect(getSecondaryButton(container, "Rename fleet")).toBeDisabled();
  });

  it("leaves actions enabled when GitOps mode is off", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          config: {
            gitops: { gitops_mode_enabled: false, repository_url: "" },
          },
        },
      },
    });

    const { container } = render(
      <ActionButtons baseClass="test" actions={[manageSecrets, renameFleet]} />
    );

    expect(
      getSecondaryButton(container, "Manage enroll secrets")
    ).toBeEnabled();
    expect(getSecondaryButton(container, "Rename fleet")).toBeEnabled();
  });

  it("does not render a hidden action", () => {
    const render = renderInGitOpsMode({
      labels: false,
      software: false,
      secrets: true,
    });

    render(
      <ActionButtons
        baseClass="test"
        actions={[{ ...manageSecrets, hideAction: true }]}
      />
    );

    expect(screen.queryByText("Manage enroll secrets")).toBeNull();
  });
});
