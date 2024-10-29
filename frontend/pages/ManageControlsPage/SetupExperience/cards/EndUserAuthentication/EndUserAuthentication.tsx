import React from "react";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";
import { useQuery } from "react-query";

import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { IConfig, IMdmConfig } from "interfaces/config";
import { ITeamConfig } from "interfaces/team";

import SectionHeader from "components/SectionHeader/SectionHeader";
import Spinner from "components/Spinner";

import RequireEndUserAuth from "./components/RequireEndUserAuth/RequireEndUserAuth";
import EndUserAuthForm from "./components/EndUserAuthForm/EndUserAuthForm";
import EndUserExperiencePreview from "./components/EndUserExperiencePreview";

const baseClass = "end-user-authentication";

const getEnabledEndUserAuth = (
  currentTeamId: number,
  globalConfig?: IConfig,
  teamConfig?: ITeamConfig
) => {
  if (globalConfig === undefined && teamConfig === undefined) {
    return false;
  }

  // team is "No team" when currentTeamId === 0
  if (currentTeamId === 0) {
    return (
      globalConfig?.mdm?.macos_setup.enable_end_user_authentication ?? false
    );
  }

  return teamConfig?.mdm?.macos_setup.enable_end_user_authentication ?? false;
};

const isIdPConfigured = ({
  end_user_authentication: idp,
}: Pick<IMdmConfig, "end_user_authentication">) => {
  return !!idp.entity_id && !!idp.metadata_url && !!idp.idp_name;
};

interface IEndUserAuthenticationProps {
  currentTeamId: number;
  router: InjectedRouter;
}

const EndUserAuthentication = ({
  currentTeamId,
  router,
}: IEndUserAuthenticationProps) => {
  const { data: globalConfig, isLoading: isLoadingGlobalConfig } = useQuery<
    IConfig,
    Error
  >(["config", currentTeamId], () => configAPI.loadAll(), {
    refetchOnWindowFocus: false,
    retry: false,
  });

  const { data: teamConfig, isLoading: isLoadingTeamConfig } = useQuery<
    ILoadTeamResponse,
    Error,
    ITeamConfig
  >(["team", currentTeamId], () => teamsAPI.load(currentTeamId), {
    refetchOnWindowFocus: false,
    retry: false,
    enabled: currentTeamId !== 0,
    select: (res) => res.team,
  });

  const defaultIsEndUserAuthEnabled = getEnabledEndUserAuth(
    currentTeamId,
    globalConfig,
    teamConfig
  );

  const onClickConnect = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="End user authentication" />
      {isLoadingGlobalConfig || isLoadingTeamConfig ? (
        <Spinner />
      ) : (
        <div className={`${baseClass}__content`}>
          {!globalConfig || !isIdPConfigured(globalConfig.mdm) ? (
            <RequireEndUserAuth onClickConnect={onClickConnect} />
          ) : (
            <EndUserAuthForm
              currentTeamId={currentTeamId}
              defaultIsEndUserAuthEnabled={defaultIsEndUserAuthEnabled}
            />
          )}
          <EndUserExperiencePreview />
        </div>
      )}
    </div>
  );
};

export default EndUserAuthentication;
