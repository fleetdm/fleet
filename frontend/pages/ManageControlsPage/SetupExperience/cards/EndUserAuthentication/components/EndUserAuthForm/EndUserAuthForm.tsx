import React, { useContext, useState } from "react";

import PATHS from "router/paths";
import mdmAPI from "services/entities/mdm";
import classnames from "classnames";

import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import TooltipWrapper from "components/TooltipWrapper";
import { Link } from "react-router";

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
}

const EndUserAuthForm = ({
  currentTeamId,
  defaultIsEndUserAuthEnabled,
}: IEndUserAuthFormProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const gitOpsModeEnabled = useContext(AppContext).config?.gitops
    .gitops_mode_enabled;

  const [isEndUserAuthEnabled, setEndUserAuthEnabled] = useState(
    defaultIsEndUserAuthEnabled
  );
  const [isUpdating, setIsUpdating] = useState(false);

  const onToggleEndUserAuth = (newCheckVal: boolean) => {
    setEndUserAuthEnabled(newCheckVal);
  };

  const onClickSave = async () => {
    setIsUpdating(true);
    try {
      await mdmAPI.updateEndUserAuthentication(
        currentTeamId,
        isEndUserAuthEnabled
      );
      renderFlash("success", "Successfully updated!");
    } catch {
      renderFlash("error", "Couldn’t update. Please try again.");
    } finally {
      setIsUpdating(false);
    }
  };

  const classes = classnames({ [`${baseClass}--disabled`]: gitOpsModeEnabled });
  return (
    <div className={baseClass}>
      <form>
        <Checkbox
          disabled={gitOpsModeEnabled}
          value={isEndUserAuthEnabled}
          onChange={onToggleEndUserAuth}
        >
          Turn on
        </Checkbox>
        <p className={classes}>
          Require end users to authenticate with your{" "}
          <Link to={PATHS.ADMIN_INTEGRATIONS_SSO_END_USER}>
            identity provider (IdP)
          </Link>{" "}
          when they set up their new hosts.{" "}
          <TooltipWrapper tipContent={getTooltipCopy()}>macOS</TooltipWrapper>{" "}
          hosts will also be required to agree to an{" "}
          <Link
            to={`${PATHS.ADMIN_INTEGRATIONS_MDM}#end-user-license-agreement`}
          >
            end user license agreement (EULA)
          </Link>{" "}
          if configured.
        </p>
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
