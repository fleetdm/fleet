import React, { useState } from "react";

import { SingleValue } from "react-select-5";
import { IInputFieldParseTarget } from "interfaces/form_field";

import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Slider from "components/forms/fields/Slider";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import TooltipWrapper from "components/TooltipWrapper";
import {
  ANY_SEVERITY_OPTION,
  CUSTOM_SEVERITY_OPTION,
  findOptionBySeverityRange,
  ISoftwareVulnFiltersParams,
  SEVERITY_DROPDOWN_OPTIONS,
} from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/helpers";

const baseClass = "software-filters-modal";

interface ISoftwareFiltersModalProps {
  onExit: () => void;
  onSubmit: (vulnFilters: ISoftwareVulnFiltersParams) => void;
  vulnFilters: ISoftwareVulnFiltersParams;
  isPremiumTier: boolean;
}

type IFormData = {
  minScore: string;
  maxScore: string;
};

type IFormErrors = {
  minScore?: string;
  maxScore?: string;
  disableApplyButton?: React.ReactNode;
};

const hasAtMostOneDecimal = (n: number) =>
  Number.isFinite(n) && Number((n * 10).toFixed(0)) / 10 === n;

const isBetween0and10 = (n: number) => n >= 0 && n <= 10;

const validate = (data: IFormData): IFormErrors => {
  const errors: IFormErrors = {};
  const min = data.minScore ? parseFloat(data.minScore) : undefined;
  const max = data.maxScore ? parseFloat(data.maxScore) : undefined;

  if (data.minScore) {
    if (
      typeof min !== "number" ||
      !hasAtMostOneDecimal(min) ||
      !isBetween0and10(min)
    ) {
      errors.minScore = "Must be from 0-10 in 0.1 increments";
    }
  }
  if (data.maxScore) {
    if (
      max === undefined ||
      !hasAtMostOneDecimal(max) ||
      !isBetween0and10(max)
    ) {
      errors.maxScore = "Must be from 0-10 in 0.1 increments";
    }
  }
  if (
    data.minScore &&
    data.maxScore &&
    !errors.minScore &&
    !errors.maxScore &&
    min !== undefined &&
    max !== undefined &&
    min > max
  ) {
    errors.disableApplyButton = (
      <>
        Minimum CVSS score cannot be greater
        <br /> than the maximum CVSS score.
      </>
    );
  }
  return errors;
};

