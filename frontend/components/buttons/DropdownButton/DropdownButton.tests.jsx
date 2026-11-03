import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

import { DropdownButton } from "./DropdownButton";

describe("DropdownButton - component", () => {
  it("calls the clicked item's onClick attribute", () => {
    const optionSpy = jest.fn();
    const dropdownOptions = [
      { label: "btn1", onClick: noop },
      { label: "btn2", onClick: optionSpy },
    ];

    render(
      <DropdownButton options={dropdownOptions}>New Button</DropdownButton>
    );

    fireEvent.click(screen.getByText("New Button"));

    fireEvent.click(screen.getByRole("button", { name: "btn2" }));
    expect(optionSpy).toHaveBeenCalled();
  });

  it("explains a disabled GitOps-compatible option with the GitOps tooltip", async () => {
    // GitOps mode on, so a disabled GitOps-compatible option renders its tooltip.
    const renderWithGitOps = createCustomRenderer({
      context: {
        app: {
          config: {
            gitops: { gitops_mode_enabled: true, repository_url: "a.b.cc" },
          },
        },
      },
    });
    const { user } = renderWithGitOps(
      <DropdownButton
        options={[
          {
            label: "Rename fleet",
            onClick: noop,
            disabled: true,
            gitOpsModeCompatible: true,
          },
        ]}
      >
        More options
      </DropdownButton>
    );

    await user.hover(screen.getByRole("button", { name: "Rename fleet" }));
    await waitFor(() => {
      expect(screen.getByRole("tooltip")).toBeInTheDocument();
    });
  });

  it("does not wrap a disabled GitOps-compatible option when GitOps mode is off", () => {
    const { container } = render(
      <DropdownButton
        options={[
          {
            label: "Rename fleet",
            onClick: noop,
            disabled: true,
            gitOpsModeCompatible: true,
          },
        ]}
      >
        More options
      </DropdownButton>
    );

    expect(container.querySelector(".gitops-mode-tooltip-wrapper")).toBeNull();
    expect(screen.getByRole("button", { name: "Rename fleet" })).toBeDisabled();
  });

  it("does not wrap an enabled GitOps-compatible option in GitOps mode", () => {
    const renderWithGitOps = createCustomRenderer({
      context: {
        app: {
          config: {
            gitops: { gitops_mode_enabled: true, repository_url: "a.b.cc" },
          },
        },
      },
    });
    const { container } = renderWithGitOps(
      <DropdownButton
        options={[
          {
            label: "Rename fleet",
            onClick: noop,
            disabled: false,
            gitOpsModeCompatible: true,
          },
        ]}
      >
        More options
      </DropdownButton>
    );

    expect(container.querySelector(".gitops-mode-tooltip-wrapper")).toBeNull();
    expect(screen.getByRole("button", { name: "Rename fleet" })).toBeEnabled();
  });
});
