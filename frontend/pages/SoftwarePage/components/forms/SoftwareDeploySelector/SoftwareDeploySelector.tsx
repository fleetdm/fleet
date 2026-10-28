import React from "react";
import { browserHistory } from "react-router";
import { SingleValue } from "react-select-5";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";
import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import Radio from "components/forms/fields/Radio";
import InfoBanner from "components/InfoBanner";
import usePatchDeadlineSettings from "hooks/usePatchDeadlineSettings";
import { formatWeekdayList } from "interfaces/patch_deadline_settings";
import { isMacOS, isWindows } from "interfaces/platform";
import PATHS from "router/paths";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { internationalTimeOnlyFormat } from "utilities/helpers";
import { getPathWithQueryParams } from "utilities/url";

const baseClass = "software-deploy-selector";

export type PatchOption = "closed" | "force" | "manual";
export type EndUserExperience = "deadline" | "immediate" | "notify";

/** Picks the end-user experience the dropdown should show when the user
 * changes the patch option. Deadline is preselected whenever Force patch is
 * chosen on a macOS target (Figma dev note on #55500). Non-macOS platforms
 * never render the dropdown, so this is a no-op for them. Any other
 * transition keeps the user's current choice. */
export const getNextEndUserExperience = (
  nextPatchOption: PatchOption,
  prevEndUserExperience: EndUserExperience,
  platform: string | undefined
): EndUserExperience => {
  if (nextPatchOption === "force" && isMacOS(platform)) {
    return "deadline";
  }
  return prevEndUserExperience;
};

/** Returns the four policy flags derived from the deploy selector state. */
export const getPatchPolicyFlags = (
  patchOption: PatchOption,
  endUserExperience: EndUserExperience = "deadline"
) => {
  const notifyActive =
    patchOption === "force" && endUserExperience === "notify";
  const deadlineActive =
    patchOption === "force" && endUserExperience === "deadline";
  return {
    patch_when_closed: patchOption === "closed",
    notify_before_patching: notifyActive,
    patch_deadline: deadlineActive,
    continuous_automations_enabled:
      patchOption === "closed" || notifyActive || deadlineActive,
  };
};

const END_USER_EXPERIENCE_LINK = `${LEARN_MORE_ABOUT_BASE_LINK}/patching-end-user-experience`;

const END_USER_EXPERIENCE_OPTIONS: CustomOptionType[] = [
  { label: "Deadline", value: "deadline" },
  { label: "Notify before patching", value: "notify" },
  { label: "Patch immediately", value: "immediate" },
];

/**
 * `showLink` is only true when the user could plausibly opt into Notify or
 * Deadline — i.e. macOS today. On Windows and unknown platforms, those
 * variants are unavailable so the "End user experience" learn-more link
 * would point at a feature the user has no way to reach and gets suppressed.
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

/** Banner for Force patch -> Deadline. Reads the fleet's days + time from the
 * patch_deadline_settings API via the shared hook so the Library settings
 * page and the banner dedupe through the same React Query cache key. The
 * "Edit deadline" action uses browserHistory.push rather than an anchor tag
 * so navigation stays in-SPA — a hard navigation would spin up a fresh
 * QueryClient on the settings page, and the settings page's
 * invalidateQueries(["patchDeadlineSettings", fleetId]) wouldn't reach this
 * banner on Back (the banner would show stale days/time). */
const DeadlineBanner = ({ fleetId }: { fleetId: number }) => {
  const { data } = usePatchDeadlineSettings(fleetId);
  const daysLabel = formatWeekdayList(data?.days ?? []);
  const timeLabel = data?.time ? internationalTimeOnlyFormat(data.time) : "";
  const settingsPath = getPathWithQueryParams(
    PATHS.SOFTWARE_LIBRARY_SETTINGS_PATCH_DEADLINE,
    { fleet_id: fleetId }
  );
  return (
    <InfoBanner icon="error-outline" iconColor="ui-fleet-black-50">
      Patch is done silently when app is closed or forced on{" "}
      <b>
        {daysLabel || "---"}
        {timeLabel ? ` at ${timeLabel}` : ""}
      </b>{" "}
      (host&apos;s local time).{" "}
      <Button variant="link" onClick={() => browserHistory.push(settingsPath)}>
        Edit deadline
      </Button>
    </InfoBanner>
  );
};

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
  /** Fleet id used by the Deadline banner to fetch the fleet's days + time
   * and build the "Edit deadline" link. Required to show Deadline details;
   * when absent the dropdown still works but the banner renders placeholders. */
  fleetId?: number;
}

interface IPatchOptionSelectorProps {
  patchOption: PatchOption;
  onSelectPatchOption: (value: PatchOption) => void;
  disabled?: boolean;
  showPatchWhenClosedNotice?: boolean;
  platform?: string;
  endUserExperience?: EndUserExperience;
  onSelectEndUserExperience?: (value: EndUserExperience) => void;
  fleetId?: number;
}

export const PatchOptionSelector = ({
  patchOption,
  onSelectPatchOption,
  disabled = false,
  showPatchWhenClosedNotice = false,
  platform,
  endUserExperience = "deadline",
  onSelectEndUserExperience,
  fleetId,
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
      if (endUserExperience === "deadline" && fleetId !== undefined) {
        return <DeadlineBanner fleetId={fleetId} />;
      }
      if (endUserExperience === "notify") {
        return <NotifyBanner />;
      }
      return <ImmediateBanner showLink />;
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
            ariaLabel="End user experience"
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
  fleetId,
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
          fleetId={fleetId}
        />
      )}
    </div>
  );
};

export default SoftwareDeploySelector;
