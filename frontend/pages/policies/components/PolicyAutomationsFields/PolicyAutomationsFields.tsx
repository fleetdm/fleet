/* eslint-disable @typescript-eslint/no-use-before-define */

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

import { AppContext } from "context/app";
import { IConfig } from "interfaces/config";
import { IPolicy } from "interfaces/policy";
import { ITeamConfig, API_NO_TEAM_ID } from "interfaces/team";
import { QueryablePlatform } from "interfaces/platform";
import { ProfilePlatform } from "interfaces/mdm";

import permissions from "utilities/permissions";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import useGitOpsMode from "hooks/useGitOpsMode";

import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TooltipWrapper from "components/TooltipWrapper";

import {
  findFirstAddedPackage,
  generateSoftwareOptionHelpText,
  generateSoftwarePackageOptionHelpText,
  getTicketOrWebhookInfo,
  getTicketOrWebhookLabel,
} from "pages/policies/helpers";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";
import { PatchOption } from "pages/SoftwarePage/components/forms/SoftwareDeploySelector";

import { IPolicyAutomationUpdate } from "pages/policies/hooks";

import { IAutomationCheckboxRow } from "./types";
import { useProfiles, useScripts, useSoftwareTitles } from "./hooks";
import {
  filterValidProfiles,
  rewriteProfilePlatform,
  VALID_PROFILE_PLATFORMS,
} from "./helpers";

const baseClass = "policy-automations-fields";

/** Result returned to the parent on save, describing what (if anything)
 *  changed. Field-level validation errors are displayed within the component
 *  itself; the parent only needs `isValid` to decide whether to stop the
 *  submit, then persists the update parts. */
export interface IPolicyAutomationsPayload {
  /** false when a checked automation is missing its required selection. The
   *  component surfaces the specific error(s) above the table; the parent just
   *  stops the submit. */
  isValid: boolean;
  /** false when nothing changed from the policy's stored automations. */
  isDirty: boolean;
  policyUpdate?: IPolicyAutomationUpdate;
  webhookOrTicketUpdate?: { enabled: boolean };
}

export interface IPolicyAutomationsFieldsHandle {
  /** Validates the current selection (surfacing any field errors in-place) and
   *  returns whether it's valid plus the changed automation parts for the
   *  parent to persist. */
  getAutomationsPayload: () => IPolicyAutomationsPayload;
}

/** Validation errors keyed by the automation that requires a selection. */
interface IAutomationsErrors {
  install_software?: string;
  run_script?: string;
  resend_configuration_profile?: string;
}

