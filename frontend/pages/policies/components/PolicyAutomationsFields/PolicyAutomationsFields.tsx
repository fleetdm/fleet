import React, {
  forwardRef,
  useCallback,
  useContext,
  useEffect,
  useImperativeHandle,
  useMemo,
  useState,
} from "react";
import { SingleValue } from "react-select-5";

import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";
import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TooltipWrapper from "components/TooltipWrapper";
import { AppContext } from "context/app";
import { IConfig } from "interfaces/config";
import { ProfilePlatform } from "interfaces/mdm";
import { QueryablePlatform } from "interfaces/platform";
import { IPolicy, PolicyAutomationType } from "interfaces/policy";
import { ITeamConfig, API_NO_TEAM_ID } from "interfaces/team";
import {
  findFirstAddedPackage,
  generateSoftwareOptionHelpText,
  generateSoftwarePackageOptionHelpText,
  getTicketOrWebhookLabel,
  getTicketOrWebhookState,
} from "pages/policies/helpers";
import { IPolicyAutomationUpdate } from "pages/policies/hooks";
import {
  EndUserExperience,
  PatchOption,
} from "pages/SoftwarePage/components/forms/SoftwareDeploySelector";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";
import permissions from "utilities/permissions";

import {
  filterValidProfiles,
  rewriteProfilePlatform,
  VALID_PROFILE_PLATFORMS,
} from "./helpers";
import { useProfiles, useScripts, useSoftwareTitles } from "./hooks";

const baseClass = "policy-automations-fields";

const NONE = "none";

const WEBHOOK_OR_TICKET_ADMIN_ONLY =
  "Only admins can turn the webhook or ticket automation on or off.";

const SET_BY_PATCH =
  "To choose another automation, select End user initiated (manual) under Patch.";

/** Result returned to the parent on save. Field-level validation errors are
 *  displayed within the component itself; the parent only needs `isValid` to
 *  decide whether to stop the submit. */
export interface IPolicyAutomationsPayload {
  /** false when the automation is missing its required selection. */
  isValid: boolean;
  /** false when nothing changed from the policy's stored automation. */
  isDirty: boolean;
  policyUpdate?: IPolicyAutomationUpdate;
}

export interface IPolicyAutomationsFieldsHandle {
  /** Validates the current selection (surfacing any field errors in-place) and
   *  returns whether it's valid plus the update for the parent to persist. */
  getAutomationsPayload: () => IPolicyAutomationsPayload;
}

/** Validation errors keyed by the automation that requires a selection. */
interface IAutomationsErrors {
  software?: string;
  script?: string;
  configuration_profile?: string;
}

interface IPolicyAutomationsFieldsProps {
  policy: IPolicy;
  /** When true, only the webhook or ticket automation is offered (global /
   *  "All fleets" policies, and every policy on Fleet Free). */
  isGlobalPolicy: boolean;
  /** undefined for "All fleets", 0 for "Unassigned", positive for a fleet. */
  teamIdForApi: number | undefined;
  /** Config that owns the policy's automations (team config, or global config
   *  for inherited/global policies). */
  automationsConfig: IConfig | ITeamConfig | undefined;
  /** Global config — needed to read conditional access on the "Unassigned"
   *  view. */
  globalConfig: IConfig | undefined;
  /** Fleet display name, used in the "Not enabled for <fleet>" hints. */
  fleetName: string;
  /** Present only for patch policies on Premium. */
  patchOption?: PatchOption;
  /** Present only for patch policies on Premium — the "Force patch" end-user
   *  experience choice. Controls whether continuous automations is locked on
   *  (same rule as `patchOption === "closed"`). */
  endUserExperience?: EndUserExperience;
  /** Rendered between the automation and the continuous-automation checkbox —
   *  the edit-policy Patch radios (owned by PolicyForm). */
  patchSlot?: React.ReactNode;
  /** The platforms of the updated policy, used to filter automation fields by platform */
  selectedPlatforms: QueryablePlatform[];
  /** Fired when conditional access becomes, or stops being, the selected
   *  automation, because the backend rejects hidden + conditional access. */
  onConditionalAccessChange?: (enabled: boolean) => void;
}

const PolicyAutomationsFields = forwardRef<
  IPolicyAutomationsFieldsHandle,
  IPolicyAutomationsFieldsProps
