import React, { useState } from "react";

import { SingleValue } from "react-select-5";
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
  CUSTOM_SERVERITY_OPTION,
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

interface IFormErrors {
  minScore?: string | null;
  maxScore?: string | null;
}

const validateCvssScore = (cvssScore, currentErrors: IFormErrors) => {
  const errors: IFormErrors = {};

  const { minScore, maxScore } = cvssScore;

  if (minScore < 0 || minScore > 10) {
    errors.minScore = "Must be from 0-10 in 0.1 increments";
  }

  if (maxScore < 0 || maxScore > 10) {
    errors.minScore = "Must be from 0-10 in 0.1 increments";
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
  const [minScore, setMinScore] = useState(
    vulnFilters.minCvssScore?.toString()
  );
  const [maxScore, setMaxScore] = useState(
    vulnFilters.maxCvssScore?.toString()
  );
  const [hasKnownExploit, setHasKnownExploit] = useState(vulnFilters.exploit);

  const [formErrors, setFormErrors] = useState();

  const onInputBlur = () => {
    setFormErrors(validateCvssScore(formData));
  };

  const onChangeSeverity = (
    selectedSeverity: SingleValue<CustomOptionType>
  ) => {
    const selectedOption = SEVERITY_DROPDOWN_OPTIONS.find(
      (option) => option.value === selectedSeverity?.value
    );
    if (selectedOption) {
      setSeverity(selectedOption);
      // Auto populate severity range except for custom serverity
      if (selectedOption.value === "any") {
        setMinScore(undefined);
        setMaxScore(undefined);
      } else if (selectedOption.value !== "custom") {
        setMinScore(selectedOption.minSeverity?.toString());
        setMaxScore(selectedOption.maxSeverity?.toString());
      }
    }
  };

  const onChangeMinScore = (selectedScore: string) => {
    const newMin = selectedScore;
    const newMax = maxScore; // current state
    if (!newMin && !newMax) {
      setSeverity(ANY_SEVERITY_OPTION);
      setMinScore("");
      setMaxScore("");
      return;
    }
    const selectedScoreNumber = parseFloat(newMin);
    const selectedOption = SEVERITY_DROPDOWN_OPTIONS.find(
      (option) =>
        option.minSeverity === selectedScoreNumber &&
        option.maxSeverity === parseFloat(newMax || "10")
    );
    setSeverity(selectedOption || CUSTOM_SERVERITY_OPTION);
    setMinScore(newMin);
  };

  const onChangeMaxScore = (selectedScore: string) => {
    const newMax = selectedScore;
    const newMin = minScore; // current state
    if (!newMin && !newMax) {
      setSeverity(ANY_SEVERITY_OPTION);
      setMinScore("");
      setMaxScore("");
      return;
    }
    const selectedScoreNumber = parseFloat(newMax);
    const selectedOption = SEVERITY_DROPDOWN_OPTIONS.find(
      (option) =>
        option.minSeverity === parseFloat(newMin || "0") &&
        option.maxSeverity === selectedScoreNumber
    );
    setSeverity(selectedOption || CUSTOM_SERVERITY_OPTION);
    setMaxScore(newMax);
  };

  const onApplyFilters = () => {
    const min = minScore ? parseFloat(minScore) : undefined;
    const max = maxScore ? parseFloat(maxScore) : undefined;

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
      <>
        <Slider
          value={vulnSoftwareFilterEnabled}
          onChange={() =>
            setVulnSoftwareFilterEnabled(!vulnSoftwareFilterEnabled)
          }
          inactiveText="Vulnerable software"
          activeText="Vulnerable software"
        />
        {isPremiumTier && (
          <DropdownWrapper
            name="severity-filter"
            label={renderSeverityLabel()}
            options={SEVERITY_DROPDOWN_OPTIONS}
            value={severity}
            onChange={onChangeSeverity}
            placeholder="Any severity"
            className={`${baseClass}__select-severity`}
            isDisabled={!vulnSoftwareFilterEnabled}
          />
        )}
        {isPremiumTier && (
          <div className={`${baseClass}__cvss-range`}>
            <InputField
              label="Min score"
              onChange={onChangeMinScore}
              name="minScore"
              value={minScore}
              disabled={!vulnSoftwareFilterEnabled}
              type="number"
              min={0}
              max={10}
            />
            <InputField
              label="Max score"
              onChange={onChangeMaxScore}
              name="maxScore"
              value={maxScore}
              disabled={!vulnSoftwareFilterEnabled}
              type="number"
              min={0}
              max={10}
            />
          </div>
        )}
        {isPremiumTier && (
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
        )}
        <div className="modal-cta-wrap">
          <Button onClick={onApplyFilters}>Apply</Button>
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    );
  };

  return (
    <Modal title="Filters" onExit={onExit} className={baseClass}>
      {renderModalContent()}
    </Modal>
  );
};

export default SoftwareFiltersModal;
