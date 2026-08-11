import React from "react";
import { SingleValue } from "react-select-5";

import { IInputFieldParseTarget } from "interfaces/form_field";

import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import InputField from "components/forms/fields/InputField";
import TooltipWrapper from "components/TooltipWrapper";

import {
  CUSTOM_SEVERITY_VALUE,
  getSeverityBand,
  getSeverityOption,
  ISeverityFieldErrors,
  ISeverityFilterValue,
  SEVERITY_DROPDOWN_OPTIONS,
  SEVERITY_HELP_TEXT,
  SeverityScoreField,
} from "./helpers";

const baseClass = "severity-filter";

export type { ISeverityFilterValue };

interface ISeverityFilterProps extends ISeverityFilterValue {
  onChange: (next: ISeverityFilterValue) => void;
  disabled?: boolean;
  errors?: ISeverityFieldErrors;
  onScoreBlur?: (field: SeverityScoreField) => void;
  onScoreFocus?: (field: SeverityScoreField) => void;
}

const SeverityFilter = ({
  severity,
  minScore,
  maxScore,
  onChange,
  disabled = false,
  errors,
  onScoreBlur,
  onScoreFocus,
}: ISeverityFilterProps) => {
  const onChangeSeverity = (selected: SingleValue<CustomOptionType>) => {
    const option = getSeverityOption(selected?.value);
    if (!option) {
      return;
    }
    const band = getSeverityBand(option.value);
    onChange({
      severity: option.value,
      minScore: band ? String(band.min) : "",
      maxScore: band ? String(band.max) : "",
    });
  };

  const onScoreChange = ({ name, value }: IInputFieldParseTarget) => {
    onChange({ severity, minScore, maxScore, [name]: value as string });
  };

  const renderLabel = () => (
    <TooltipWrapper
      tipContent={
        <>
          The worst case impact across different environments
          <br />
          (CVSS version 3.x base score).
        </>
      }
      clickable={false}
    >
      Severity
    </TooltipWrapper>
  );

  return (
    <div className={baseClass}>
      <DropdownWrapper
        name="severity-filter"
        ariaLabel="Severity"
        label={renderLabel()}
        options={SEVERITY_DROPDOWN_OPTIONS}
        value={severity}
        onChange={onChangeSeverity}
        placeholder="Any severity"
        className={`${baseClass}__dropdown`}
        isDisabled={disabled}
        helpText={SEVERITY_HELP_TEXT}
      />
      {severity === CUSTOM_SEVERITY_VALUE && (
        <div className={`${baseClass}__cvss-range`}>
          <InputField
            label="Min score"
            onChange={onScoreChange}
            onBlur={() => onScoreBlur?.("minScore")}
            onFocus={() => onScoreFocus?.("minScore")}
            name="minScore"
            value={minScore}
            disabled={disabled}
            type="number"
            placeholder="0.0"
            min={0}
            max={10}
            step={0.1}
            parseTarget
            error={errors?.minScore}
          />
          <InputField
            label="Max score"
            onChange={onScoreChange}
            onBlur={() => onScoreBlur?.("maxScore")}
            onFocus={() => onScoreFocus?.("maxScore")}
            name="maxScore"
            value={maxScore}
            disabled={disabled}
            type="number"
            placeholder="10.0"
            min={0}
            max={10}
            step={0.1}
            parseTarget
            error={errors?.maxScore}
          />
        </div>
      )}
    </div>
  );
};

export default SeverityFilter;
