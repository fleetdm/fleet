import React, { useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import { getErrorReason } from "interfaces/errors";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import configAPI from "services/entities/config";

import PATHS from "router/paths";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import EmptyState from "components/EmptyState";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";
import PageDescription from "components/PageDescription";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import { IOSSettingsCommonProps } from "../../OSSettingsNavItems";

const baseClass = "byod-permissions";

const BYODPermissions = ({
  currentTeamId,
  router,
  onMutation,
}: IOSSettingsCommonProps) => {
  const {
    isPremiumTier,
    config,
    isTeamTechnician,
    isGlobalTechnician,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const isTechnician = isTeamTechnician || isGlobalTechnician;

  // Apple MDM only — BYOD permission gating is Apple-specific.
  const mdmEnabled = config?.mdm.enabled_and_configured;

  const [allowWipe, setAllowWipe] = useState<boolean | undefined>(undefined);
  const [allowLock, setAllowLock] = useState<boolean | undefined>(undefined);
  const [updating, setUpdating] = useState(false);

  const {
    isLoading: isLoadingTeam,
    isSuccess: isTeamSuccess,
    isError: isTeamError,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["team", currentTeamId],
    () => teamsAPI.load(currentTeamId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: currentTeamId !== API_NO_TEAM_ID,
      select: (res) => res.fleet,
      onSuccess: (res) => {
        setAllowWipe(res.mdm?.allow_byod_wipe ?? true);
        setAllowLock(res.mdm?.allow_byod_lock ?? true);
      },
      onError: () => {
        renderFlash("error", "Couldn't load fleet settings. Please try again.");
      },
    }
  );

  // Sync state from global config when "no fleet" is selected.
  useEffect(() => {
    if (currentTeamId === API_NO_TEAM_ID) {
      setAllowWipe(config?.mdm.allow_byod_wipe ?? true);
      setAllowLock(config?.mdm.allow_byod_lock ?? true);
    }
  }, [currentTeamId, config?.mdm.allow_byod_wipe, config?.mdm.allow_byod_lock]);

  const isTeamQuery = currentTeamId !== API_NO_TEAM_ID;
  const showLoading = isTeamQuery && isLoadingTeam;
  const isFormReady = !isTeamQuery || (isTeamSuccess && !isLoadingTeam);
  const isFormDisabled =
    !isFormReady ||
    isTeamError ||
    allowWipe === undefined ||
    allowLock === undefined;

  const onSave = async () => {
    setUpdating(true);
    try {
      if (currentTeamId === API_NO_TEAM_ID) {
        await configAPI.update({
          mdm: {
            allow_byod_wipe: allowWipe,
            allow_byod_lock: allowLock,
          },
        });
      } else {
        await teamsAPI.updateConfig(
          {
            mdm: {
              allow_byod_wipe: allowWipe,
              allow_byod_lock: allowLock,
            },
          },
          currentTeamId
        );
      }
      renderFlash("success", "Successfully updated BYOD permissions.");
      onMutation();
    } catch (e) {
      const errorMsg =
        getErrorReason(e) ??
        "Couldn't update BYOD permissions. Please try again.";
      renderFlash("error", errorMsg);
    } finally {
      setUpdating(false);
    }
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="BYOD permissions" alignLeftHeaderVertically />
      <PageDescription
        variant="right-panel"
        content="Control what Fleet can do to manually enrolled (BYOD) iPhones, iPads, and Macs. Already-enrolled BYOD hosts narrow to these permissions at their next SCEP certificate renewal. Re-enabling a permission later does not restore it on already-enrolled hosts — they would need to re-enroll."
      />
      {!isPremiumTier && <PremiumFeatureMessage />}
      {isPremiumTier && mdmEnabled === undefined && <Spinner />}
      {isPremiumTier && mdmEnabled === false && (
        <EmptyState
          variant="form"
          header="Manage your hosts"
          info="MDM must be turned on to apply BYOD permissions."
          primaryButton={
            <Button onClick={() => router.push(PATHS.ADMIN_INTEGRATIONS_MDM)}>
              Turn on
            </Button>
          }
        />
      )}
      {isPremiumTier && mdmEnabled === true && showLoading && <Spinner />}
      {isPremiumTier && mdmEnabled === true && !showLoading && !isTechnician && (
        <div className={`form ${baseClass}-content`}>
          <Checkbox
            disabled={isFormDisabled || config?.gitops.gitops_mode_enabled}
            onChange={(value: boolean) => setAllowWipe(value)}
            value={allowWipe ?? true}
            className={`${baseClass}__checkbox`}
            helpText="Lets Fleet erase BYOD hosts. Turn off to prevent wipe on manually enrolled iPhones, iPads, and Macs in this fleet."
          >
            Allow wipe
          </Checkbox>
          <Checkbox
            disabled={isFormDisabled || config?.gitops.gitops_mode_enabled}
            onChange={(value: boolean) => setAllowLock(value)}
            value={allowLock ?? true}
            className={`${baseClass}__checkbox`}
            helpText="Lets Fleet lock BYOD hosts. Turn off to prevent lock on manually enrolled Macs in this fleet. Manual iOS/iPadOS hosts cannot be locked regardless of this setting."
          >
            Allow lock
          </Checkbox>
          <div className="button-wrap">
            <GitOpsModeTooltipWrapper
              tipOffset={8}
              renderChildren={(gitopsDisabled) => (
                <Button
                  disabled={isFormDisabled || gitopsDisabled}
                  isLoading={updating}
                  className={`${baseClass}__save-button`}
                  onClick={onSave}
                >
                  Save
                </Button>
              )}
            />
          </div>
        </div>
      )}
    </div>
  );
};

export default BYODPermissions;
