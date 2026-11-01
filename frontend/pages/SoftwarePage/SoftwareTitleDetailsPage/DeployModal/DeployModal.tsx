import React, { useState } from "react";

import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { getErrorReason } from "interfaces/errors";
import { isMacOS } from "interfaces/platform";
import {
  getInstallablePlatform,
  ISoftwareTitleDetails,
} from "interfaces/software";
import {
  EndUserExperience,
  getNextEndUserExperience,
  getPatchPolicyFlags,
  PatchOption,
  SoftwareDeploySelector,
} from "pages/SoftwarePage/components/forms/SoftwareDeploySelector";
import {
  getFleetAppPolicyDescription,
  getFleetAppPolicyName,
} from "pages/SoftwarePage/SoftwareAddPage/SoftwareFleetMaintained/FleetMaintainedAppDetailsPage/helpers";
import softwareAPI from "services/entities/software";
import teamPoliciesAPI from "services/entities/team_policies";

const baseClass = "deploy-modal";

interface IDeployModalProps {
  softwareTitle: ISoftwareTitleDetails;
  teamId: number;
  onExit: () => void;
  onSuccess: () => void;
}

const DeployModal = ({
  softwareTitle,
  teamId,
  onExit,
  onSuccess,
}: IDeployModalProps) => {
  const softwarePackage = softwareTitle.software_package;
  const automaticInstallPolicies =
    softwarePackage?.automatic_install_policies ?? [];
  const patchPolicy = softwarePackage?.patch_policy;
  const forceInstallPolicy = automaticInstallPolicies.find(
    (policy) =>
      policy.type === "dynamic" &&
      policy.name === getFleetAppPolicyName(softwareTitle.name)
  );
  const patchHasAutomation =
    !!patchPolicy &&
    automaticInstallPolicies.some((policy) => policy.id === patchPolicy.id);
  // With no existing patch policy, default to the top option ("closed") so a
  // newly-checked Patch matches the Add software page. An existing policy
  // reflects its real state instead.
  let initialPatchOption: PatchOption = patchPolicy ? "manual" : "closed";
  if (patchPolicy?.patch_when_closed) {
    initialPatchOption = "closed";
  } else if (patchHasAutomation) {
    initialPatchOption = "force";
  }

  const platform = getInstallablePlatform(softwareTitle.source);

  // If a legacy policy stored multiple flags (invariant violation), Patch when
  // closed wins the radio and the dropdown stays hidden — carrying "notify" or
  // "deadline" in state would reveal it the moment the user picked Force patch.
  // Deadline is also the default when nothing is set on macOS, matching the
  // add form. Non-macOS titles never show the dropdown and never send
  // patch_deadline/notify_before_patching, so they default to "immediate".
  const getInitialEndUserExperience = (): EndUserExperience => {
    if (patchPolicy?.patch_when_closed) {
      return isMacOS(platform) ? "deadline" : "immediate";
    }
    if (patchPolicy?.patch_deadline) return "deadline";
    if (patchPolicy?.notify_before_patching) return "notify";
    if (patchPolicy) return "immediate";
    return isMacOS(platform) ? "deadline" : "immediate";
  };
  const initialEndUserExperience = getInitialEndUserExperience();

  const [forceInstall, setForceInstall] = useState(!!forceInstallPolicy);
  const [patch, setPatch] = useState(!!patchPolicy);
  const [patchOption, setPatchOption] = useState<PatchOption>(
    initialPatchOption
  );
  const [endUserExperience, setEndUserExperience] = useState<EndUserExperience>(
    initialEndUserExperience
  );
  const [isSaving, setIsSaving] = useState(false);

  const fleetMaintainedAppId = softwarePackage?.fleet_maintained_app_id;

  const onSave = async () => {
    setIsSaving(true);
    let savedAnyChange = false;
    const patchFlags = getPatchPolicyFlags(patchOption, endUserExperience);
    try {
      if (forceInstall !== !!forceInstallPolicy) {
        if (forceInstall) {
          const {
            fleet_maintained_app: fleetMaintainedApp,
          } = await softwareAPI.getFleetMaintainedApp(
            fleetMaintainedAppId as number,
            String(teamId)
          );
          if (!fleetMaintainedApp?.automatic_install_query) {
            // A plain Error wouldn't survive getErrorReason in the catch below
            // (it only unwraps API responses), so surface the toast directly
            // and bail — the finally resets isSaving.
            notify.error(
              "Couldn't create the Force install policy. Try again."
            );
            return;
          }
          await teamPoliciesAPI.create({
            team_id: teamId,
            name: getFleetAppPolicyName(softwareTitle.name),
            description: getFleetAppPolicyDescription(softwareTitle.name),
            query: fleetMaintainedApp.automatic_install_query,
            platform: fleetMaintainedApp.platform,
            software_title_id: softwareTitle.id,
          });
        } else if (forceInstallPolicy) {
          await teamPoliciesAPI.destroy(teamId, [forceInstallPolicy.id]);
        }
        savedAnyChange = true;
      }

      if (patch !== !!patchPolicy) {
        if (patch) {
          await teamPoliciesAPI.create({
            team_id: teamId,
            type: "patch",
            patch_software_title_id: softwareTitle.id,
            ...(patchOption !== "manual" && {
              software_title_id: softwareTitle.id,
            }),
            ...patchFlags,
          });
        } else if (patchPolicy) {
          await teamPoliciesAPI.destroy(teamId, [patchPolicy.id]);
        }
        savedAnyChange = true;
      } else if (
        patch &&
        patchPolicy &&
        // Normalize undefined to false on the stored side: legacy policies
        // (notify_before_patching, patch_deadline) can arrive without the
        // field at all, and strict !== would otherwise flag undefined vs
        // false as a change and fire a no-op PATCH.
        (patchOption !== initialPatchOption ||
          endUserExperience !== initialEndUserExperience ||
          patchHasAutomation !== (patchOption !== "manual") ||
          (patchPolicy.patch_when_closed ?? false) !==
            patchFlags.patch_when_closed ||
          (patchPolicy.notify_before_patching ?? false) !==
            patchFlags.notify_before_patching ||
          (patchPolicy.patch_deadline ?? false) !== patchFlags.patch_deadline ||
          (patchPolicy.continuous_automations_enabled ?? false) !==
            patchFlags.continuous_automations_enabled)
      ) {
        await teamPoliciesAPI.update(patchPolicy.id, {
          team_id: teamId,
          software_title_id: patchOption === "manual" ? null : softwareTitle.id,
          ...patchFlags,
        });
        savedAnyChange = true;
      }

      if (savedAnyChange) {
        notify.success("Successfully updated deploy options.");
      }
      onSuccess();
      onExit();
    } catch (error) {
      if (savedAnyChange) {
        // A partial save (e.g. Force install created, then the patch policy
        // failed): tell the user some changes landed — mirroring the Add
        // flow — rather than only the raw error, then refresh + close so a
        // retry starts from the real state.
        notify.error(
          "Some changes were saved, but others couldn't be. Try again.",
          { response: error }
        );
        onSuccess();
        onExit();
      } else {
        notify.error(getErrorReason(error), { response: error });
      }
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Modal
      className={baseClass}
      title="Deploy"
      onExit={onExit}
      isContentDisabled={isSaving}
    >
      <>
        <GitOpsModeTooltipWrapper
          entityType="software"
          renderChildren={(disableChildren) => (
            <SoftwareDeploySelector
              forceInstall={forceInstall}
              patch={patch}
              patchOption={patchOption}
              platform={platform}
              endUserExperience={endUserExperience}
              fleetId={teamId}
              onToggleForceInstall={setForceInstall}
              onTogglePatch={setPatch}
              onSelectPatchOption={(next) => {
                setPatchOption(next);
                setEndUserExperience(
                  getNextEndUserExperience(next, endUserExperience, platform)
                );
              }}
              onSelectEndUserExperience={setEndUserExperience}
              disabled={disableChildren}
              showPatchWhenClosedNotice
              hideLabel
            />
          )}
        />
        <div className="modal-cta-wrap">
          <GitOpsModeTooltipWrapper
            entityType="software"
            position="top"
            tipOffset={8}
            renderChildren={(disableChildren) => (
              <Button
                onClick={onSave}
                isLoading={isSaving}
                disabled={disableChildren}
              >
                Save
              </Button>
            )}
          />
          <Button variant="secondary" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeployModal;
