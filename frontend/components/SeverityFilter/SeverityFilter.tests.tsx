import React, { useState } from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";

import { renderWithSetup } from "test/test-utils";

import SeverityFilter, { ISeverityFilterValue } from "./SeverityFilter";
import { SEVERITY_SCORE_RANGE_ERROR, validateSeverityScores } from "./helpers";

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

  it("hides the score inputs for every preset and shows them only for Custom", async () => {
    const { user } = renderWithSetup(<ControlledSeverityFilter />);

    expect(screen.queryByLabelText(/Min score/i)).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/Max score/i)).not.toBeInTheDocument();

    await selectSeverity(user, "Critical severity");
    expect(screen.queryByLabelText(/Min score/i)).not.toBeInTheDocument();

    await selectSeverity(user, "High severity");
    expect(screen.queryByLabelText(/Min score/i)).not.toBeInTheDocument();

    await selectSeverity(user, "Medium severity");
    expect(screen.queryByLabelText(/Min score/i)).not.toBeInTheDocument();

    await selectSeverity(user, "Low severity");
    expect(screen.queryByLabelText(/Min score/i)).not.toBeInTheDocument();

    await selectSeverity(user, "Custom severity");
    expect(getMinInput()).toBeInTheDocument();
    expect(getMaxInput()).toBeInTheDocument();
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

    await selectSeverity(user, "Custom severity");
    expect(onChange).toHaveBeenLastCalledWith({
      severity: "custom",
      minScore: "",
      maxScore: "",
    });
    expect(getMinInput()).toHaveValue(null);
    expect(getMaxInput()).toHaveValue(null);
  });

  it("starts Custom empty from a preset that was seeded, not clicked", async () => {
    const { user } = renderWithSetup(
      <ControlledSeverityFilter severity="medium" minScore="4" maxScore="6.9" />
    );

    await selectSeverity(user, "Custom severity");

    expect(getMinInput()).toHaveValue(null);
    expect(getMaxInput()).toHaveValue(null);
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

    await selectSeverity(user, "Custom severity");
    expect(getMinInput()).toHaveValue(null);
    expect(getMaxInput()).toHaveValue(null);
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
