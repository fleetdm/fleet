import { screen, waitFor, within } from "@testing-library/react";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

import ActionButtons, { IActionButtonProps } from "./ActionButtons";

// GitOps mode on, so GitOpsModeTooltipWrapper renders its tooltip for a
// disabled GitOps-compatible action.
const render = createCustomRenderer({
  context: {
    app: {
      config: {
        gitops: { gitops_mode_enabled: true, repository_url: "a.b.cc" },
      },
    },
  },
});

const action = (overrides: Partial<IActionButtonProps> = {}) => ({
  type: "secondary" as const,
  label: "Manage enroll secrets",
  buttonVariant: "secondary" as const,
  onClick: jest.fn(),
  gitOpsModeCompatible: true,
  ...overrides,
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
  it("disables a disabled action on both surfaces", () => {
    const { container } = render(
      <ActionButtons baseClass="test" actions={[action({ disabled: true })]} />
    );

    expect(
      inButtons(container).getByRole("button", {
        name: "Manage enroll secrets",
      })
    ).toBeDisabled();
    expect(
      inDropdown(container).getByRole("button", {
        name: "Manage enroll secrets",
      })
    ).toBeDisabled();
  });

  it("enables an action that is not disabled on both surfaces, even in GitOps mode", () => {
    const { container } = render(
      <ActionButtons baseClass="test" actions={[action({ disabled: false })]} />
    );

    expect(
      inButtons(container).getByRole("button", {
        name: "Manage enroll secrets",
      })
    ).toBeEnabled();
    expect(
      inDropdown(container).getByRole("button", {
        name: "Manage enroll secrets",
      })
    ).toBeEnabled();
  });

  it("does not fire a disabled dropdown option's handler", async () => {
    const onClick = jest.fn();
    const { container, user } = render(
      <ActionButtons
        baseClass="test"
        actions={[action({ disabled: true, onClick })]}
      />
    );

    await user.click(
      inDropdown(container).getByRole("button", {
        name: "Manage enroll secrets",
      })
    );
    expect(onClick).not.toHaveBeenCalled();
  });

  it("fires an enabled dropdown option's handler", async () => {
    const onClick = jest.fn();
    const { container, user } = render(
      <ActionButtons
        baseClass="test"
        actions={[action({ disabled: false, onClick })]}
      />
    );

    await user.click(
      inDropdown(container).getByRole("button", {
        name: "Manage enroll secrets",
      })
    );
    expect(onClick).toHaveBeenCalled();
  });

  it("explains a disabled GitOps-compatible action with the GitOps tooltip", async () => {
    const { container, user } = render(
      <ActionButtons baseClass="test" actions={[action({ disabled: true })]} />
    );

    await user.hover(
      inButtons(container).getByRole("button", {
        name: "Manage enroll secrets",
      })
    );
    await waitFor(() => {
      expect(screen.getByRole("tooltip")).toBeInTheDocument();
    });
  });

  it("disables a disabled primary action", () => {
    render(
      <ActionButtons
        baseClass="test"
        actions={[
          action({ type: "primary", label: "Add hosts", disabled: true }),
        ]}
      />
    );

    expect(screen.getByRole("button", { name: "Add hosts" })).toBeDisabled();
  });

  it("does not render a hidden action", () => {
    render(
      <ActionButtons
        baseClass="test"
        actions={[action({ hideAction: true })]}
      />
    );

    expect(screen.queryByText("Manage enroll secrets")).toBeNull();
  });
});