>(
  (
    {
      policy,
      isGlobalPolicy,
      teamIdForApi,
      automationsConfig,
      globalConfig,
      fleetName,
      patchOption,
      endUserExperience,
      patchSlot,
      selectedPlatforms,
      onConditionalAccessChange,
    },
    ref
  ) => {
    const { currentUser, isGlobalAdmin } = useContext(AppContext);

    const ticketOrWebhookState = getTicketOrWebhookState(automationsConfig);

    // Only admins (global, or fleet-level admin of this policy's fleet) may
    // manage the webhook/ticket automation. For "All fleets" and "Unassigned"
    // policies there is no team-level admin, so it's global-admin only.
    const canEditWebhookOrTicket =
      !!isGlobalAdmin ||
      (teamIdForApi !== undefined &&
        teamIdForApi !== API_NO_TEAM_ID &&
        !!currentUser &&
        permissions.isTeamAdmin(currentUser, teamIdForApi));

    // Calendar events are a fleet-only feature: they're never available for
    // "All fleets" or "Unassigned" policies.
    const isUnassigned = teamIdForApi === API_NO_TEAM_ID;
    const isCalendarEnabledForTeam =
      !isGlobalPolicy && !isUnassigned
        ? (automationsConfig as ITeamConfig | undefined)?.integrations
            ?.google_calendar?.enable_calendar_events ?? false
        : false;

    const getIsConditionalAccessEnabledForTeam = () => {
      if (isGlobalPolicy) return false;
      if (isUnassigned) {
        return globalConfig?.integrations?.conditional_access_enabled ?? false;
      }
      return (
        (automationsConfig as ITeamConfig | undefined)?.integrations
          ?.conditional_access_enabled ?? false
      );
    };
    const isConditionalAccessEnabledForTeam = getIsConditionalAccessEnabledForTeam();

    const initialAutomation = policy.automation_type ?? null;
    const initialContinuous = policy.continuous_automations_enabled ?? false;
    const initialPatchWhenClosed = policy.patch_when_closed ?? false;
    const initialNotifyBeforePatching = policy.notify_before_patching ?? false;
    const initialPatchDeadline = policy.patch_deadline ?? false;

    const [automation, setAutomation] = useState<PolicyAutomationType | null>(
      initialAutomation
    );
    // On a patch policy, the Patch radios decide whether it patches: any option
    // but manual makes software its automation.
    const getEffectiveAutomation = (): PolicyAutomationType | null => {
      if (patchOption === undefined) return automation;
      if (patchOption !== "manual") return "software";
      return automation === "software" ? null : automation;
    };
    const effectiveAutomation = getEffectiveAutomation();

    const [continuousEnabled, setContinuousEnabled] = useState(
      initialPatchWhenClosed ||
        initialNotifyBeforePatching ||
        initialPatchDeadline
        ? false
        : initialContinuous
    );
    const patchWhenClosed = patchOption
      ? patchOption === "closed"
      : initialPatchWhenClosed;
    const notifyBeforePatching =
      patchOption !== undefined
        ? patchOption === "force" && endUserExperience === "notify"
        : initialNotifyBeforePatching;
    const patchDeadline =
      patchOption !== undefined
        ? patchOption === "force" && endUserExperience === "deadline"
        : initialPatchDeadline;
    // Continuous automations is locked on for Patch when app is closed, Notify
    // before patching, and Deadline — the backend forces it on for all three,
    // so an editable checkbox would lie.
    const isContinuousAutomationsRequired =
      patchWhenClosed || notifyBeforePatching || patchDeadline;
    const getContinuousAutomationsRequiredTooltip = () => {
      if (patchWhenClosed) {
        return "Continuous automation can't be disabled when Patch when app is closed is selected.";
      }
      if (notifyBeforePatching) {
        return "Continuous automation can't be disabled when Notify before patching is selected.";
      }
      if (patchDeadline) {
        return "Continuous automation can't be disabled when Deadline is selected.";
      }
      return undefined;
    };
    const getEffectiveContinuousEnabled = () => {
      if (isContinuousAutomationsRequired) return true;
      // Manual patch hides the continuous checkbox; preserve the stored value
      // on the wire so a policy that had continuous on doesn't silently flip.
      if (patchOption === "manual") return initialContinuous;
      // Same for automations that hide it, so an earlier toggle doesn't stick.
      if (
        effectiveAutomation !== "software" &&
        effectiveAutomation !== "script"
      ) {
        return initialContinuous;
      }
      return continuousEnabled;
    };
    const continuousAutomationsRequiredTooltip = getContinuousAutomationsRequiredTooltip();
    const effectiveContinuousEnabled = getEffectiveContinuousEnabled();

    const [softwareTitleId, setSoftwareTitleId] = useState<number | null>(
      policy.install_software?.software_title_id ?? null
    );
    // Pins the automation to a specific package on a multi-package title.
    // When the policy payload doesn't carry `software_package_id` (VPP
    // titles never do; single-package titles didn't need it), the
    // auto-select effect below resolves to first-added.
    const [softwarePackageId, setSoftwarePackageId] = useState<number | null>(
      policy.install_software?.software_package_id ?? null
    );
    const [scriptId, setScriptId] = useState<number | null>(
      policy.run_script?.id ?? null
    );
    const [profileUUID, setProfileUUID] = useState<string | null>(
      policy.resend_configuration_profile?.profile_uuid ?? null
    );
    const hasProfilePlatform = selectedPlatforms.some((p) =>
      VALID_PROFILE_PLATFORMS.includes(p as ProfilePlatform)
    );

    const [errors, setErrors] = useState<IAutomationsErrors>({});

    const clearError = useCallback(
      (key: keyof IAutomationsErrors) =>
        setErrors((prev) => {
          if (!prev[key]) return prev;
          const next = { ...prev };
          delete next[key];
          return next;
        }),
      [setErrors]
    );

    // An empty selection means the platform checkboxes haven't hydrated yet
    // (they mount unchecked and are filled in by an effect), so it can't be
    // read as "no profile platform" without wiping a stored selection.
    const hasPlatformSelection = selectedPlatforms.length > 0;
    useEffect(() => {
      if (hasPlatformSelection && !hasProfilePlatform) {
        setProfileUUID(null);
        setAutomation((prev) =>
          prev === "configuration_profile" ? null : prev
        );
        clearError("configuration_profile");
      }
    }, [hasPlatformSelection, hasProfilePlatform, clearError]);

    const showSoftwarePicker =
      effectiveAutomation === "software" && patchOption === undefined;

    const canFetchTeamScopedLists =
      !isGlobalPolicy && teamIdForApi !== undefined;
    const { data: softwareTitlesData } = useSoftwareTitles({
      fleetId: teamIdForApi ?? 0,
      enabled: canFetchTeamScopedLists && showSoftwarePicker,
    });
    const { data: scriptsData } = useScripts({
      fleetId: teamIdForApi ?? 0,
      enabled: canFetchTeamScopedLists && effectiveAutomation === "script",
    });
    const { data: profilesData } = useProfiles({
      fleetId: teamIdForApi ?? 0,
      enabled:
        canFetchTeamScopedLists &&
        effectiveAutomation === "configuration_profile",
    });

    const softwareOptions: CustomOptionType[] = useMemo(
      () =>
        (softwareTitlesData?.software_titles ?? []).map((t) => ({
          label: getDisplayedSoftwareName(t.name, t.display_name),
          value: String(t.id),
          helpText: generateSoftwareOptionHelpText(t),
        })),
      [softwareTitlesData]
    );

    // Packages on the currently-selected title. Non-null only for custom
    // multi-package titles — VPP / App Store titles carry no packages[].
    const selectedTitlePackages = useMemo(() => {
      if (softwareTitleId === null) return null;
      const selected = softwareTitlesData?.software_titles?.find(
        (t) => t.id === softwareTitleId
      );
      return selected?.packages ?? null;
    }, [softwareTitleId, softwareTitlesData]);

    const packageOptions: CustomOptionType[] = useMemo(
      () =>
        (selectedTitlePackages ?? []).map((pkg) => ({
          label: pkg.name,
          value: String(pkg.installer_id),
          helpText: generateSoftwarePackageOptionHelpText(pkg),
        })),
      [selectedTitlePackages]
    );

    // Auto-select the first-added package whenever the current selection
    // isn't valid for the resolved packages list — covers three cases:
    //   1. Fresh title selection: installer id was reset to null in
    //      handleSelectSoftware; pick first-added.
    //   2. Legacy policy load: hydrated with software_title_id but no
    //      software_package_id (e.g., policies created before backend
    //      surfaced the field); resolve to first-added on the title's packages.
    //   3. Stale selection: an installer id that no longer appears on the
    //      title's packages (rare — e.g., a race where the package was
    //      deleted server-side); fall back to first-added rather than saving
    //      a broken pin.
    useEffect(() => {
      if (!selectedTitlePackages || selectedTitlePackages.length === 0) return;
      const stillValid =
        softwarePackageId !== null &&
        selectedTitlePackages.some((p) => p.installer_id === softwarePackageId);
      if (stillValid) return;
      const first = findFirstAddedPackage(selectedTitlePackages);
      if (first) setSoftwarePackageId(first.installer_id);
    }, [selectedTitlePackages, softwarePackageId]);

    const scriptOptions: CustomOptionType[] = useMemo(
      () =>
        (scriptsData?.scripts ?? []).map((s) => ({
          label: s.name,
          value: String(s.id),
        })),
      [scriptsData]
    );

    const profileOptions: CustomOptionType[] = useMemo(
      () =>
        (profilesData?.profiles ?? []).filter(filterValidProfiles).map(
          (p) =>
            ({
              label: p.name,
              value: p.profile_uuid,
              helpText: rewriteProfilePlatform(p.platform),
            } as CustomOptionType)
        ),
      [profilesData]
    );

    const validate = (): IAutomationsErrors => {
      const newErrors: IAutomationsErrors = {};
      if (showSoftwarePicker) {
        if (softwareTitleId === null) {
          newErrors.software = "Please select software to install.";
        } else if (
          (selectedTitlePackages?.length ?? 0) > 0 &&
          softwarePackageId === null
        ) {
          // Only reachable when a custom title (with packages[]) is selected
          // but its packages haven't hydrated yet — the auto-select effect
          // resolves this as soon as the softwareTitlesData query returns.
          // VPP / App Store titles carry no packages[] and legitimately have
          // no installer id, so the gate above excludes them.
          newErrors.software = "Please select a package to install.";
        }
      }
      if (effectiveAutomation === "script" && scriptId === null) {
        newErrors.script = "Please select a script to run.";
      }
      if (
        effectiveAutomation === "configuration_profile" &&
        profileUUID === null
      ) {
        newErrors.configuration_profile =
          "Please select a configuration profile to resend.";
      }
      return newErrors;
    };

    const handleSelectAutomation = (opt: SingleValue<CustomOptionType>) => {
      const next =
        !opt || opt.value === NONE ? null : (opt.value as PolicyAutomationType);
      setAutomation(next);
      setErrors({});
      onConditionalAccessChange?.(next === "conditional_access");
    };
    const handleSelectSoftware = (id: number | null) => {
      setSoftwareTitleId(id);
      // A title change invalidates the pinned installer — reset so the
      // auto-select effect can pick first-added on the new title's packages.
      setSoftwarePackageId(null);
      if (id !== null) clearError("software");
    };
    const handleSelectPackage = (id: number | null) => {
      setSoftwarePackageId(id);
      if (id !== null) clearError("software");
    };
    const handleSelectScript = (id: number | null) => {
      setScriptId(id);
      if (id !== null) clearError("script");
    };
    const handleSelectProfile = (uuid: string | null) => {
      setProfileUUID(uuid);
      if (uuid !== null) clearError("configuration_profile");
    };

    useImperativeHandle(ref, () => ({
      getAutomationsPayload: () => {
        const newErrors = validate();
        setErrors(newErrors);
        if (Object.keys(newErrors).length > 0) {
          return { isValid: false, isDirty: false };
        }

        const isSoftware = showSoftwarePicker;
        const isScript = effectiveAutomation === "script";
        const isProfile = effectiveAutomation === "configuration_profile";
        const patchFlagsChanged =
          patchOption !== undefined &&
          (patchWhenClosed !== initialPatchWhenClosed ||
            notifyBeforePatching !== initialNotifyBeforePatching ||
            patchDeadline !== initialPatchDeadline);
        const isDirty =
          effectiveAutomation !== initialAutomation ||
          (isSoftware &&
            (softwareTitleId !==
              (policy.install_software?.software_title_id ?? null) ||
              softwarePackageId !==
                (policy.install_software?.software_package_id ?? null))) ||
          (isScript && scriptId !== (policy.run_script?.id ?? null)) ||
          (isProfile &&
            profileUUID !==
              (policy.resend_configuration_profile?.profile_uuid ?? null)) ||
          (!isGlobalPolicy &&
            effectiveContinuousEnabled !== initialContinuous) ||
          patchFlagsChanged;

        if (!isDirty) {
          return { isValid: true, isDirty: false };
        }

        return {
          isValid: true,
          isDirty: true,
          policyUpdate: {
            automation_type: effectiveAutomation,
            // A patch policy's app is fixed, so automation_type alone turns
            // patching on.
            ...(isSoftware && {
              software_title_id: softwareTitleId,
              software_package_id: softwarePackageId,
            }),
            ...(isScript && { script_id: scriptId }),
            ...(isProfile && { profile_uuid: profileUUID }),
            ...(!isGlobalPolicy && {
              continuous_automations_enabled: effectiveContinuousEnabled,
            }),
            ...(patchOption !== undefined &&
              patchWhenClosed !== initialPatchWhenClosed && {
                patch_when_closed: patchWhenClosed,
              }),
            ...(patchOption !== undefined &&
              notifyBeforePatching !== initialNotifyBeforePatching && {
                notify_before_patching: notifyBeforePatching,
              }),
            ...(patchOption !== undefined &&
              patchDeadline !== initialPatchDeadline && {
                patch_deadline: patchDeadline,
              }),
          },
        };
      },
    }));

    const notEnabledTooltip = `Not enabled for ${fleetName}.`;
    const isWebhookOrTicketLocked =
      !canEditWebhookOrTicket && effectiveAutomation === "webhook_or_ticket";

    const getWebhookOrTicketTooltip = () => {
      if (ticketOrWebhookState === "disabled") return notEnabledTooltip;
      if (!canEditWebhookOrTicket) return WEBHOOK_OR_TICKET_ADMIN_ONLY;
      return undefined;
    };
    const automationOptions: CustomOptionType[] = [
      { label: "None", value: NONE },
      {
        label: getTicketOrWebhookLabel(ticketOrWebhookState),
        value: "webhook_or_ticket",
        isDisabled:
          ticketOrWebhookState === "disabled" || !canEditWebhookOrTicket,
        tooltipContent: getWebhookOrTicketTooltip(),
      },
    ];
    if (!isGlobalPolicy) {
      // A patch policy's software automation is set by the Patch options.
      if (patchOption === undefined) {
        automationOptions.push({
          label: "Install software",
          value: "software",
        });
      }
      automationOptions.push(
        { label: "Run script", value: "script" },
        {
          label: "Resend configuration profile",
          value: "configuration_profile",
          isDisabled: !hasProfilePlatform,
        }
      );
      if (!isUnassigned) {
        automationOptions.push({
          label: "Calendar event",
          value: "calendar",
          isDisabled: !isCalendarEnabledForTeam,
          tooltipContent: isCalendarEnabledForTeam
            ? undefined
            : notEnabledTooltip,
        });
      }
      automationOptions.push({
        label: "Conditional access",
        value: "conditional_access",
        isDisabled: !isConditionalAccessEnabledForTeam,
        tooltipContent: isConditionalAccessEnabledForTeam
          ? undefined
          : notEnabledTooltip,
      });
    }

    const isPatching = patchOption !== undefined && patchOption !== "manual";
    // A patching policy's automation is software, which isn't an option.
    const patchingOption: CustomOptionType = {
      label: "Install software",
      value: "software",
    };
    const selectedAutomationOption = isPatching
      ? patchingOption
      : automationOptions.find(
          (o) => o.value === (effectiveAutomation ?? NONE)
        ) ?? automationOptions[0];

    const getAutomationDisabledTooltip = () => {
      if (isWebhookOrTicketLocked) return WEBHOOK_OR_TICKET_ADMIN_ONLY;
      if (isPatching) return SET_BY_PATCH;
      return undefined;
    };

    const renderItemPicker = () => {
      if (showSoftwarePicker) {
        return (
          <>
            <GitOpsModeTooltipWrapper
              isInputField
              renderChildren={(disableChildren) => (
                <DropdownWrapper
                  name="software-title"
                  label="Software"
                  isSearchable
                  isDisabled={disableChildren}
                  value={
                    softwareOptions.find(
                      (o) => o.value === String(softwareTitleId ?? "")
                    ) ?? null
                  }
                  options={softwareOptions}
                  placeholder="Select software"
                  error={errors.software}
                  onChange={(opt: SingleValue<CustomOptionType>) =>
                    handleSelectSoftware(opt ? Number(opt.value) : null)
                  }
                />
              )}
            />
            {/* Only surfaces for multi-package titles; first-added is
              auto-selected above, so this is pin-adjustment. */}
            {packageOptions.length > 1 && (
              <GitOpsModeTooltipWrapper
                isInputField
                renderChildren={(disableChildren) => (
                  <DropdownWrapper
                    name="software-package"
                    label="Package"
                    isDisabled={disableChildren}
                    value={
                      packageOptions.find(
                        (o) => o.value === String(softwarePackageId ?? "")
                      ) ?? null
                    }
                    options={packageOptions}
                    placeholder="Select package"
                    onChange={(opt: SingleValue<CustomOptionType>) =>
                      handleSelectPackage(opt ? Number(opt.value) : null)
                    }
                  />
                )}
              />
            )}
          </>
        );
      }
      if (effectiveAutomation === "script") {
        return (
          <GitOpsModeTooltipWrapper
            isInputField
            renderChildren={(disableChildren) => (
              <DropdownWrapper
                name="script"
                label="Script"
                isSearchable
                isDisabled={disableChildren}
                value={
                  scriptOptions.find(
                    (o) => o.value === String(scriptId ?? "")
                  ) ?? null
                }
                options={scriptOptions}
                placeholder="Select script"
                error={errors.script}
                onChange={(opt: SingleValue<CustomOptionType>) =>
                  handleSelectScript(opt ? Number(opt.value) : null)
                }
              />
            )}
          />
        );
      }
      if (effectiveAutomation === "configuration_profile") {
        return (
          <GitOpsModeTooltipWrapper
            isInputField
            renderChildren={(disableChildren) => (
              <DropdownWrapper
                name="profile"
                label="Configuration profile"
                isSearchable
                isDisabled={disableChildren}
                value={
                  profileOptions.find((o) => o.value === (profileUUID ?? "")) ??
                  null
                }
                options={profileOptions}
                placeholder="Select profile"
                error={errors.configuration_profile}
                onChange={(opt: SingleValue<CustomOptionType>) =>
                  handleSelectProfile(opt ? String(opt.value) : null)
                }
              />
            )}
          />
        );
      }
      return null;
    };

    const showContinuous =
      !isGlobalPolicy &&
      patchOption !== "manual" &&
      (effectiveAutomation === "software" || effectiveAutomation === "script");

    return (
      <div className={baseClass}>
        <GitOpsModeTooltipWrapper
          isInputField
          renderChildren={(disableChildren) => (
            <DropdownWrapper
              name="automation-type"
              label="Automation"
              ariaLabel="Automation"
              isDisabled={
                disableChildren || isWebhookOrTicketLocked || isPatching
              }
              disabledTooltipContent={getAutomationDisabledTooltip()}
              value={selectedAutomationOption}
              options={automationOptions}
              onChange={handleSelectAutomation}
              helpText={
                <>
                  <CustomLink
                    url="https://fleetdm.com/learn-more-about/policy-automations"
                    text="Learn more"
                    newTab
                  />{" "}
                  about automation types and their supported platforms.
                </>
              }
            />
          )}
        />

        {renderItemPicker()}

        {patchSlot}

        {showContinuous && (
          <div className={`${baseClass}__continuous`}>
            <GitOpsModeTooltipWrapper
              renderChildren={(disableChildren) => (
                <Checkbox
                  name="continuous-automations-enabled"
                  value={effectiveContinuousEnabled}
                  disabled={disableChildren || isContinuousAutomationsRequired}
                  onChange={setContinuousEnabled}
                  iconTooltipContent={continuousAutomationsRequiredTooltip}
                  helpText="If the automations do not resolve the policy, this could cause a retry loop."
                >
                  <TooltipWrapper
                    tipContent="Automations run on a host's first failure, and when a host's response changes from pass to fail. If enabled, script & software automations will also run on every subsequent failure."
                    clickable={false}
                  >
                    Continuous
                  </TooltipWrapper>{" "}
                  software &amp; script automations
                </Checkbox>
              )}
            />
          </div>
        )}
      </div>
    );
  }
);

export default PolicyAutomationsFields;