const SoftwareFiltersModal = ({
  onExit,
  onSubmit,
  vulnFilters,
  isPremiumTier,
}: ISoftwareFiltersModalProps) => {
  const [vulnSoftwareFilterEnabled, setVulnSoftwareFilterEnabled] = useState(
    vulnFilters.vulnerable || false
  );
  const [severity, setSeverity] = useState(
    findOptionBySeverityRange(
      vulnFilters.minCvssScore,
      vulnFilters.maxCvssScore
    )
  );
  // Unified form state:
  const [formData, setFormData] = useState<IFormData>({
    minScore: vulnFilters.minCvssScore?.toString() ?? "",
    maxScore: vulnFilters.maxCvssScore?.toString() ?? "",
  });
  const [hasKnownExploit, setHasKnownExploit] = useState(vulnFilters.exploit);
  const [formErrors, setFormErrors] = useState<IFormErrors>({});

  const onChangeSeverity = (
    selectedSeverity: SingleValue<CustomOptionType>
  ) => {
    const selectedOption = SEVERITY_DROPDOWN_OPTIONS.find(
      (option) => option.value === selectedSeverity?.value
    );
    if (selectedOption) {
      setSeverity(selectedOption);
      // Auto populate severity range except for custom severity
      if (selectedOption.value === "any") {
        setFormData({ minScore: "", maxScore: "" });
      } else if (selectedOption.value !== "custom") {
        setFormData({
          minScore: selectedOption.minSeverity?.toString() ?? "",
          maxScore: selectedOption.maxSeverity?.toString() ?? "",
        });
      }
    }
  };

  const onScoreChange = ({ name, value }: IInputFieldParseTarget) => {
    // Prepare new form data
    const newFormData = { ...formData, [name]: value as string };

    // If both fields are empty, reset to "Any severity"
    if (!newFormData.minScore && !newFormData.maxScore) {
      setSeverity(ANY_SEVERITY_OPTION);
      setFormData({ minScore: "", maxScore: "" });
      setFormErrors({});
      return;
    }

    // Parse values for matching severity option
    const minVal = parseFloat(newFormData.minScore || "0");
    const maxVal = parseFloat(newFormData.maxScore || "10");

    const selectedOption = SEVERITY_DROPDOWN_OPTIONS.find(
      (option) => option.minSeverity === minVal && option.maxSeverity === maxVal
    );
    setSeverity(selectedOption || CUSTOM_SEVERITY_OPTION);
    setFormData(newFormData);
    // InputField only allows numbers
    // Only errors if number outside range or multiple decimals
    setFormErrors(validate(newFormData));
  };

  const handleSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    const errors = validate(formData);
    if (Object.keys(errors).length > 0) {
      setFormErrors(errors);
      return;
    }
    const min = formData.minScore ? parseFloat(formData.minScore) : undefined;
    const max = formData.maxScore ? parseFloat(formData.maxScore) : undefined;

    onSubmit({
      vulnerable: vulnSoftwareFilterEnabled,
      exploit: hasKnownExploit || undefined,
      minCvssScore: min === 0 ? undefined : min,
      maxCvssScore: max === 10 ? undefined : max,
    });
  };

  const renderSeverityLabel = () => {
    return (
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
  };

  const renderModalContent = () => {
    return (
      <form onSubmit={handleSubmit}>
        <Slider
          value={vulnSoftwareFilterEnabled}
          onChange={() =>
            setVulnSoftwareFilterEnabled(!vulnSoftwareFilterEnabled)
          }
          inactiveText="Vulnerable software"
          activeText="Vulnerable software"
        />
        {isPremiumTier && (
          <>
            <DropdownWrapper
              name="severity-filter"
              label={renderSeverityLabel()}
              options={SEVERITY_DROPDOWN_OPTIONS}
              value={severity}
              onChange={onChangeSeverity}
              placeholder="Any severity"
              className={`${baseClass}__select-severity`}
              isDisabled={!vulnSoftwareFilterEnabled}
              helpText="CVSS scores (v3) range from 0.0 to 10.0 in 0.1 increments."
            />

            <div className={`${baseClass}__cvss-range`}>
              <InputField
                label="Min score"
                onChange={onScoreChange}
                name="minScore"
                value={formData.minScore}
                disabled={!vulnSoftwareFilterEnabled}
                type="number"
                min={0}
                max={10}
                step="0.1"
                parseTarget
                error={formErrors.minScore}
              />
              <InputField
                label="Max score"
                onChange={onScoreChange}
                name="maxScore"
                value={formData.maxScore}
                disabled={!vulnSoftwareFilterEnabled}
                type="number"
                min={0}
                max={10}
                step="0.1"
                parseTarget
                error={formErrors.maxScore}
              />
            </div>
            <Checkbox
              onChange={({ value }: { value: boolean }) =>
                setHasKnownExploit(value)
              }
              name="hasKnownExploit"
              value={hasKnownExploit}
              parseTarget
              helpText="Software has vulnerabilities that have been actively exploited in the wild."
              disabled={!vulnSoftwareFilterEnabled}
            >
              Has known exploit
            </Checkbox>
          </>
        )}
        <div className="modal-cta-wrap">
          <TooltipWrapper
            tipContent={formErrors.disableApplyButton}
            disableTooltip={!formErrors.disableApplyButton}
            showArrow
            position="top"
            tipOffset={8}
            underline={false}
          >
            <Button
              type="submit"
              disabled={
                !!formErrors.disableApplyButton ||
                !!formErrors.minScore ||
                !!formErrors.maxScore
              }
            >
              Apply
            </Button>
          </TooltipWrapper>
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </form>
    );
  };

  return (
    <Modal title="Filters" onExit={onExit} className={baseClass}>
      {renderModalContent()}
    </Modal>
  );
};

export default SoftwareFiltersModal;
