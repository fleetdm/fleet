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
  // An error forces the section open and pins it there: setting an error also
  // sets showAdvanced, and the toggle ignores clicks while an error shows.
  // Otherwise collapsing while an error is visible and then focusing the field
  // (which clears the error) would unmount it mid-edit.
  const hasScoreError = !!(formErrors.minScore || formErrors.maxScore);
  const advancedVisible = showAdvanced || hasScoreError;

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
    if (fieldError) {
      setShowAdvanced(true);
    }
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
    } else if (severity !== ANY_SEVERITY_VALUE) {
      // The bounds now apply, so don't submit them from a collapsed section.
      setShowAdvanced(true);
    }
    setVulnSoftwareFilterEnabled(next);
  };

  const handleSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    if (vulnSoftwareFilterEnabled) {
      const errors = validateSeverityScores(formData);
      if (Object.keys(errors).length > 0) {
        setFormErrors(errors);
        setShowAdvanced(true);
        return;
      }
    }
    // A 0-10 range clears the severity filter rather than submitting bounds
    // that narrow nothing — severityFilters comes back empty for it.
    const { min, max } = vulnSoftwareFilterEnabled
      ? severityFilters(formData)
      : {};

    onSubmit({
      vulnerable: vulnSoftwareFilterEnabled,
      exploit: hasKnownExploit || undefined,
      minCvssScore: min,
      maxCvssScore: max,
      types: selectedTypes,
    });
  };

  // Implicit submission clicks the form's first submit-type button, which is
  // the Vulnerable software Slider (its <button> has no type), so Enter is
  // handled here instead: ignored in the type search, Apply in score fields.
  const onFormKeyDown = (evt: React.KeyboardEvent<HTMLFormElement>) => {
    const target = evt.target as HTMLInputElement;
    if (evt.key !== "Enter" || target.tagName !== "INPUT") return;
    evt.preventDefault();
    if (target.type === "number") evt.currentTarget.requestSubmit();
  };

  const renderModalContent = () => {
    return (
      // Only intercepts Enter bubbling from the form's own inputs.
      // eslint-disable-next-line jsx-a11y/no-noninteractive-element-interactions
      <form onSubmit={handleSubmit} onKeyDown={onFormKeyDown}>
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
              onClick={() => {
                if (!hasScoreError) {
                  setShowAdvanced(!showAdvanced);
                }
              }}
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