interface IPolicyAutomationsFieldsProps {
  policy: IPolicy;
  /** When true, only the webhook/ticket row is shown and the continuous-retry
   *  option is hidden (global / "All fleets" policies). */
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
  /** Rendered between the automation types and the continuous-automation
   *  checkbox — the edit-policy Patch radios (owned by PolicyForm). */
  patchSlot?: React.ReactNode;
  /** The platforms of the updated policy, used to filter automation fields by platform */
  selectedPlatforms: QueryablePlatform[];
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
      patchSlot,
      selectedPlatforms,
    },
    ref
  ) => {
    const { gitOpsModeEnabled } = useGitOpsMode();
    const { currentUser, isGlobalAdmin } = useContext(AppContext);

    const {
      state: ticketOrWebhookState,
      policyIds: webhookOrTicketPolicyIds,
    } = getTicketOrWebhookInfo(automationsConfig);
    const isTicketWebhookEnabled = ticketOrWebhookState !== "disabled";

    // Only admins (global, or fleet-level admin of this policy's fleet) may
    // manage the webhook/ticket automation. For "All fleets" and "Unassigned"
    // policies there is no team-level admin, so it's global-admin only.
    const canEditWebhookOrTicket =
      !!isGlobalAdmin ||
      (teamIdForApi !== undefined &&
        teamIdForApi !== API_NO_TEAM_ID &&
        !!currentUser &&
        permissions.isTeamAdmin(currentUser, teamIdForApi));

    // Calendar events are a team/fleet-only feature: they're never available for
    // global ("All fleets") or "No team"/"Unassigned" policies.
    const isCalendarEnabledForTeam =
      !isGlobalPolicy && teamIdForApi !== API_NO_TEAM_ID
        ? (automationsConfig as ITeamConfig | undefined)?.integrations
            ?.google_calendar?.enable_calendar_events ?? false
        : false;

    const getIsConditionalAccessEnabledForTeam = () => {
      if (isGlobalPolicy) return false;
      if (teamIdForApi === API_NO_TEAM_ID) {
        return globalConfig?.integrations?.conditional_access_enabled ?? false;
      }
      return (
        (automationsConfig as ITeamConfig | undefined)?.integrations
          ?.conditional_access_enabled ?? false
      );
    };
    const isConditionalAccessEnabledForTeam = getIsConditionalAccessEnabledForTeam();

    const initialWebhookOrTicket = webhookOrTicketPolicyIds.includes(policy.id);
    const initialInstallSoftware = !!policy.install_software;
    const initialRunScript = !!policy.run_script;
    const initialResendConfigProfile = !!policy.resend_configuration_profile;
    const initialCalendar = policy.calendar_events_enabled;
    const initialConditionalAccess = policy.conditional_access_enabled;
    const initialContinuous = policy.continuous_automations_enabled ?? false;
    const initialPatchWhenClosed = policy.patch_when_closed ?? false;

    const [webhookOrTicketEnabled, setWebhookOrTicketEnabled] = useState(
      initialWebhookOrTicket
    );
    const [installSoftware, setInstallSoftware] = useState(
      initialInstallSoftware
    );
    const [runScript, setRunScript] = useState(initialRunScript);
    const [resendConfigProfile, setResendConfigProfile] = useState(
      initialResendConfigProfile
    );
    const [calendarEvent, setCalendarEvent] = useState(initialCalendar);
    const [conditionalAccess, setConditionalAccess] = useState(
      initialConditionalAccess
    );
    const [continuousEnabled, setContinuousEnabled] = useState(
      initialPatchWhenClosed ? false : initialContinuous
    );
    const patchWhenClosed = patchOption
      ? patchOption === "closed"
      : initialPatchWhenClosed;
    let effectiveContinuousEnabled = continuousEnabled;
    if (patchWhenClosed) {
      effectiveContinuousEnabled = true;
    } else if (patchOption === "manual") {
      effectiveContinuousEnabled = false;
    }

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
    const patchSoftwareTitleId =
      policy.patch_software?.software_title_id ?? null;
    const effectiveInstallSoftware =
      patchOption === undefined ? installSoftware : patchOption !== "manual";
    let effectiveSoftwareTitleId = softwareTitleId;
    let effectiveSoftwarePackageId = softwarePackageId;
    if (patchOption !== undefined) {
      effectiveSoftwareTitleId = effectiveInstallSoftware
        ? patchSoftwareTitleId
        : null;
      effectiveSoftwarePackageId =
        effectiveInstallSoftware &&
        policy.install_software?.software_title_id === patchSoftwareTitleId
          ? softwarePackageId
          : null;
    }
    const [scriptId, setScriptId] = useState<number | null>(
      policy.run_script?.id ?? null
    );
    const [profileUUID, setProfileUUID] = useState<string | null>(
      policy.resend_configuration_profile?.profile_uuid ?? null
    );
    const hasProfilePlatform = selectedPlatforms.some((p) =>
      VALID_PROFILE_PLATFORMS.includes(p as ProfilePlatform)
    );
    const resendProfileDisabled = !hasProfilePlatform;

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
        setResendConfigProfile(false);
        clearError("resend_configuration_profile");
      }
    }, [hasPlatformSelection, hasProfilePlatform, clearError]);

    const validate = (): IAutomationsErrors => {
      const newErrors: IAutomationsErrors = {};
      if (effectiveInstallSoftware && effectiveSoftwareTitleId === null) {
        newErrors.install_software = "Please select software to install.";
      } else if (
        patchOption === undefined &&
        effectiveInstallSoftware &&
        effectiveSoftwareTitleId !== null &&
        (selectedTitlePackages?.length ?? 0) > 0 &&
        effectiveSoftwarePackageId === null
      ) {
        // Only reachable when a custom title (with packages[]) is selected
        // but its packages haven't hydrated yet — the auto-select effect
        // resolves this as soon as the softwareTitlesData query returns.
        // VPP / App Store titles carry no packages[] and legitimately have
        // no installer id, so the gate above excludes them.
        newErrors.install_software = "Please select a package to install.";
      }
      if (runScript && scriptId === null) {
        newErrors.run_script = "Please select a script to run.";
      }
      if (resendConfigProfile && profileUUID === null) {
        newErrors.resend_configuration_profile =
          "Please select a configuration profile to resend.";
      }
      return newErrors;
    };

    const handleToggleInstallSoftware = (next: boolean) => {
      setInstallSoftware(next);
      if (!next) clearError("install_software");
    };
    const handleToggleRunScript = (next: boolean) => {
      setRunScript(next);
      if (!next) clearError("run_script");
    };
    const handleToggleResendConfigProfile = (next: boolean) => {
      setResendConfigProfile(next);
      if (!next) clearError("resend_configuration_profile");
    };
    const handleSelectSoftware = (id: number | null) => {
      setSoftwareTitleId(id);
      // A title change invalidates the pinned installer — reset so the
      // auto-select effect can pick first-added on the new title's packages.
      setSoftwarePackageId(null);
      if (id !== null) clearError("install_software");
    };
    const handleSelectPackage = (id: number | null) => {
      setSoftwarePackageId(id);
      if (id !== null) clearError("install_software");
    };
    const handleSelectScript = (id: number | null) => {
      setScriptId(id);
      if (id !== null) clearError("run_script");
    };
    const handleSelectProfile = (uuid: string | null) => {
      setProfileUUID(uuid);
      if (uuid !== null) clearError("resend_configuration_profile");
    };
    const handleToggleContinuous = (next: boolean) => {
      setContinuousEnabled(next);
    };

    const canFetchTeamScopedLists =
      !isGlobalPolicy && teamIdForApi !== undefined;
    const { data: softwareTitlesData } = useSoftwareTitles({
      fleetId: teamIdForApi ?? 0,
      enabled:
        canFetchTeamScopedLists &&
        effectiveInstallSoftware &&
        patchOption === undefined,
    });
    const { data: scriptsData } = useScripts({
      fleetId: teamIdForApi ?? 0,
      enabled: canFetchTeamScopedLists && runScript,
    });
    const { data: profilesData } = useProfiles({
      fleetId: teamIdForApi ?? 0,
      enabled: canFetchTeamScopedLists && resendConfigProfile,
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

    useImperativeHandle(ref, () => ({
      getAutomationsPayload: () => {
        const newErrors = validate();
        setErrors(newErrors);
        if (Object.keys(newErrors).length > 0) {
          return { isValid: false, isDirty: false };
        }

        const perPolicyDirty =
          !isGlobalPolicy &&
          (effectiveInstallSoftware !== initialInstallSoftware ||
            effectiveSoftwareTitleId !==
              (policy.install_software?.software_title_id ?? null) ||
            effectiveSoftwarePackageId !==
              (policy.install_software?.software_package_id ?? null) ||
            runScript !== initialRunScript ||
            scriptId !== (policy.run_script?.id ?? null) ||
            resendConfigProfile !== initialResendConfigProfile ||
            profileUUID !==
              (policy.resend_configuration_profile?.profile_uuid ?? null) ||
            calendarEvent !== initialCalendar ||
            conditionalAccess !== initialConditionalAccess ||
            effectiveContinuousEnabled !== initialContinuous ||
            (patchOption !== undefined &&
              patchWhenClosed !== initialPatchWhenClosed));
        const webhookDirty = webhookOrTicketEnabled !== initialWebhookOrTicket;

        return {
          isValid: true,
          isDirty: perPolicyDirty || webhookDirty,
          policyUpdate: perPolicyDirty
            ? {
                software_title_id: effectiveInstallSoftware
                  ? effectiveSoftwareTitleId
                  : null,
                // Send the pinned package id when install-software is on.
                // Null clears the automation or lets the backend select the
                // Fleet-maintained app's package when a Patch radio owns it.
                software_package_id: effectiveInstallSoftware
                  ? effectiveSoftwarePackageId
                  : null,
                script_id: runScript ? scriptId : null,
                profile_uuid: resendConfigProfile ? profileUUID : null,
                // When the team has the feature disabled, the row is locked
                // and the user can't toggle it — so we omit the field instead
                // of carrying the stale state through to the PATCH. That
                // preserves the policy's stored intent for if/when the team
                // admin re-enables the feature.
                ...(isCalendarEnabledForTeam && {
                  calendar_events_enabled: calendarEvent,
                }),
                ...(isConditionalAccessEnabledForTeam && {
                  conditional_access_enabled: conditionalAccess,
                }),
                continuous_automations_enabled: effectiveContinuousEnabled,
                ...(patchOption !== undefined &&
                  patchWhenClosed !== initialPatchWhenClosed && {
                    patch_when_closed: patchWhenClosed,
                  }),
              }
            : undefined,
          webhookOrTicketUpdate: webhookDirty
            ? { enabled: webhookOrTicketEnabled }
            : undefined,
        };
      },
    }));

    const rows: IAutomationCheckboxRow[] = [
      {
        key: "ticket_webhook",
        label: getTicketOrWebhookLabel(ticketOrWebhookState),
        checked: webhookOrTicketEnabled && isTicketWebhookEnabled,
        onToggle: setWebhookOrTicketEnabled,
        isDisabled: !isTicketWebhookEnabled,
        // Webhook/ticket config requires admin (it writes app/team config, not
        // the policy itself). Lock for non-admins with no explanation.
        isLocked: !canEditWebhookOrTicket,
      },
    ];
    if (!isGlobalPolicy) {
      rows.push(
        {
          key: "install_software",
          label: "Install software",
          tooltip: (
            <AutomationRowTooltip
              text="The selected software will be installed when hosts fail the policy. Host counts will reset when new software is selected."
              learnMoreUrl="https://fleetdm.com/learn-more-about/policy-automation-install-software"
            />
          ),
          checked: effectiveInstallSoftware,
          onToggle: handleToggleInstallSoftware,
          isDisabled: false,
          isLocked: patchOption !== undefined,
          picker:
            effectiveInstallSoftware && patchOption === undefined ? (
              <div className={`${baseClass}__software-pickers`}>
                <DropdownWrapper
                  name="software-title"
                  className={`${baseClass}__row-picker`}
                  isDisabled={gitOpsModeEnabled}
                  value={
                    softwareOptions.find(
                      (o) => o.value === String(softwareTitleId ?? "")
                    ) ?? null
                  }
                  options={softwareOptions}
                  placeholder="Select software"
                  onChange={(opt: SingleValue<CustomOptionType>) =>
                    handleSelectSoftware(opt ? Number(opt.value) : null)
                  }
                />
                {/* Only surfaces for multi-package titles; first-added is
                  auto-selected above, so this is pin-adjustment. */}
                {packageOptions.length > 1 && (
                  <DropdownWrapper
                    name="software-package"
                    className={`${baseClass}__row-picker`}
                    isDisabled={gitOpsModeEnabled}
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
              </div>
            ) : undefined,
        },
        {
          key: "run_script",
          label: "Run script",
          tooltip: (
            <AutomationRowTooltip
              text="The selected script will run when hosts fail the policy. Host counts will reset when new scripts are selected."
              learnMoreUrl="https://fleetdm.com/learn-more-about/policy-automation-run-script"
            />
          ),
          checked: runScript,
          onToggle: handleToggleRunScript,
          isDisabled: false,
          picker: runScript ? (
            <DropdownWrapper
              name="script"
              className={`${baseClass}__row-picker`}
              isDisabled={gitOpsModeEnabled}
              value={
                scriptOptions.find((o) => o.value === String(scriptId ?? "")) ??
                null
              }
              options={scriptOptions}
              placeholder="Select script"
              onChange={(opt: SingleValue<CustomOptionType>) =>
                handleSelectScript(opt ? Number(opt.value) : null)
              }
            />
          ) : undefined,
        },
        {
          key: "resend_configuration_profile",
          label: "Resend configuration profile",
          tooltip: (
            <AutomationRowTooltip
              text="The selected configuration profile will be resent when hosts fail the policy. Host counts will reset when new configuration profile is selected."
              learnMoreUrl={`${LEARN_MORE_ABOUT_BASE_LINK}/policy-automation-resend-configuration-profile`}
            />
          ),
          checked: resendConfigProfile,
          onToggle: handleToggleResendConfigProfile,
          isDisabled: false,
          isLocked: resendProfileDisabled,
          picker: resendConfigProfile ? (
            <DropdownWrapper
              name="profile"
              className={`${baseClass}__row-picker`}
              isDisabled={gitOpsModeEnabled}
              value={
                profileOptions.find((o) => o.value === (profileUUID ?? "")) ??
                null
              }
              options={profileOptions}
              placeholder="Select profile"
              onChange={(opt: SingleValue<CustomOptionType>) =>
                handleSelectProfile(opt ? String(opt.value) : null)
              }
            />
          ) : undefined,
        },
        {
          key: "calendar_event",
          label: "Calendar event",
          tooltip: (
            <AutomationRowTooltip
              text="A calendar event will be created for end users if one of their hosts fail the policy."
              learnMoreUrl="https://www.fleetdm.com/learn-more-about/calendar-events"
            />
          ),
          checked: calendarEvent && isCalendarEnabledForTeam,
          onToggle: setCalendarEvent,
          isDisabled: !isCalendarEnabledForTeam,
        },
        {
          key: "conditional_access",
          label: "Conditional access",
          tooltip: (
            <AutomationRowTooltip
              text="Single sign-on will be blocked for end users whose hosts fail the policy."
              learnMoreUrl="https://fleetdm.com/learn-more-about/conditional-access"
            />
          ),
          checked: conditionalAccess && isConditionalAccessEnabledForTeam,
          onToggle: setConditionalAccess,
          isDisabled: !isConditionalAccessEnabledForTeam,
        }
      );
    }

    const errorMessages = Object.values(errors).filter(
      (msg): msg is string => !!msg
    );

    return (
      <div className={baseClass}>
        <div className={`${baseClass}__section`}>
          {errorMessages.length > 0 && (
            <div className={`${baseClass}__errors`} role="alert">
              {errorMessages.map((msg) => (
                <span key={msg} className={`${baseClass}__error`}>
                  {msg}
                </span>
              ))}
            </div>
          )}
          <table className={`${baseClass}__table`}>
            <tbody>
              {rows.map((row) => (
                <tr
                  key={row.key}
                  className={`${baseClass}__row${
                    row.isDisabled || row.isLocked
                      ? ` ${baseClass}__row--disabled`
                      : ""
                  }`}
                >
                  <td
                    id={
                      row.key === "install_software"
                        ? "install-software-row-label"
                        : undefined
                    }
                    className={`${baseClass}__row-label`}
                  >
                    <GitOpsModeTooltipWrapper
                      renderChildren={(disableChildren) => (
                        <Checkbox
                          name={row.key}
                          value={row.checked}
                          disabled={
                            row.isDisabled || row.isLocked || disableChildren
                          }
                          onChange={row.onToggle}
                        >
                          {row.tooltip ? (
                            <TooltipWrapper tipContent={row.tooltip} clickable>
                              {row.label}
                            </TooltipWrapper>
                          ) : (
                            row.label
                          )}
                        </Checkbox>
                      )}
                    />
                  </td>
                  <td className={`${baseClass}__row-trailing`}>
                    {row.isDisabled ? (
                      <span className={`${baseClass}__row-disabled-hint`}>
                        Not enabled for {fleetName}
                      </span>
                    ) : (
                      row.picker
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className={`${baseClass}__learn-more`}>
            <CustomLink
              url="https://fleetdm.com/learn-more-about/policy-automations"
              text="Learn more"
              newTab
            />{" "}
            about automation types and their supported platforms.
          </div>
        </div>

        {patchSlot}

        {!isGlobalPolicy && patchOption !== "manual" && (
          <div className={`${baseClass}__section`}>
            <GitOpsModeTooltipWrapper
              renderChildren={(disableChildren) => (
                <Checkbox
                  name="continuous-automations-enabled"
                  value={effectiveContinuousEnabled}
                  disabled={disableChildren || patchWhenClosed}
                  onChange={handleToggleContinuous}
                  iconTooltipContent={
                    patchWhenClosed
                      ? "Continuous automation can't be disabled when Patch when app is closed is selected."
                      : undefined
                  }
                  helpText={
                    <>
                      If the install software automation does not resolve the
                      policy after{" "}
                      <TooltipWrapper tipContent="Count does not include skipped installs and failed pre-install queries.">
                        10 attempts
                      </TooltipWrapper>
                      , Fleet will wait 24 hours before retrying.
                    </>
                  }
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

interface IAutomationRowTooltipProps {
  text: string;
  learnMoreUrl: string;
}

function AutomationRowTooltip({
  text,
  learnMoreUrl,
}: IAutomationRowTooltipProps): JSX.Element {
  return (
    <>
      {text}{" "}
      <CustomLink
        url={learnMoreUrl}
        text="Learn more"
        newTab
        variant="tooltip-link"
      />
    </>
  );
}

export default PolicyAutomationsFields;
