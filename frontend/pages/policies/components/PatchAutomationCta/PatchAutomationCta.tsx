import React from "react";

import { IPolicy } from "interfaces/policy";

import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

const baseClass = "patch-automation-cta";

interface IPatchAutomationCtaProps {
  storedPolicy: IPolicy;
  /** Some users only have access to read-only view */
  canEditPolicy: boolean;
  onAddAutomation: () => void;
  isAddingAutomation?: boolean;
}

/** CTA card shown above the automations section for patch policies that have
 *  a patch software target but haven't been wired to install it yet. Returns
 *  null when the conditions aren't met, so callers can render unconditionally. */
const PatchAutomationCta = ({
  storedPolicy,
  canEditPolicy,
  onAddAutomation,
  isAddingAutomation,
}: IPatchAutomationCtaProps): JSX.Element | null => {
  const isPatchPolicy = storedPolicy.type === "patch";
  const hasSoftwareAutomation = !!storedPolicy.install_software;

  if (
    !isPatchPolicy ||
    !storedPolicy.patch_software ||
    hasSoftwareAutomation ||
    !canEditPolicy
  ) {
    return null;
  }

  const patchSoftwareName = getDisplayedSoftwareName(
    storedPolicy.patch_software.name,
    storedPolicy.patch_software.display_name
  );

  return (
    <div className={baseClass}>
      <span className={`${baseClass}__label`}>
        Automatically patch {patchSoftwareName}
      </span>
      <GitOpsModeTooltipWrapper
        position="top"
        renderChildren={(disableChildren) => (
          <Button
            onClick={onAddAutomation}
            variant="text-icon"
            disabled={disableChildren || isAddingAutomation}
          >
            {isAddingAutomation ? (
              "Adding..."
            ) : (
              <>
                <Icon name="plus" /> Add automation
              </>
            )}
          </Button>
        )}
      />
    </div>
  );
};

export default PatchAutomationCta;
