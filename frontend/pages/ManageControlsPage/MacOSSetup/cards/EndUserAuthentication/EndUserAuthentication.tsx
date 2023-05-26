import React, { useContext } from "react";
import { Link } from "react-router";
import PATHS from "router/paths";

import configAPI from "services/entities/config";
import mdmAPI from "services/entities/mdm";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { IConfig } from "interfaces/config";

import SectionHeader from "components/SectionHeader/SectionHeader";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox/Checkbox";
import EndUserExperiencePreview from "pages/ManageControlsPage/components/EndUserExperiencePreview";
import { useQuery } from "react-query";
import { ITeamConfig } from "interfaces/team";
import Spinner from "components/Spinner";
import { NotificationContext } from "context/notification";

const baseClass = "end-user-authentication";

interface IEndUserAuthenticationProps {
  currentTeamId: number;
}

const EndUserAuthentication = ({
  currentTeamId,
}: IEndUserAuthenticationProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [isEndUserAuthEnabled, setEndUserAuthEnabled] = React.useState(false);
  const [isUpdating, setIsUpdating] = React.useState(false);

  const { isLoading: isLoadingGlobalConfig } = useQuery<IConfig, Error>(
    ["config"],
    () => configAPI.loadAll(),
    {
      refetchOnWindowFocus: false,
      retry: false,
      enabled: currentTeamId === 0,
      onSuccess: (res) => {
        setEndUserAuthEnabled(
          res.mdm?.macos_setup.enable_end_user_authentication ?? false
        );
      },
    }
  );

  const { isLoading: isLoadingTeamConfig } = useQuery<ILoadTeamResponse, Error>(
    ["team", currentTeamId],
    () => teamsAPI.load(currentTeamId),
    {
      refetchOnWindowFocus: false,
      retry: false,
      enabled: currentTeamId !== 0,
      onSuccess: (res) => {
        setEndUserAuthEnabled(
          res.team.mdm?.macos_setup.enable_end_user_authentication ?? false
        );
      },
    }
  );

  const onToggleEndUserAuth = (newCheckVal: boolean) => {
    setEndUserAuthEnabled(newCheckVal);
  };

  const onClickSave = async (e: React.FormEvent<SubmitEvent>) => {
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
      <SectionHeader title="End user authentication" />
      {isLoadingGlobalConfig || isLoadingTeamConfig ? (
        <Spinner />
      ) : (
        <div className={`${baseClass}__content`}>
          <form>
            <Checkbox
              value={isEndUserAuthEnabled}
              onChange={onToggleEndUserAuth}
            >
              On
            </Checkbox>
            <p>
              Require end users to authenticate with your identity provider
              (IdP) and agree to an end user license agreement (EULA) when they
              setup their new macOS hosts.{" "}
              <Link to={PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT}>
                View IdP and EULA
              </Link>
            </p>
            <Button isLoading={isUpdating} onClick={onClickSave}>
              Save
            </Button>
          </form>
          <EndUserExperiencePreview previewImage="">
            <p>
              When the end user reaches the <b>Remote Management</b> pane in the
              macOS Setup Assistant, they are asked to authenticate and agree to
              the end user license agreement (EULA).
            </p>
            <p>
              After, Fleet enrolls the Mac, applies macOS settings, and installs
              the bootstrap package.
            </p>
          </EndUserExperiencePreview>
        </div>
      )}
    </div>
  );
};

export default EndUserAuthentication;
