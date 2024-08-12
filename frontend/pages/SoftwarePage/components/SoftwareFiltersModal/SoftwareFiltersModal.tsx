import React, { useState } from "react";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Slider from "components/forms/fields/Slider";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "software-filters-modal";

interface ISoftwareFiltersModalProps {
  onExit: () => void;
  onSubmit: (filters: any) => void;
}

export const SEVERITY_DROPDOWN_OPTIONS = [
  {
    disabled: false,
    label: "Any severity",
    value: "any",
    helpText: "CVSS score 0-10",
  },
  {
    disabled: false,
    label: "Low severity",
    value: "low",
    helpText: "CVSS score 0.1-3.9",
  },
  {
    disabled: false,
    label: "Medium severity",
    value: "medium",
    helpText: "CVSS score 4.0-6.9",
  },
  {
    disabled: false,
    label: "High severity",
    value: "high",
    helpText: "CVSS score 7.0-8.9",
  },
  {
    disabled: false,
    label: "Critical severity",
    value: "critical",
    helpText: "CVSS score 9.0-10",
  },
];

const SoftwareFiltersModal = ({
  onExit,
  onSubmit,
}: ISoftwareFiltersModalProps) => {
  const [vulnSoftwareFilterEnabled, setVulnSoftwareFilterEnabled] = useState(
    false
  );
  const [severity, setSeverity] = useState("any");
  const [hasKnownExploit, setHasKnownExploit] = useState(false);

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
          onChange={(value: string) => {
            setSeverity(value);
          }}
          placeholder="Choose Table..."
          className={`${baseClass}__table-select`}
        />
        <Checkbox
          onChange={(value: boolean) => setHasKnownExploit(value)}
          name="hasKnownExploit"
          value={hasKnownExploit}
          parseTarget
          helpText="Software has vulnerabilities that have been actively exploited in the wild."
        >
          Has known exploit
        </Checkbox>
        <div className="modal-cta-wrap">
          <Button variant="brand" onClick={onSubmit}>
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
