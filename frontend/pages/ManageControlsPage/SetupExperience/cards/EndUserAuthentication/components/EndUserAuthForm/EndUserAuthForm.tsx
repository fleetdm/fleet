import React, { useContext, useState } from "react";
import classnames from "classnames";

import PATHS from "router/paths";
import mdmAPI from "services/entities/mdm";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TooltipWrapper from "components/TooltipWrapper";
import RevealButton from "components/buttons/RevealButton";

const baseClass = "end-user-auth-form";

const getTooltipCopy = (android = false) => {
  return (
    <>
      {android ? "Android" : "Apple"} MDM must be turned on in <b>Settings</b>{" "}
      &gt; <b>Integrations</b> &gt; <b>Mobile Device Management (MDM)</b> to
      turn on end user authentication.
    </>
  );
};
interface IEndUserAuthFormProps {
  currentTeamId: number;
  defaultIsEndUserAuthEnabled: boolean;
  defaultLockEndUserInfo: boolean;
}

const EndUserAuthForm = ({
  currentTeamId,
  defaultIsEndUserAuthEnabled,
  defaultLockEndUserInfo,
}: IEndUserAuthFormProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const gitOpsModeEnabled = useContext(AppContext).config?.gitops
    .gitops_mode_enabled;

  const [isEndUserAuthEnabled, setEndUserAuthEnabled] = useState(
    defaultIsEndUserAuthEnabled
  );
  const [lockEndUserInfo, setLockEndUserInfo] = useState(
    defaultLockEndUserInfo
  );
  const [isUpdating, setIsUpdating] = useState(false);
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

  const onToggleEndUserAuth = (newCheckVal: boolean) => {
    setEndUserAuthEnabled(newCheckVal);
  };

  const onChangeLockEndUserInfo = (newCheckVal: boolean) => {
    setLockEndUserInfo(newCheckVal);
  };

  const onClickSave = async () => {
    setIsUpdating(true);
    const canLockEndUserInfo = isEndUserAuthEnabled && lockEndUserInfo;
    try {
      await mdmAPI.updateEndUserAuthentication(
        currentTeamId,
        isEndUserAuthEnabled,
        canLockEndUserInfo
      );
      renderFlash("success", "Successfully updated.");
    } catch {
      renderFlash("error", "Couldn’t update. Please try again.");
    } finally {
      setIsUpdating(false);
      setLockEndUserInfo(canLockEndUserInfo);
    }
  };

  const classes = classnames({ [`${baseClass}--disabled`]: gitOpsModeEnabled });
  return (
    <div className={baseClass}>
      <form>
        <p className={classes}>
          Require end users to authenticate with your{" "}
          <CustomLink
            url={PATHS.ADMIN_INTEGRATIONS_SSO_END_USERS}
            text="identity provider (IdP)"
          />{" "}
          when they set up their new hosts.{" "}
          <TooltipWrapper tipContent={getTooltipCopy()}>macOS</TooltipWrapper>{" "}
          hosts will also be required to agree to an{" "}
          <CustomLink
            url={`${PATHS.ADMIN_INTEGRATIONS_MDM}#end-user-license-agreement`}
            text="end user license agreement (EULA)"
          />{" "}
          if configured.
        </p>
        <Checkbox
          disabled={gitOpsModeEnabled}
          value={isEndUserAuthEnabled}
          onChange={onToggleEndUserAuth}
        >
          Turn on
        </Checkbox>
        <RevealButton
          isShowing={showAdvancedOptions}
          showText="Advanced options"
          hideText="Advanced options"
          caretPosition="after"
          onClick={() => setShowAdvancedOptions(!showAdvancedOptions)}
        />
        {showAdvancedOptions && (
          <Checkbox
            disabled={!isEndUserAuthEnabled}
            onChange={onChangeLockEndUserInfo}
            value={lockEndUserInfo}
          >
            <TooltipWrapper
              tipContent={
                <span>
                  End user can&apos;t edit the local account&apos;s{" "}
                  <b>Account Name</b> and
                  <br />
                  <b>Full Name</b> in macOS Setup Assistant. These fields will
                  be
                  <br />
                  locked to values from your IdP.
                </span>
              }
            >
              Lock end user info
            </TooltipWrapper>
          </Checkbox>
        )}

        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              isLoading={isUpdating}
              onClick={onClickSave}
            >
              Save
            </Button>
          )}
        />
      </form>
    </div>
  );
};

export default EndUserAuthForm;
