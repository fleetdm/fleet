import { screen } from "@testing-library/react";
import React from "react";

import { renderWithSetup } from "test/test-utils";

import ActionButton from "./ActionButton";

describe("ActionButton", () => {
  it("calls onClick with the target ids when enabled", async () => {
    const onClick = jest.fn();
    const { user } = renderWithSetup(
      <ActionButton
        name="run-script"
        buttonText="Run script"
        onClick={onClick}
        targetIds={[1, 2]}
      />
    );

    await user.click(screen.getByRole("button", { name: "Run script" }));

    expect(onClick).toHaveBeenCalledWith([1, 2]);
  });

  it("renders a disabled button that ignores clicks when isDisabled", async () => {
    const onClick = jest.fn();
    const { user } = renderWithSetup(
      <ActionButton
        name="run-script"
        buttonText="Run script"
        onClick={onClick}
        isDisabled
        tooltipContent="Choose different filters to run a script."
      />
    );

    const button = screen.getByRole("button", { name: "Run script" });
    expect(button).toBeDisabled();

    await user.click(button);
    expect(onClick).not.toHaveBeenCalled();
  });

  it("can't be activated from the keyboard when isDisabled", async () => {
    const onClick = jest.fn();
    const { user } = renderWithSetup(
      <>
        <button type="button">Before</button>
        <ActionButton
          name="run-script"
          buttonText="Run script"
          onClick={onClick}
          isDisabled
          tooltipContent="Choose different filters to run a script."
        />
      </>
    );

    await user.tab();
    await user.tab();
    expect(
      screen.getByRole("button", { name: "Run script" })
    ).not.toHaveFocus();

    await user.keyboard("{Enter}");
    expect(onClick).not.toHaveBeenCalled();
  });
});
