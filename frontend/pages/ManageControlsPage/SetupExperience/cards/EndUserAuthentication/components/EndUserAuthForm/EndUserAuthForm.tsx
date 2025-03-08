import React, { useContext, useState } from "react";
import { Link } from "react-router";

import PATHS from "router/paths";
import mdmAPI from "services/entities/mdm";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

const baseClass = "end-user-auth-form";

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
          Require end users to authenticate with your identity provider (IdP)
          and agree to an end user license agreement (EULA) when they setup
          their new macOS hosts.{" "}
          <Link to={PATHS.ADMIN_INTEGRATIONS_MDM}>View IdP and EULA</Link>
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
