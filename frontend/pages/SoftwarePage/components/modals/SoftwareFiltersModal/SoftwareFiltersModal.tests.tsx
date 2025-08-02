import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import { noop } from "lodash";

import SoftwareFiltersModal from "./SoftwareFiltersModal";

const vulnFiltersDefault = {
  vulnerable: false,
  exploit: false,
  minCvssScore: undefined,
  maxCvssScore: undefined,
};

const renderModal = (props = {}) =>
  render(
    <SoftwareFiltersModal
      onExit={noop}
      onSubmit={noop}
      vulnFilters={vulnFiltersDefault}
      isPremiumTier
      {...props}
    />
  );

describe("SoftwareFiltersModal component", () => {
  it("renders modal title and form fields", () => {
    renderModal();
    expect(screen.getByText(/Filters/i)).toBeInTheDocument();
    expect(screen.getByText(/Vulnerable software/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Min score/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Max score/i)).toBeInTheDocument();
    expect(screen.getByText(/Has known exploit/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Apply/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Cancel/i })).toBeInTheDocument();
  });

  it("disables input fields when Vulnerable software is off", () => {
    renderModal();
    expect(screen.getByRole("combobox")).toBeDisabled(); // Disabled dropdown
    expect(screen.getByLabelText(/Min score/i)).toBeDisabled();
    expect(screen.getByLabelText(/Max score/i)).toBeDisabled();
    const checkbox = screen.getByRole("checkbox", {
      name: /hasKnownExploit/i,
    });
    expect(checkbox).toHaveAttribute("aria-disabled", "true");
  });

  it("enables input fields when Vulnerable software is toggled on", async () => {
    const { user } = renderWithSetup(
      <SoftwareFiltersModal
        onExit={noop}
        onSubmit={noop}
        vulnFilters={vulnFiltersDefault}
        isPremiumTier
      />
    );
    await user.click(screen.getByRole("switch"));
    expect(screen.getByRole("combobox")).toBeEnabled(); // Enabled dropdown
    expect(screen.getByLabelText(/Min score/i)).toBeEnabled();
    expect(screen.getByLabelText(/Max score/i)).toBeEnabled();
    const checkbox = screen.getByRole("checkbox", {
      name: /hasKnownExploit/i,
    });
    expect(checkbox).toHaveAttribute("aria-disabled", "false");
  });

  it("shows validation errors for non-numeric or out-of-range scores", async () => {
    const { user } = renderWithSetup(
      <SoftwareFiltersModal
        onExit={noop}
        onSubmit={noop}
        vulnFilters={vulnFiltersDefault}
        isPremiumTier
      />
    );
    await user.click(screen.getByRole("switch"));
    const minInput = screen.getByLabelText(/Min score/i);
    const maxInput = screen.getByLabelText(/Max score/i);

    // Out of range
    await user.type(minInput, "11");
    expect(screen.getByText(/Must be from 0-10/i)).toBeInTheDocument();

    await user.clear(minInput);
    await user.type(minInput, "-1");
    expect(screen.getByText(/Must be from 0-10/i)).toBeInTheDocument();

    await user.clear(minInput);
    await user.type(minInput, "5.55");
    expect(screen.getByText(/Must be from 0-10/i)).toBeInTheDocument();

    // Valid value, but min > max
    await user.clear(minInput);
    await user.type(minInput, "7");
    await user.clear(maxInput);
    await user.type(maxInput, "3");

    const applyButton = screen.getByRole("button", { name: /Apply/i });

    await user.hover(applyButton);
    expect(
      screen.getByText(/Minimum CVSS score cannot be greater/i)
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Apply/i })).toBeDisabled();
  });

  it("calls onSubmit with the correct values when form is valid", async () => {
    const onSubmitSpy = jest.fn();
    const { user } = renderWithSetup(
      <SoftwareFiltersModal
        onExit={noop}
        onSubmit={onSubmitSpy}
        vulnFilters={vulnFiltersDefault}
        isPremiumTier
      />
    );
    await user.click(screen.getByRole("switch"));

    const minInput = screen.getByLabelText(/Min score/i);
    const maxInput = screen.getByLabelText(/Max score/i);

    await user.clear(minInput);
    await user.type(minInput, "3");
    await user.clear(maxInput);
    await user.type(maxInput, "8.5");

    // Enable "Has known exploit"
    await user.click(screen.getByText(/Has known exploit/i));

    // Submit
    await user.click(screen.getByRole("button", { name: /Apply/i }));

    expect(onSubmitSpy).toHaveBeenCalledWith({
      vulnerable: true,
      exploit: true,
      minCvssScore: 3,
      maxCvssScore: 8.5,
    });
  });

  it("shows and resets severity dropdown according to score inputs", async () => {
    const { user } = renderWithSetup(
      <SoftwareFiltersModal
        onExit={noop}
        onSubmit={noop}
        vulnFilters={vulnFiltersDefault}
        isPremiumTier
      />
    );
    await user.click(screen.getByRole("switch"));

    // If both min/max are cleared, selects "Any severity".
    const minInput = screen.getByLabelText(/Min score/i);
    const maxInput = screen.getByLabelText(/Max score/i);

    await user.clear(minInput);
    await user.clear(maxInput);

    // The severity select should now show "Any severity"
    expect(screen.getByText(/Any severity/i)).toBeInTheDocument();
  });
});
