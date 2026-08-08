import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import ProfileAdvancedOptions from "./ProfileAdvancedOptions";

const ACTIVATION = JSON.stringify({ Type: "com.apple.activation.simple" });

describe("ProfileAdvancedOptions", () => {
  it("keeps the activation editor collapsed until advanced options are revealed", async () => {
    render(
      <ProfileAdvancedOptions
        customActivation={ACTIVATION}
        onChangeCustomActivation={jest.fn()}
      />
    );

    const toggle = screen.getByRole("button", { name: "Advanced options" });

    expect(screen.queryByText("Custom activation")).not.toBeInTheDocument();
    // the chevron conveys the state visually; aria-expanded conveys it to
    // assistive tech.
    expect(toggle).toHaveAttribute("aria-expanded", "false");

    await userEvent.click(toggle);

    expect(screen.getByText("Custom activation")).toBeInTheDocument();
    expect(toggle).toHaveAttribute("aria-expanded", "true");
  });

  it("stays expandable but read-only when the caller disables editing", async () => {
    // GitOps mode is read-only, not hidden: an admin still needs to be able to
    // open the section and read the activation Fleet is serving.
    const { container } = render(
      <ProfileAdvancedOptions
        customActivation={ACTIVATION}
        onChangeCustomActivation={jest.fn()}
        readOnly
      />
    );

    const toggle = screen.getByRole("button", { name: "Advanced options" });
    expect(toggle).toBeEnabled();

    await userEvent.click(toggle);

    expect(screen.getByText("Custom activation")).toBeInTheDocument();
    expect(container.querySelector("textarea")).toHaveAttribute("readonly");
  });
});
