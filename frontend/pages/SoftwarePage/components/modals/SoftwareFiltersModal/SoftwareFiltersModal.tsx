import React, { useRef, useState } from "react";

import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Checkbox from "components/forms/fields/Checkbox";
import Slider from "components/forms/fields/Slider";
import Modal from "components/Modal";
import SeverityFilter, {
  ANY_SEVERITY_VALUE,
  ISeverityFieldErrors,
  ISeverityFilterValue,
  severityFilters,
  severityForRange,
  SeverityScoreField,
  SeverityValue,
  validateSeverityScores,
} from "components/SeverityFilter";
import { ISoftwareType } from "interfaces/software";
import { ISoftwareFilters } from "pages/SoftwarePage/SoftwareInventory/SoftwareInventoryTable/helpers";

import SoftwareTypesPicker from "./SoftwareTypesPicker";

const baseClass = "software-filters-modal";

interface ISoftwareFiltersModalProps {
  onExit: () => void;
  onSubmit: (filters: ISoftwareFilters) => void;
  filters: ISoftwareFilters;
  isPremiumTier: boolean;
  /** Types offered in the picker. The picker is hidden when omitted. */
  availableTypes?: readonly ISoftwareType[];
}

type IFormData = {
  minScore: string;
  maxScore: string;
};

const SoftwareFiltersModal = ({
  onExit,
  onSubmit,
  filters,
  isPremiumTier,
  availableTypes,
}: ISoftwareFiltersModalProps) => {
  const [selectedTypes, setSelectedTypes] = useState(filters.types ?? []);
  const [vulnSoftwareFilterEnabled, setVulnSoftwareFilterEnabled] = useState(
    filters.vulnerable || false
  );
  const [severity, setSeverity] = useState<SeverityValue>(
    severityForRange(filters.minCvssScore, filters.maxCvssScore)
  );
  // Unified form state:
  const [formData, setFormData] = useState<IFormData>({
    minScore: filters.minCvssScore?.toString() ?? "",
    maxScore: filters.maxCvssScore?.toString() ?? "",
  });
  const [hasKnownExploit, setHasKnownExploit] = useState(filters.exploit);
  const [formErrors, setFormErrors] = useState<ISeverityFieldErrors>({});
  const dirtyFields = useRef(new Set<SeverityScoreField>());
  // Collapsed by default, but an applied severity shouldn't be hidden. Bounds
  // only apply while the vulnerable filter is on.
  const [showAdvanced, setShowAdvanced] = useState(
    !!filters.vulnerable && severity !== ANY_SEVERITY_VALUE
  );
  // Keep the section open while it holds an error, or Apply would fail with
  // nothing visible to fix.
  const advancedVisible =
    showAdvanced || !!(formErrors.minScore || formErrors.maxScore);

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
      types: selectedTypes,
    });
  };

  const renderModalContent = () => {
    return (
      <form onSubmit={handleSubmit}>
        {availableTypes && (
          <SoftwareTypesPicker
            availableTypes={availableTypes}
            selectedKeys={selectedTypes}
            onChange={setSelectedTypes}
          />
        )}
        <Slider
          value={vulnSoftwareFilterEnabled}
          onChange={onToggleVulnSoftware}
          inactiveText="Vulnerable software"
          activeText="Vulnerable software"
        />
        {isPremiumTier && (
          <>
            <div className={`${baseClass}__kev`}>
              <h3 className={`${baseClass}__section-title`}>
                CISA known exploit (KEV)
              </h3>
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
            </div>
            <RevealButton
              className={`${baseClass}__advanced-toggle`}
              isShowing={advancedVisible}
              showText="Advanced"
              hideText="Advanced"
              caretPosition="after"
              onClick={() => setShowAdvanced(!advancedVisible)}
            />
            {advancedVisible && (
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
            )}
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
