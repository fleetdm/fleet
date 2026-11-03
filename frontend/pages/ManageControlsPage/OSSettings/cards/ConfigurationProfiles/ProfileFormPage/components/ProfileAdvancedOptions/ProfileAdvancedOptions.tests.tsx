import { screen } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { createCustomRenderer } from "test/test-utils";

import ProfileAdvancedOptions from "./ProfileAdvancedOptions";

const render = createCustomRenderer({
  context: { app: { config: createMockConfig() } },
});

const defaultProps = {
  isOpen: true,
  onToggle: jest.fn(),
  canSelfService: true,
  selfService: false,
  hidden: false,
  onChange: jest.fn(),
};

const hiddenCheckbox = () =>
  screen.getByRole("checkbox", { name: "Hide from end user" });

describe("ProfileAdvancedOptions", () => {
  it("shows only the toggle while closed", async () => {
    const onToggle = jest.fn();
    const { user } = render(
      <ProfileAdvancedOptions
        {...defaultProps}
        isOpen={false}
        onToggle={onToggle}
      />
    );

    expect(screen.queryByText("Deploy")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("checkbox", { name: "Hide from end user" })
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Advanced options" }));
    expect(onToggle).toHaveBeenCalledTimes(1);
  });

  it("clears hidden when a .mobileconfig is switched to self-service", async () => {
    const onChange = jest.fn();
    const { user } = render(
      <ProfileAdvancedOptions {...defaultProps} hidden onChange={onChange} />
    );

    await user.click(screen.getByRole("combobox"));
    // the options have no option role
    await user.click(await screen.findByText("End user initiated (manual)"));

    expect(onChange).toHaveBeenCalledWith({ selfService: true, hidden: false });
  });

  it("disables Hide from end user for a self-service profile", () => {
    render(<ProfileAdvancedOptions {...defaultProps} selfService />);

    expect(screen.getByText("End user initiated (manual)")).toBeInTheDocument();
    expect(hiddenCheckbox()).toHaveAttribute("aria-disabled", "true");
  });

  it("is force-only for other types, which can still be hidden", async () => {
    const onChange = jest.fn();
    const { user } = render(
      <ProfileAdvancedOptions
        {...defaultProps}
        canSelfService={false}
        onChange={onChange}
      />
    );

    expect(screen.getByText("Force install")).toBeInTheDocument();
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();

    await user.click(hiddenCheckbox());
    expect(onChange).toHaveBeenCalledWith({ hidden: true });
  });

  it("disables both controls when asked to", () => {
    render(<ProfileAdvancedOptions {...defaultProps} disabled />);

    expect(screen.getByRole("combobox")).toBeDisabled();
    expect(hiddenCheckbox()).toHaveAttribute("aria-disabled", "true");
  });
});
