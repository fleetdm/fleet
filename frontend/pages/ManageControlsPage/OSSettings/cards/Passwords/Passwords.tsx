import React, { useContext, useState } from "react";
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
import recoveryLockPasswordAPI from "services/entities/recovery_lock_password";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
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
    />
  </>
);

const Passwords = ({ currentTeamId, onMutation }: IOSSettingsCommonProps) => {
  const {
    isPremiumTier,
    config,
    isTeamTechnician,
    isGlobalTechnician,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const isTechnician = isTeamTechnician || isGlobalTechnician;

  const defaultEnableRecoveryLock = currentTeamId
    ? false
    : config?.mdm.enable_recovery_lock_password ?? false;

  const [enableRecoveryLockPassword, setEnableRecoveryLockPassword] = useState(
    defaultEnableRecoveryLock
  );

  const { isLoading: isLoadingTeam } = useQuery<
    ILoadTeamResponse,
    Error,
    ITeamConfig
  >(["team", currentTeamId], () => teamsAPI.load(currentTeamId), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: currentTeamId !== API_NO_TEAM_ID,
    select: (res) => res.team,
    onSuccess: (res) => {
      setEnableRecoveryLockPassword(
        res.mdm?.enable_recovery_lock_password ?? false
      );
    },
  });

  const showLoading = currentTeamId !== API_NO_TEAM_ID && isLoadingTeam;

  const onUpdateRecoveryLockPassword = async () => {
    try {
      // TODO - use response to set state of checkbox, once confirmed it will be there.
      // TODO clarify how this is supposed to work
      await recoveryLockPasswordAPI.updateRecoveryLockPassword(
        enableRecoveryLockPassword,
        currentTeamId
      );
      renderFlash(
        "success",
        "Successfully updated Recovery Lock password enforcement."
      );
      onMutation();
    } catch (e) {
      const errorMsg =
        getErrorReason(e) ??
        "Couldn't enable Recovery Lock password. Please try again.";
      renderFlash("error", errorMsg);
    }
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Passwords" alignLeftHeaderVertically />
      <PageDescription
        variant="right-panel"
        content="Manage passwords used for recovery, security, or administrative access across supported platforms."
      />
      {showLoading && <Spinner />}
      {isPremiumTier && !showLoading && !isTechnician && (
        <div className="form passwords-content">
          <div className={`${baseClass}__recovery-lock-header`}>
            <TooltipWrapper tipContent={RECOVERY_LOCK_TOOLTIP_CONTENT}>
              Recovery Lock password
            </TooltipWrapper>
          </div>
          <Checkbox
            disabled={config?.gitops.gitops_mode_enabled}
            onChange={(value: boolean) => setEnableRecoveryLockPassword(value)}
            value={enableRecoveryLockPassword}
            className={`${baseClass}__checkbox`}
            helpText="This setting is only available on macOS hosts with Apple silicon."
          >
            Turn on Recovery Lock password
          </Checkbox>
          <div className="button-wrap">
            <GitOpsModeTooltipWrapper
              tipOffset={8}
              renderChildren={(disabled) => (
                <Button
                  disabled={disabled}
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
