import React, { useState } from "react";
import { useQuery } from "react-query";

import { ISoftwareTitleDetails } from "interfaces/software";
import { getErrorReason } from "interfaces/errors";
import softwareAPI from "services/entities/software";
import teamPoliciesAPI from "services/entities/team_policies";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import {
  getPatchPolicyFlags,
  PatchOption,
  SoftwareDeploySelector,
} from "pages/SoftwarePage/components/forms/SoftwareDeploySelector";
import {
  getFleetAppPolicyDescription,
  getFleetAppPolicyName,
} from "pages/SoftwarePage/SoftwareAddPage/SoftwareFleetMaintained/FleetMaintainedAppDetailsPage/helpers";

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
  let initialPatchOption: PatchOption = "manual";
  if (patchPolicy?.patch_when_closed) {
    initialPatchOption = "closed";
  } else if (patchHasAutomation) {
    initialPatchOption = "force";
  }

  const [forceInstall, setForceInstall] = useState(!!forceInstallPolicy);
  const [patch, setPatch] = useState(!!patchPolicy);
  const [patchOption, setPatchOption] = useState<PatchOption>(
    initialPatchOption
  );
  const [isSaving, setIsSaving] = useState(false);

  const fleetMaintainedAppId = softwarePackage?.fleet_maintained_app_id;
  const {
    data: fleetMaintainedApp,
    isLoading: isLoadingFleetMaintainedApp,
  } = useQuery(
    ["fleet-maintained-app", fleetMaintainedAppId, teamId],
    () =>
      softwareAPI.getFleetMaintainedApp(
        fleetMaintainedAppId as number,
        String(teamId)
      ),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: !!fleetMaintainedAppId && !forceInstallPolicy,
      select: (res) => res.fleet_maintained_app,
    }
  );

  const onSave = async () => {
    setIsSaving(true);
    let savedAnyChange = false;
    try {
      if (forceInstall !== !!forceInstallPolicy) {
        if (forceInstall) {
          if (!fleetMaintainedApp?.automatic_install_query) {
            throw new Error(
              "Couldn't create the Force install policy. Try again."
            );
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
            ...getPatchPolicyFlags(patchOption),
          });
        } else if (patchPolicy) {
          await teamPoliciesAPI.destroy(teamId, [patchPolicy.id]);
        }
        savedAnyChange = true;
      } else if (
        patch &&
        patchPolicy &&
        (patchOption !== initialPatchOption ||
          patchHasAutomation !== (patchOption !== "manual") ||
          patchPolicy.patch_when_closed !==
            getPatchPolicyFlags(patchOption).patch_when_closed ||
          patchPolicy.continuous_automations_enabled !==
            getPatchPolicyFlags(patchOption).continuous_automations_enabled)
      ) {
        await teamPoliciesAPI.update(patchPolicy.id, {
          team_id: teamId,
          software_title_id: patchOption === "manual" ? null : softwareTitle.id,
          ...getPatchPolicyFlags(patchOption),
        });
        savedAnyChange = true;
      }

      onSuccess();
      onExit();
    } catch (error) {
      notify.error(getErrorReason(error), { response: error });
      if (savedAnyChange) {
        onSuccess();
        onExit();
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
              onToggleForceInstall={setForceInstall}
              onTogglePatch={setPatch}
              onSelectPatchOption={setPatchOption}
              disabled={disableChildren || isLoadingFleetMaintainedApp}
              showPatchWhenClosedNotice
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
                disabled={disableChildren || isLoadingFleetMaintainedApp}
              >
                Save
              </Button>
            )}
          />
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeployModal;
