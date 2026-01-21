import React, { useContext, useState } from "react";

import PATHS from "router/paths";
import mdmAPI from "services/entities/mdm";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import TooltipWrapper from "components/TooltipWrapper";

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
  defaultLockPrimaryAccountInfo: boolean;
}

const EndUserAuthForm = ({
  currentTeamId,
  defaultIsEndUserAuthEnabled,
  defaultLockPrimaryAccountInfo,
}: IEndUserAuthFormProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const gitOpsModeEnabled = useContext(AppContext).config?.gitops
    .gitops_mode_enabled;

  const [isEndUserAuthEnabled, setEndUserAuthEnabled] = useState(
    defaultIsEndUserAuthEnabled
  );
  const [lockPrimaryAccountInfo, setLockPrimaryAccountInfo] = useState(
    defaultLockPrimaryAccountInfo
  );
  const [isUpdating, setIsUpdating] = useState(false);

  const onToggleEndUserAuth = (newCheckVal: boolean) => {
    setEndUserAuthEnabled(newCheckVal);
  };

  const onToggleLockPrimaryAccountInfo = (newCheckVal: boolean) => {
    setLockPrimaryAccountInfo(newCheckVal);
  };

  const onClickSave = async () => {
    setIsUpdating(true);
    try {
      await mdmAPI.updateEndUserAuthentication(
        currentTeamId,
        isEndUserAuthEnabled,
        lockPrimaryAccountInfo
      );
      renderFlash("success", "Successfully updated!");
    } catch {
      renderFlash("error", "Couldn't update. Please try again.");
    } finally {
      setIsUpdating(false);
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
          when they set up their new hosts.
          <br />
          <TooltipWrapper tipContent={getTooltipCopy()}>
            macOS
          </TooltipWrapper>{" "}
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
        <Checkbox
          disabled={gitOpsModeEnabled}
          value={lockPrimaryAccountInfo}
          onChange={onToggleLockPrimaryAccountInfo}
        >
          Lock primary account information
        </Checkbox>
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
