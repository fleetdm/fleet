import React, { useState } from "react";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Slider from "components/forms/fields/Slider";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import {
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
  const [hasKnownExploit, setHasKnownExploit] = useState(vulnFilters.exploit);

  const onChangeSeverity = (value: string) => {
    const selectedOption = SEVERITY_DROPDOWN_OPTIONS.find(
      (option) => option.value === value
    );
    if (selectedOption) {
      setSeverity(selectedOption);
    }
  };

  const onApplyFilters = () => {
    onSubmit({
      vulnerable: vulnSoftwareFilterEnabled,
      exploit: hasKnownExploit || undefined,
      minCvssScore: severity?.minSeverity || undefined,
      maxCvssScore: severity?.maxSeverity || undefined,
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
        {isPremiumTier && (
          <Dropdown
            label={renderSeverityLabel()}
            options={SEVERITY_DROPDOWN_OPTIONS}
            value={severity}
            onChange={onChangeSeverity}
            placeholder="Any severity"
            className={`${baseClass}__select-severity`}
            disabled={!vulnSoftwareFilterEnabled}
          />
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
