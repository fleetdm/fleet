import React, { useRef, useState } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Slider from "components/forms/fields/Slider";
import Checkbox from "components/forms/fields/Checkbox";
import SeverityFilter, {
  ISeverityFieldErrors,
  ISeverityFilterValue,
  severityFilters,
  severityForRange,
  SeverityScoreField,
  SeverityValue,
  validateSeverityScores,
} from "components/SeverityFilter";
import { ISoftwareVulnFiltersParams } from "pages/SoftwarePage/SoftwareInventory/SoftwareInventoryTable/helpers";

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

const SoftwareFiltersModal = ({
  onExit,
  onSubmit,
  vulnFilters,
  isPremiumTier,
}: ISoftwareFiltersModalProps) => {
  const [vulnSoftwareFilterEnabled, setVulnSoftwareFilterEnabled] = useState(
    vulnFilters.vulnerable || false
  );
  const [severity, setSeverity] = useState<SeverityValue>(
    severityForRange(vulnFilters.minCvssScore, vulnFilters.maxCvssScore)
  );
  // Unified form state:
  const [formData, setFormData] = useState<IFormData>({
    minScore: vulnFilters.minCvssScore?.toString() ?? "",
    maxScore: vulnFilters.maxCvssScore?.toString() ?? "",
  });
  const [hasKnownExploit, setHasKnownExploit] = useState(vulnFilters.exploit);
  const [formErrors, setFormErrors] = useState<ISeverityFieldErrors>({});
  const dirtyFields = useRef(new Set<SeverityScoreField>());

  const onChangeSeverity = ({
    severity: nextSeverity,
    minScore,
    maxScore,
  }: ISeverityFilterValue) => {
    if (nextSeverity === severity) {
      if (minScore !== formData.minScore) dirtyFields.current.add("minScore");
      if (maxScore !== formData.maxScore) dirtyFields.current.add("maxScore");
    } else {
      setFormErrors({});
    }
    setSeverity(nextSeverity);
    setFormData({ minScore, maxScore });
  };

  // Blur validates that one field and nothing else.
  const onScoreBlur = (field: SeverityScoreField) => {
    if (!dirtyFields.current.has(field)) {
      return;
    }
    const { [field]: fieldError } = validateSeverityScores(formData);
    setFormErrors((prev) => ({ ...prev, [field]: fieldError }));
  };

  // Focus clears immediately so the label returns while the user edits.
  const onScoreFocus = (field: SeverityScoreField) => {
    setFormErrors((prev) =>
      prev[field] ? { ...prev, [field]: undefined } : prev
    );
  };

  const onToggleVulnSoftware = () => {
    const next = !vulnSoftwareFilterEnabled;
    if (!next) {
      setFormErrors({});
    }
    setVulnSoftwareFilterEnabled(next);
  };

  const handleSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    if (vulnSoftwareFilterEnabled) {
      const errors = validateSeverityScores(formData);
      if (Object.keys(errors).length > 0) {
        setFormErrors(errors);
        return;
      }
    }
    // A 0-10 range clears the severity filter rather than submitting bounds
    // that narrow nothing — severityFilters comes back empty for it.
    const { min, max } = severityFilters(formData);

    onSubmit({
      vulnerable: vulnSoftwareFilterEnabled,
      exploit: hasKnownExploit || undefined,
      minCvssScore: min,
      maxCvssScore: max,
    });
  };

  const renderModalContent = () => {
    return (
      <form onSubmit={handleSubmit}>
        <Slider
          value={vulnSoftwareFilterEnabled}
          onChange={onToggleVulnSoftware}
          inactiveText="Vulnerable software"
          activeText="Vulnerable software"
        />
        {isPremiumTier && (
          <>
            <SeverityFilter
              severity={severity}
              minScore={formData.minScore}
              maxScore={formData.maxScore}
              onChange={onChangeSeverity}
              disabled={!vulnSoftwareFilterEnabled}
              errors={formErrors}
              onScoreBlur={onScoreBlur}
              onScoreFocus={onScoreFocus}
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
          </>
        )}
        <div className="modal-cta-wrap">
          <Button type="submit">Apply</Button>
          <Button variant="secondary" onClick={onExit}>
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
