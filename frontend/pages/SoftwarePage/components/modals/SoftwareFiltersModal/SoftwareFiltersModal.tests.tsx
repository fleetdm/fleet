import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import { noop } from "lodash";

import {
  SEVERITY_RANGE_INVALID_MSG,
  SEVERITY_SCORE_RANGE_ERROR,
} from "components/SeverityFilter";

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

const setUpModal = (props = {}) =>
  renderWithSetup(
    <SoftwareFiltersModal
      onExit={noop}
      onSubmit={noop}
      vulnFilters={vulnFiltersDefault}
      isPremiumTier
      {...props}
    />
  );

// react-select renders its options as plain divs, so target them by the testid
// the shared custom Option component sets rather than by role.
const selectSeverity = async (
  user: ReturnType<typeof renderWithSetup>["user"],
  label: string
) => {
  await user.click(screen.getByRole("combobox", { name: "Severity" }));
  const option = screen
    .getAllByTestId("dropdown-option")
    .find((el) => el.textContent?.startsWith(label));
  if (!option) {
    throw new Error(`No severity option matching "${label}"`);
  }
  await user.click(option);
};

describe("SoftwareFiltersModal component", () => {
  it("renders modal title and form fields", () => {
    renderModal();
    expect(screen.getByText(/Filters/i)).toBeInTheDocument();
    expect(screen.getByText(/Vulnerable software/i)).toBeInTheDocument();
    expect(
      screen.getByRole("combobox", { name: "Severity" })
    ).toBeInTheDocument();
    expect(screen.getByText(/Has known exploit/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Apply/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Cancel/i })).toBeInTheDocument();
  });

  it("disables input fields when Vulnerable software is off", () => {
    renderModal({
      vulnFilters: { ...vulnFiltersDefault, minCvssScore: 2 },
    });
    expect(screen.getByRole("combobox", { name: "Severity" })).toBeDisabled();
    expect(screen.getByLabelText(/Min score/i)).toBeDisabled();
    expect(screen.getByLabelText(/Max score/i)).toBeDisabled();
    const checkbox = screen.getByRole("checkbox", {
      name: /hasKnownExploit/i,
    });
    expect(checkbox).toHaveAttribute("aria-disabled", "true");
  });

  it("enables input fields when Vulnerable software is toggled on", async () => {
    const { user } = setUpModal({
      vulnFilters: { ...vulnFiltersDefault, minCvssScore: 2 },
    });
    await user.click(screen.getByRole("switch"));
    expect(screen.getByRole("combobox", { name: "Severity" })).toBeEnabled();
    expect(screen.getByLabelText(/Min score/i)).toBeEnabled();
    expect(screen.getByLabelText(/Max score/i)).toBeEnabled();
    const checkbox = screen.getByRole("checkbox", {
      name: /hasKnownExploit/i,
    });
    expect(checkbox).toHaveAttribute("aria-disabled", "false");
  });

  // Per frontend/docs/patterns.md#data-validation: Apply stays enabled, errors
  // appear on blur of a dirty field or on a submit attempt, and clear on focus.
  describe("validation lifecycle", () => {
    // Once an error replaces the label, getByLabelText can no longer find the
    // field, so address the score inputs by name.
    const scoreInput = (field: "minScore" | "maxScore") =>
      document.querySelector(`input[name="${field}"]`) as HTMLInputElement;

    const setUpCustom = async () => {
      const rendered = setUpModal();
      await rendered.user.click(screen.getByRole("switch"));
      await selectSeverity(rendered.user, "Custom severity");
      return rendered;
    };

    it("keeps Apply enabled with invalid input", async () => {
      const { user } = await setUpCustom();

      await user.type(scoreInput("minScore"), "11");

      expect(screen.getByRole("button", { name: /Apply/i })).toBeEnabled();
    });

    it("shows nothing while typing, then the error on blur", async () => {
      const { user } = await setUpCustom();

      await user.type(scoreInput("minScore"), "11");
      expect(
        screen.queryByText(SEVERITY_SCORE_RANGE_ERROR)
      ).not.toBeInTheDocument();

      await user.tab();
      expect(screen.getByText(SEVERITY_SCORE_RANGE_ERROR)).toBeInTheDocument();
    });

    it("clears the error on focus, restoring the label", async () => {
      const { user } = await setUpCustom();

      await user.type(scoreInput("minScore"), "11");
      await user.tab();
      expect(screen.getByText(SEVERITY_SCORE_RANGE_ERROR)).toBeInTheDocument();

      await user.click(scoreInput("minScore"));
      expect(
        screen.queryByText(SEVERITY_SCORE_RANGE_ERROR)
      ).not.toBeInTheDocument();
      expect(screen.getByLabelText(/Min score/i)).toBeInTheDocument();
    });

    it("shows an inverted range on the maximum, the dependent field", async () => {
      const { user } = await setUpCustom();

      await user.type(scoreInput("minScore"), "7");
      await user.type(scoreInput("maxScore"), "3");
      await user.tab();

      expect(screen.getByText(SEVERITY_RANGE_INVALID_MSG)).toBeInTheDocument();
      // Attached to Max, so Min keeps its label.
      expect(screen.getByLabelText(/Min score/i)).toBeInTheDocument();
    });

    it("surfaces every error and does not submit when Apply is clicked", async () => {
      const onSubmitSpy = jest.fn();
      const rendered = setUpModal({ onSubmit: onSubmitSpy });
      const { user } = rendered;
      await user.click(screen.getByRole("switch"));
      await selectSeverity(user, "Custom severity");

      await user.type(scoreInput("minScore"), "11");
      await user.click(screen.getByRole("button", { name: /Apply/i }));

      expect(onSubmitSpy).not.toHaveBeenCalled();
      expect(screen.getByText(SEVERITY_SCORE_RANGE_ERROR)).toBeInTheDocument();
    });

    it("drops the error when Vulnerable software is switched off", async () => {
      const onSubmitSpy = jest.fn();
      const { user } = await setUpCustom();

      await user.type(scoreInput("minScore"), "11");
      await user.tab();
      expect(screen.getByText(SEVERITY_SCORE_RANGE_ERROR)).toBeInTheDocument();

      // Disabled fields are never in an error state, and the scores are not
      // submitted while the filter is off.
      await user.click(screen.getByRole("switch"));
      expect(
        screen.queryByText(SEVERITY_SCORE_RANGE_ERROR)
      ).not.toBeInTheDocument();
      expect(onSubmitSpy).not.toHaveBeenCalled();
    });

    it("clears a score error when the severity option changes", async () => {
      const { user } = await setUpCustom();

      await user.type(scoreInput("minScore"), "11");
      await user.tab();
      expect(screen.getByText(SEVERITY_SCORE_RANGE_ERROR)).toBeInTheDocument();

      await selectSeverity(user, "High severity");

      expect(
        screen.queryByText(SEVERITY_SCORE_RANGE_ERROR)
      ).not.toBeInTheDocument();
    });
  });

  it("calls onSubmit with the correct values when form is valid", async () => {
    const onSubmitSpy = jest.fn();
    const { user } = setUpModal({ onSubmit: onSubmitSpy });
    await user.click(screen.getByRole("switch"));
    await selectSeverity(user, "Custom severity");

    const minInput = screen.getByLabelText(/Min score/i);
    const maxInput = screen.getByLabelText(/Max score/i);

    await user.type(minInput, "3");
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

  it("submits the bounds of the selected preset", async () => {
    const onSubmitSpy = jest.fn();
    const { user } = setUpModal({ onSubmit: onSubmitSpy });
    await user.click(screen.getByRole("switch"));
    await selectSeverity(user, "High severity");

    await user.click(screen.getByRole("button", { name: /Apply/i }));

    expect(onSubmitSpy).toHaveBeenCalledWith({
      vulnerable: true,
      exploit: undefined,
      minCvssScore: 7,
      maxCvssScore: 8.9,
    });
  });

  it("submits no CVSS bounds for Any severity", async () => {
    const onSubmitSpy = jest.fn();
    const { user } = setUpModal({
      onSubmit: onSubmitSpy,
      vulnFilters: {
        ...vulnFiltersDefault,
        minCvssScore: 7,
        maxCvssScore: 8.9,
      },
    });
    await user.click(screen.getByRole("switch"));
    await selectSeverity(user, "Any severity");

    await user.click(screen.getByRole("button", { name: /Apply/i }));

    expect(onSubmitSpy).toHaveBeenCalledWith({
      vulnerable: true,
      exploit: undefined,
      minCvssScore: undefined,
      maxCvssScore: undefined,
    });
  });

  it("submits a lone 0 minimum rather than clearing the filter", async () => {
    const onSubmitSpy = jest.fn();
    const { user } = setUpModal({ onSubmit: onSubmitSpy });
    await user.click(screen.getByRole("switch"));
    await selectSeverity(user, "Custom severity");

    // A lone "0" in Min used to collapse the control to "Any severity" while
    // still submitting min_cvss_score=0.
    await user.type(screen.getByLabelText(/Min score/i), "0");
    await user.click(screen.getByRole("button", { name: /Apply/i }));

    expect(onSubmitSpy).toHaveBeenCalledWith({
      vulnerable: true,
      exploit: undefined,
      minCvssScore: 0,
      maxCvssScore: undefined,
    });
  });

  it("hides the severity filter on Fleet Free", () => {
    renderModal({ isPremiumTier: false });

    expect(
      screen.queryByRole("combobox", { name: "Severity" })
    ).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/Min score/i)).not.toBeInTheDocument();
  });
});
