import React, { useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import { getErrorReason } from "interfaces/errors";

import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import configAPI from "services/entities/config";

import PATHS from "router/paths";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import EmptyState from "components/EmptyState";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";
import PageDescription from "components/PageDescription";
import TooltipWrapper from "components/TooltipWrapper";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import { IOSSettingsCommonProps } from "../../OSSettingsNavItems";

const baseClass = "passwords";

const RECOVERY_LOCK_TOOLTIP_CONTENT = (
  <>
    Configure and escrow macOS Recovery Lock passwords. These restrict access to
    recoveryOS and are securely stored for authorized admin retrieval.{" "}
    <CustomLink
      text="Learn more"
      url={`${LEARN_MORE_ABOUT_BASE_LINK}/recovery-lock-passwords`}
      newTab
      variant="tooltip-link"
    />
  </>
);

const Passwords = ({
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

  // Recovery Lock is macOS only, so we only check for macOS MDM
  const mdmEnabled = config?.mdm.enabled_and_configured;

  const [enableRecoveryLockPassword, setEnableRecoveryLockPassword] = useState<
    boolean | undefined
  >(undefined);
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
        setEnableRecoveryLockPassword(
          res.mdm?.enable_recovery_lock_password ?? false
        );
      },
      onError: () => {
        renderFlash("error", "Couldn't load team settings. Please try again.");
      },
    }
  );

  // Sync state from global config when "no team" is selected
  useEffect(() => {
    if (currentTeamId === API_NO_TEAM_ID) {
      setEnableRecoveryLockPassword(
        config?.mdm.enable_recovery_lock_password ?? false
      );
    }
  }, [currentTeamId, config?.mdm.enable_recovery_lock_password]);

  const isTeamQuery = currentTeamId !== API_NO_TEAM_ID;
  const showLoading = isTeamQuery && isLoadingTeam;
  const isFormReady = !isTeamQuery || (isTeamSuccess && !isLoadingTeam);
  const isFormDisabled =
    !isFormReady || isTeamError || enableRecoveryLockPassword === undefined;

  const onUpdateRecoveryLockPassword = async () => {
    setUpdating(true);
    try {
      if (currentTeamId === API_NO_TEAM_ID) {
        await configAPI.update({
          mdm: { enable_recovery_lock_password: enableRecoveryLockPassword },
        });
      } else {
        await teamsAPI.updateConfig(
          {
            mdm: { enable_recovery_lock_password: enableRecoveryLockPassword },
          },
          currentTeamId
        );
      }
      renderFlash(
        "success",
        "Successfully updated Recovery Lock password enforcement."
      );
      onMutation();
    } catch (e) {
      const errorMsg =
        getErrorReason(e) ??
        "Couldn't update Recovery Lock password enforcement. Please try again.";
      renderFlash("error", errorMsg);
    } finally {
      setUpdating(false);
    }
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Passwords" alignLeftHeaderVertically />
      <PageDescription
        variant="right-panel"
        content="Manage passwords used for recovery, security, or administrative access across supported platforms."
      />
      {!isPremiumTier && <PremiumFeatureMessage />}
      {isPremiumTier && mdmEnabled === undefined && <Spinner />}
      {isPremiumTier && mdmEnabled === false && (
        <EmptyState
          variant="form"
          header="Manage your hosts"
          info="MDM must be turned on to apply password settings."
          primaryButton={
            <Button onClick={() => router.push(PATHS.ADMIN_INTEGRATIONS_MDM)}>
              Turn on
            </Button>
          }
        />
      )}
      {isPremiumTier && mdmEnabled === true && showLoading && <Spinner />}
      {isPremiumTier && mdmEnabled === true && !showLoading && !isTechnician && (
        <div className="form passwords-content">
          <div className={`${baseClass}__recovery-lock-header`}>
            <TooltipWrapper tipContent={RECOVERY_LOCK_TOOLTIP_CONTENT}>
              Recovery Lock password
            </TooltipWrapper>
          </div>
          <Checkbox
            disabled={isFormDisabled || config?.gitops.gitops_mode_enabled}
            onChange={(value: boolean) => setEnableRecoveryLockPassword(value)}
            value={enableRecoveryLockPassword ?? false}
            className={`${baseClass}__checkbox`}
            helpText="This setting is only available on macOS hosts with Apple silicon."
          >
            Turn on Recovery Lock password
          </Checkbox>
          <div className="button-wrap">
            <GitOpsModeTooltipWrapper
              tipOffset={8}
              renderChildren={(gitopsDisabled) => (
                <Button
                  disabled={isFormDisabled || gitopsDisabled}
                  isLoading={updating}
                  className={`${baseClass}__save-button`}
                  onClick={onUpdateRecoveryLockPassword}
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

export default Passwords;
