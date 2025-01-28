import React, { useContext, useState } from "react";
import { Link } from "react-router";

import PATHS from "router/paths";
import mdmAPI from "services/entities/mdm";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import { NotificationContext } from "context/notification";

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

  return (
    <div className={baseClass}>
      <form>
        <Checkbox value={isEndUserAuthEnabled} onChange={onToggleEndUserAuth}>
          Turn on
        </Checkbox>
        <p>
          Require end users to authenticate with your identity provider (IdP)
          and agree to an end user license agreement (EULA) when they setup
          their new macOS hosts.{" "}
          <Link to={PATHS.ADMIN_INTEGRATIONS_MDM}>View IdP and EULA</Link>
        </p>
        <Button isLoading={isUpdating} onClick={onClickSave}>
          Save
        </Button>
      </form>
    </div>
  );
};

export default EndUserAuthForm;
