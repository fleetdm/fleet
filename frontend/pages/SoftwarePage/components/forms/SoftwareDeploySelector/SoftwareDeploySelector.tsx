import React from "react";
import { SingleValue } from "react-select-5";

import { isMacOS, isWindows } from "interfaces/platform";

import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import Radio from "components/forms/fields/Radio";
import InfoBanner from "components/InfoBanner";

const baseClass = "software-deploy-selector";

export type PatchOption = "closed" | "force" | "manual";
export type EndUserExperience = "immediate" | "notify";

/**
 * Returns the three policy flags derived from the deploy selector state.
 * `notify_before_patching` is gated on `isMacOS(platform)` at the wire
 * boundary — Notify before patching is macOS-only today, and the UI hides
 * the dropdown for Windows, so the flag must not slip through if state
 * ever carries "notify" on a non-Mac software (legacy data, hydration
 * mismatch, or a future third platform). `endUserExperience` and
 * `platform` are defaulted so pre-migration call sites keep compiling.
 */
export const getPatchPolicyFlags = (
  patchOption: PatchOption,
  endUserExperience: EndUserExperience = "immediate",
  platform?: string
) => {
  const notifyActive =
    patchOption === "force" &&
    endUserExperience === "notify" &&
    isMacOS(platform);
  return {
    patch_when_closed: patchOption === "closed",
    notify_before_patching: notifyActive,
    continuous_automations_enabled: patchOption === "closed" || notifyActive,
  };
};

const END_USER_EXPERIENCE_LINK =
  "https://fleetdm.com/learn-more-about/patching-end-user-experience";

const END_USER_EXPERIENCE_OPTIONS: CustomOptionType[] = [
  { label: "Patch immediately", value: "immediate" },
  { label: "Notify before patching", value: "notify" },
];

/**
 * `showLink` is only true when the user could plausibly opt into Notify —
 * i.e. macOS today. On Windows and unknown platforms, Notify is unavailable
 * so the "End user experience" learn-more link would point at a feature the
 * user has no way to reach and gets suppressed.
 */
const ImmediateBanner = ({ showLink }: { showLink: boolean }) => (
  <InfoBanner icon="error-outline" iconColor="ui-fleet-black-50">
    End user is not notified. Patch is forced as soon as policy fails.
    {showLink && (
      <>
        {" "}
        <CustomLink
          url={END_USER_EXPERIENCE_LINK}
          text="End user experience"
          newTab
        />
      </>
    )}
  </InfoBanner>
);

const NotifyBanner = () => (
  <InfoBanner icon="error-outline" iconColor="ui-fleet-black-50">
    End user is notified when the policy fails. Patch is forced <b>1 hour</b>{" "}
    later. Fleet Desktop is required.{" "}
    <CustomLink
      url={END_USER_EXPERIENCE_LINK}
      text="End user experience"
      newTab
    />
  </InfoBanner>
);

const WindowsBanner = () => (
  <InfoBanner icon="error-outline" iconColor="ui-fleet-black-50">
    End user is not notified. Patch is forced as soon as policy fails.
    Notifications for Windows are coming soon.
  </InfoBanner>
);

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
  /** Platform of the target software. Drives whether the End user experience
   * dropdown renders (macOS) or the Windows coming-soon banner shows. */
  platform?: string;
  endUserExperience?: EndUserExperience;
  onSelectEndUserExperience?: (value: EndUserExperience) => void;
}

interface IPatchOptionSelectorProps {
  patchOption: PatchOption;
  onSelectPatchOption: (value: PatchOption) => void;
  disabled?: boolean;
  showPatchWhenClosedNotice?: boolean;
  platform?: string;
  endUserExperience?: EndUserExperience;
  onSelectEndUserExperience?: (value: EndUserExperience) => void;
}

export const PatchOptionSelector = ({
  patchOption,
  onSelectPatchOption,
  disabled = false,
  showPatchWhenClosedNotice = false,
  platform,
  endUserExperience = "immediate",
  onSelectEndUserExperience,
}: IPatchOptionSelectorProps) => {
  const onChangePatchOption = (value: string) =>
    onSelectPatchOption(value as PatchOption);

  const onChangeEndUserExperience = (
    newValue: SingleValue<CustomOptionType>
  ) => {
    if (newValue?.value) {
      onSelectEndUserExperience?.(newValue.value as EndUserExperience);
    }
  };

  const showEndUserExperienceDropdown =
    patchOption === "force" && isMacOS(platform) && !!onSelectEndUserExperience;

  const renderForceBanner = () => {
    // Mac wins if the platform names it — even a hypothetical `darwin,windows`
    // policy should get the dropdown-aware banner, not the "coming soon for
    // Windows" one. The widened isMacOS/isWindows both return true for such
    // strings, so precedence matters here.
    if (isMacOS(platform)) {
      return endUserExperience === "notify" ? (
        <NotifyBanner />
      ) : (
        <ImmediateBanner showLink />
      );
    }
    if (isWindows(platform)) {
      return <WindowsBanner />;
    }
    return <ImmediateBanner showLink={false} />;
  };

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
        {showEndUserExperienceDropdown && (
          <DropdownWrapper
            name="end-user-experience"
            label="End user experience"
            options={END_USER_EXPERIENCE_OPTIONS}
            value={
              END_USER_EXPERIENCE_OPTIONS.find(
                (o) => o.value === endUserExperience
              ) ?? END_USER_EXPERIENCE_OPTIONS[0]
            }
            onChange={onChangeEndUserExperience}
            isDisabled={disabled}
          />
        )}
        {patchOption === "force" && renderForceBanner()}
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
  platform,
  endUserExperience,
  onSelectEndUserExperience,
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
          platform={platform}
          endUserExperience={endUserExperience}
          onSelectEndUserExperience={onSelectEndUserExperience}
        />
      )}
    </div>
  );
};

export default SoftwareDeploySelector;
