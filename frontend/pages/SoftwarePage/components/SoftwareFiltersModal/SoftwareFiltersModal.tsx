import React, { useState } from "react";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Slider from "components/forms/fields/Slider";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import { ISoftwareVulnFilters } from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/helpers";

const baseClass = "software-filters-modal";

interface ISoftwareFiltersModalProps {
  onExit: () => void;
  onSubmit: (vulnFilters: ISoftwareVulnFilters) => void;
  vulnFiltersQueryParams: {
    vulnerable?: boolean;
    exploit?: boolean;
    minCvssScore?: number;
    maxCvssScore?: number;
  };
}

export const SEVERITY_DROPDOWN_OPTIONS = [
  {
    disabled: false,
    label: "Any severity",
    value: "any",
    helpText: "CVSS score 0-10",
    minSeverity: undefined,
    maxSeverity: undefined,
  },
  {
    disabled: false,
    label: "Low severity",
    value: "low",
    helpText: "CVSS score 0.1-3.9",
    minSeverity: 0.1,
    maxSeverity: 3.9,
  },
  {
    disabled: false,
    label: "Medium severity",
    value: "medium",
    helpText: "CVSS score 4.0-6.9",
    minSeverity: 4.0,
    maxSeverity: 6.9,
  },
  {
    disabled: false,
    label: "High severity",
    value: "high",
    helpText: "CVSS score 7.0-8.9",
    minSeverity: 7.0,
    maxSeverity: 8.9,
  },
  {
    disabled: false,
    label: "Critical severity",
    value: "critical",
    helpText: "CVSS score 9.0-10",
    minSeverity: 9.0,
    maxSeverity: 10,
  },
];

const SoftwareFiltersModal = ({
  onExit,
  onSubmit,
  vulnFiltersQueryParams,
}: ISoftwareFiltersModalProps) => {
  const [vulnSoftwareFilterEnabled, setVulnSoftwareFilterEnabled] = useState(
    vulnFiltersQueryParams.vulnerable || false
  );
  const [severity, setSeverity] = useState("any");
  const [hasKnownExploit, setHasKnownExploit] = useState(
    vulnFiltersQueryParams.exploit
  );
  const [minCvssScore, setMinCvssScore] = useState<number | undefined>(
    vulnFiltersQueryParams.minCvssScore
  );
  const [maxCvssScore, setMaxCvssScore] = useState<number | undefined>(
    vulnFiltersQueryParams.maxCvssScore
  );

  const onChangeSeverity = (value: string) => {
    setSeverity(value);
    const selectedOption = SEVERITY_DROPDOWN_OPTIONS.find(
      (option) => option.value === value
    );
    if (selectedOption) {
      setMinCvssScore(selectedOption.minSeverity);
      setMaxCvssScore(selectedOption.maxSeverity);
    }
  };

  const onApplyFilters = () => {
    onSubmit({
      vulnerable: vulnSoftwareFilterEnabled,
      exploit: hasKnownExploit,
      min_cvss_score: minCvssScore,
      max_cvss_score: maxCvssScore,
    });
  };

  const renderSeverityLabel = () => {
    return (
      <TooltipWrapper
        tipContent="The worst case impact across different environments (CVSS version 3.x base score)."
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
        <Dropdown
          label={renderSeverityLabel()}
          options={SEVERITY_DROPDOWN_OPTIONS}
          value={severity}
          onChange={onChangeSeverity}
          placeholder="Any severity"
          className={`${baseClass}__select-severity`}
          disabled={!vulnSoftwareFilterEnabled}
        />
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
        <div className="modal-cta-wrap">
          <Button variant="brand" onClick={onApplyFilters}>
            Apply
          </Button>
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
