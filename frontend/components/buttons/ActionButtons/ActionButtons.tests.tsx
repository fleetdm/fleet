import { screen, within } from "@testing-library/react";
import React from "react";

import { IGitOpsExceptions } from "interfaces/config";
import { createCustomRenderer } from "test/test-utils";

import ActionButtons, { IActionButtonProps } from "./ActionButtons";

const GITOPS_ON = {
  gitops_mode_enabled: true,
  repository_url: "a.b.cc",
};

const renderWithGitOps = (
  gitops: Record<string, unknown>,
  exceptions?: IGitOpsExceptions
) =>
  createCustomRenderer({
    context: {
      app: {
        isGlobalAdmin: true,
        config: {
          gitops: { ...gitops, ...(exceptions ? { exceptions } : {}) },
        },
      },
    },
  });

const manageSecrets = (onClick = jest.fn()): IActionButtonProps => ({
  type: "secondary",
  label: "Manage enroll secrets",
  buttonVariant: "secondary",
  onClick,
  gitOpsModeCompatible: true,
  entityType: "secrets",
});

const renameFleet = (onClick = jest.fn()): IActionButtonProps => ({
  type: "secondary",
  label: "Rename fleet",
  buttonVariant: "secondary",
  onClick,
  gitOpsModeCompatible: true,
});

// The same actions render twice: as buttons for wide viewports and as
// "More options" dropdown items for narrow ones. CSS decides which is visible,
// so both are always in the DOM and each surface is asserted on separately.
const inButtons = (container: HTMLElement) =>
  within(
    container.querySelector(".action-buttons__secondary-buttons") as HTMLElement
  );

const inDropdown = (container: HTMLElement) =>
  within(container.querySelector(".dropdown-button__options") as HTMLElement);

describe("ActionButtons", () => {
  describe("wide-viewport buttons", () => {
    it("keeps a GitOps-compatible action enabled when its entity type is excepted", () => {
      const render = renderWithGitOps(GITOPS_ON, {
        labels: false,
        software: false,
        secrets: true,
      });

      const { container } = render(
        <ActionButtons baseClass="test" actions={[manageSecrets()]} />
      );

      expect(
        inButtons(container).getByRole("button", {
          name: "Manage enroll secrets",
        })
      ).toBeEnabled();
    });

    it("disables a GitOps-compatible action when its entity type is not excepted", () => {
      const render = renderWithGitOps(GITOPS_ON, {
        labels: true,
        software: true,
        secrets: false,
      });

      const { container } = render(
        <ActionButtons baseClass="test" actions={[manageSecrets()]} />
      );

      expect(
        inButtons(container).getByRole("button", {
          name: "Manage enroll secrets",
        })
      ).toBeDisabled();
    });

    it("disables a GitOps-compatible action that names no entity type", () => {
      const render = renderWithGitOps(GITOPS_ON, {
        labels: true,
        software: true,
        secrets: true,
      });

      const { container } = render(
        <ActionButtons baseClass="test" actions={[renameFleet()]} />
      );

      expect(
        inButtons(container).getByRole("button", { name: "Rename fleet" })
      ).toBeDisabled();
    });

    it("leaves actions enabled when GitOps mode is off", () => {
      const render = renderWithGitOps({
        gitops_mode_enabled: false,
        repository_url: "",
      });

      const { container } = render(
        <ActionButtons
          baseClass="test"
          actions={[manageSecrets(), renameFleet()]}
        />
      );

      expect(
        inButtons(container).getByRole("button", {
          name: "Manage enroll secrets",
        })
      ).toBeEnabled();
      expect(
        inButtons(container).getByRole("button", { name: "Rename fleet" })
      ).toBeEnabled();
    });

    it("does not render a hidden action", () => {
      const render = renderWithGitOps(GITOPS_ON, {
        labels: false,
        software: false,
        secrets: true,
      });

      render(
        <ActionButtons
          baseClass="test"
          actions={[{ ...manageSecrets(), hideAction: true }]}
        />
      );

      expect(screen.queryByText("Manage enroll secrets")).toBeNull();
    });
  });

  describe("narrow-viewport dropdown", () => {
    it("disables an option whose entity type is not excepted, and does not fire its handler", async () => {
      const onClick = jest.fn();
      const render = renderWithGitOps(GITOPS_ON, {
        labels: true,
        software: true,
        secrets: false,
      });

      const { container, user } = render(
        <ActionButtons baseClass="test" actions={[manageSecrets(onClick)]} />
      );

      const option = inDropdown(container).getByRole("button", {
        name: "Manage enroll secrets",
      });
      expect(option).toBeDisabled();

      await user.click(option);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("disables an option that names no entity type", () => {
      const render = renderWithGitOps(GITOPS_ON, {
        labels: true,
        software: true,
        secrets: true,
      });

      const { container } = render(
        <ActionButtons baseClass="test" actions={[renameFleet()]} />
      );

      expect(
        inDropdown(container).getByRole("button", { name: "Rename fleet" })
      ).toBeDisabled();
    });

    it("keeps an option enabled when its entity type is excepted, and fires its handler", async () => {
      const onClick = jest.fn();
      const render = renderWithGitOps(GITOPS_ON, {
        labels: false,
        software: false,
        secrets: true,
      });

      const { container, user } = render(
        <ActionButtons baseClass="test" actions={[manageSecrets(onClick)]} />
      );

      const option = inDropdown(container).getByRole("button", {
        name: "Manage enroll secrets",
      });
      expect(option).toBeEnabled();

      await user.click(option);
      expect(onClick).toHaveBeenCalled();
    });

    it("leaves options enabled when GitOps mode is off", () => {
      const render = renderWithGitOps({
        gitops_mode_enabled: false,
        repository_url: "",
      });

      const { container } = render(
        <ActionButtons
          baseClass="test"
          actions={[manageSecrets(), renameFleet()]}
        />
      );

      expect(
        inDropdown(container).getByRole("button", {
          name: "Manage enroll secrets",
        })
      ).toBeEnabled();
      expect(
        inDropdown(container).getByRole("button", { name: "Rename fleet" })
      ).toBeEnabled();
    });
  });
});
