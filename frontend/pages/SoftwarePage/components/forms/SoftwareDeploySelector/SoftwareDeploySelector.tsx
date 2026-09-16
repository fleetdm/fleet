import React from "react";

import Checkbox from "components/forms/fields/Checkbox";
import Radio from "components/forms/fields/Radio";
import InfoBanner from "components/InfoBanner";

const baseClass = "software-deploy-selector";

export type PatchOption = "closed" | "force" | "manual";

export const getPatchPolicyFlags = (patchOption: PatchOption) => ({
  patch_when_closed: patchOption === "closed",
  continuous_automations_enabled: patchOption === "closed",
});

interface ISoftwareDeploySelectorProps {
  forceInstall: boolean;
  patch: boolean;
  patchOption: PatchOption;
  onToggleForceInstall: (value: boolean) => void;
  onTogglePatch: (value: boolean) => void;
  onSelectPatchOption: (value: PatchOption) => void;
  disabled?: boolean;
  showPatchWhenClosedNotice?: boolean;
  /** Hide the "Deploy" label — e.g. in the Deploy modal, whose title is already "Deploy". */
  hideLabel?: boolean;
}

interface IPatchOptionSelectorProps {
  patchOption: PatchOption;
  onSelectPatchOption: (value: PatchOption) => void;
  disabled?: boolean;
  showPatchWhenClosedNotice?: boolean;
}

export const PatchOptionSelector = ({
  patchOption,
  onSelectPatchOption,
  disabled = false,
  showPatchWhenClosedNotice = false,
}: IPatchOptionSelectorProps) => {
  const onChangePatchOption = (value: string) =>
    onSelectPatchOption(value as PatchOption);

  return (
    <>
      <div
        className={`${baseClass}__patch-options`}
        role="radiogroup"
        aria-label="Patch options"
      >
        <Radio
          id="patch-when-closed"
          name="patch-option"
          value="closed"
          label="Patch when app is closed"
          checked={patchOption === "closed"}
          onChange={onChangePatchOption}
          disabled={disabled}
        />
        <Radio
          id="force-patch"
          name="patch-option"
          value="force"
          label="Force patch"
          checked={patchOption === "force"}
          onChange={onChangePatchOption}
          disabled={disabled}
        />
        <Radio
          id="manual-patch"
          name="patch-option"
          value="manual"
          label="End user initiated (manual)"
          checked={patchOption === "manual"}
          onChange={onChangePatchOption}
          disabled={disabled}
        />
        {patchOption === "force" && (
          <InfoBanner icon="error-outline" iconColor="ui-fleet-black-50">
            End user is not notified. Patch is forced as soon as policy fails.
            Notifications are coming soon.
          </InfoBanner>
        )}
      </div>
      {patchOption === "closed" && showPatchWhenClosedNotice && (
        <p className={`${baseClass}__patch-when-closed-notice`}>
          <b>Patch when app is closed</b> overrides the pre-install query
          (advanced option) to check if the app is closed.
        </p>
      )}
    </>
  );
};

const SoftwareDeploySelector = ({
  forceInstall,
  patch,
  patchOption,
  onToggleForceInstall,
  onTogglePatch,
  onSelectPatchOption,
  disabled = false,
  showPatchWhenClosedNotice = false,
  hideLabel = false,
}: ISoftwareDeploySelectorProps) => {
  return (
    <div className={`form-field ${baseClass}`}>
      {!hideLabel && <div className="form-field__label">Deploy</div>}
      <div className={`${baseClass}__checkboxes`}>
        <Checkbox
          name="force-install"
          value={forceInstall}
          onChange={onToggleForceInstall}
          disabled={disabled}
        >
          Force install
        </Checkbox>
        <Checkbox
          name="patch"
          value={patch}
          onChange={onTogglePatch}
          disabled={disabled}
        >
          Patch
        </Checkbox>
      </div>
      {patch && (
        <PatchOptionSelector
          patchOption={patchOption}
          onSelectPatchOption={onSelectPatchOption}
          disabled={disabled}
          showPatchWhenClosedNotice={showPatchWhenClosedNotice}
        />
      )}
    </div>
  );
};

export default SoftwareDeploySelector;
