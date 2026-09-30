import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import React, { useState } from "react";

import { renderWithSetup } from "test/test-utils";

import { SEVERITY_SCORE_RANGE_ERROR, validateSeverityScores } from "./helpers";
import SeverityFilter, { ISeverityFilterValue } from "./SeverityFilter";

const ControlledSeverityFilter = ({
  severity = "any",
  minScore = "",
  maxScore = "",
  onChange,
  validate = false,
}: Partial<ISeverityFilterValue> & {
  onChange?: (next: ISeverityFilterValue) => void;
  validate?: boolean;
}) => {
  const [value, setValue] = useState<ISeverityFilterValue>({
    severity,
    minScore,
    maxScore,
  });

  return (
    <SeverityFilter
      {...value}
      errors={validate ? validateSeverityScores(value) : undefined}
      onChange={(next) => {
        setValue(next);
        onChange?.(next);
      }}
    />
  );
};

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

const getMinInput = () => screen.getByLabelText(/Min score/i);
const getMaxInput = () => screen.getByLabelText(/Max score/i);
const querySeverityLabel = (label: RegExp) => screen.queryByText(label);

describe("SeverityFilter", () => {
  it("renders the dropdown with its tooltip label and help text", () => {
    render(
      <SeverityFilter severity="any" minScore="" maxScore="" onChange={noop} />
    );

    expect(screen.getByText("Severity")).toBeInTheDocument();
    expect(
      screen.getByText(
        "CVSS scores (v3) range from 0.0 to 10.0 in 0.1 increments."
      )
    ).toBeInTheDocument();
  });

  it("always shows Min score and Max score, for every severity selection", async () => {
    const { user } = renderWithSetup(<ControlledSeverityFilter />);

    expect(getMinInput()).toBeInTheDocument();
    expect(getMaxInput()).toBeInTheDocument();

    await selectSeverity(user, "Critical severity");
    expect(getMinInput()).toBeInTheDocument();
    expect(getMaxInput()).toBeInTheDocument();

    await selectSeverity(user, "Any severity");
    expect(getMinInput()).toBeInTheDocument();
    expect(getMaxInput()).toBeInTheDocument();
  });

  it("does not offer Custom severity as a selectable menu option", async () => {
    const { user } = renderWithSetup(<ControlledSeverityFilter />);

    await user.click(screen.getByRole("combobox", { name: "Severity" }));

    const options = screen
      .getAllByTestId("dropdown-option")
      .map((el) => el.textContent);
    expect(options.some((text) => text?.startsWith("Custom"))).toBe(false);
  });

  describe("option labels", () => {
    it("shows the plain label for the active selection", () => {
      const { rerender } = render(
        <SeverityFilter
          severity="critical"
          minScore="9"
          maxScore="10"
          onChange={noop}
        />
      );

      expect(screen.getByText("Critical severity")).toBeInTheDocument();
      expect(
        screen.queryByText("Critical (9.0 to 10)")
      ).not.toBeInTheDocument();

      rerender(
        <SeverityFilter
          severity="custom"
          minScore="2.5"
          maxScore="6"
          onChange={noop}
        />
      );

      expect(screen.getByText("Custom severity")).toBeInTheDocument();
      expect(screen.queryByText("Custom (2.5 to 6)")).not.toBeInTheDocument();
    });

    // Custom severity is a derived value (see severityForRange) rather than a
    // choice a user clicks, but the dropdown still has to display it as the
    // current value once a parent hands it in — e.g. after Apply + reopen
    // with a range that matches no preset.
    it("shows Custom severity as the current value without listing it as an option", async () => {
      const { user } = renderWithSetup(
        <ControlledSeverityFilter
          severity="custom"
          minScore="4.5"
          maxScore="8.5"
        />
      );

      expect(screen.getByText("Custom severity")).toBeInTheDocument();

      await user.click(screen.getByRole("combobox", { name: "Severity" }));
      const options = screen
        .getAllByTestId("dropdown-option")
        .map((el) => el.textContent);
      expect(options.some((text) => text?.startsWith("Custom"))).toBe(false);
    });

    it("opens the menu focused on the current selection, not the first row", async () => {
      const onChange = jest.fn();
      const { user } = renderWithSetup(
        <SeverityFilter
          severity="critical"
          minScore="9"
          maxScore="10"
          onChange={onChange}
        />
      );

      await user.click(screen.getByRole("combobox", { name: "Severity" }));
      await user.keyboard("{Enter}");

      expect(onChange).toHaveBeenCalledWith({
        severity: "critical",
        minScore: "9",
        maxScore: "10",
      });
    });
  });

  it("populates min/max from the selected preset", async () => {
    const onChange = jest.fn();
    const { user } = renderWithSetup(
      <ControlledSeverityFilter onChange={onChange} />
    );

    await selectSeverity(user, "High severity");

    expect(onChange).toHaveBeenLastCalledWith({
      severity: "high",
      minScore: "7",
      maxScore: "8.9",
    });
  });

  it("clears min/max when Any severity is selected", async () => {
    const onChange = jest.fn();
    const { user } = renderWithSetup(
      <ControlledSeverityFilter
        severity="high"
        minScore="7"
        maxScore="8.9"
        onChange={onChange}
      />
    );

    await selectSeverity(user, "Any severity");

    expect(onChange).toHaveBeenLastCalledWith({
      severity: "any",
      minScore: "",
      maxScore: "",
    });
  });

  // Typing a range that doesn't match the selected preset does not flip the
  // dropdown to Custom mid-edit — that only happens once a parent re-derives
  // severity from the saved range (see the Dev note on #52474: only show
  // Custom severity after the user saves and reopens the modal).
  it("keeps the selected preset's label while its score inputs are edited", async () => {
    const { user } = renderWithSetup(
      <ControlledSeverityFilter severity="medium" minScore="4" maxScore="6.9" />
    );

    await user.clear(getMinInput());
    await user.type(getMinInput(), "5");

    expect(screen.getByText("Medium severity")).toBeInTheDocument();
    expect(getMinInput()).toHaveValue(5);
  });

  describe("typing never changes the dropdown", () => {
    const renderCustom = () =>
      renderWithSetup(<ControlledSeverityFilter severity="custom" />);

    const expectStillCustom = () => {
      expect(querySeverityLabel(/^Custom\b/)).toBeInTheDocument();
      expect(
        querySeverityLabel(/^(Any|Critical|High|Medium|Low)\b/)
      ).toBeNull();
      expect(getMinInput()).toBeInTheDocument();
      expect(getMaxInput()).toBeInTheDocument();
    };

    const typedCases: { label: string; min?: string; max?: string }[] = [
      { label: "a lone 0 in Min", min: "0" },
      { label: "a lone 9 in Min", min: "9" },
      { label: "a lone 10 in Max", max: "10" },
      { label: "a range matching a preset", min: "7", max: "8.9" },
      { label: "decimals, one character at a time", min: "0.5", max: "9.5" },
    ];

    it.each(typedCases)(
      "keeps Custom while typing $label",
      async ({ min, max }) => {
        const { user } = renderCustom();

        if (min) await user.type(getMinInput(), min);
        if (max) await user.type(getMaxInput(), max);

        expectStillCustom();
        expect(getMinInput()).toHaveValue(min ? Number(min) : null);
        expect(getMaxInput()).toHaveValue(max ? Number(max) : null);
      }
    );

    it("keeps Custom when both fields are cleared", async () => {
      const { user } = renderWithSetup(
        <ControlledSeverityFilter
          severity="custom"
          minScore="7"
          maxScore="8.9"
        />
      );

      await user.clear(getMinInput());
      await user.clear(getMaxInput());

      expectStillCustom();
      expect(getMinInput()).toHaveValue(null);
      expect(getMaxInput()).toHaveValue(null);
    });
  });

  it("surfaces per-field errors from the validation helper", async () => {
    const { user } = renderWithSetup(
      <ControlledSeverityFilter severity="custom" validate />
    );

    await user.type(getMinInput(), "5.55");

    expect(screen.getByText(SEVERITY_SCORE_RANGE_ERROR)).toBeInTheDocument();
  });

  it("disables the dropdown and the score inputs", () => {
    render(
      <SeverityFilter
        severity="custom"
        minScore=""
        maxScore=""
        onChange={noop}
        disabled
      />
    );

    expect(screen.getByRole("combobox", { name: "Severity" })).toBeDisabled();
    expect(getMinInput()).toBeDisabled();
    expect(getMaxInput()).toBeDisabled();
  });
});
