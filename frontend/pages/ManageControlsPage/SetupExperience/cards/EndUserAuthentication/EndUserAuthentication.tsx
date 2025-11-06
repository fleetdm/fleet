import React from "react";
import PATHS from "router/paths";
import { useQuery } from "react-query";

import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { IConfig, IMdmConfig } from "interfaces/config";
import { ITeamConfig } from "interfaces/team";

import SectionHeader from "components/SectionHeader/SectionHeader";
import Spinner from "components/Spinner";
import TurnOnMdmMessage from "components/TurnOnMdmMessage";
import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import RequireEndUserAuth from "./components/RequireEndUserAuth/RequireEndUserAuth";
import EndUserAuthForm from "./components/EndUserAuthForm/EndUserAuthForm";
import SetupExperienceContentContainer from "../../components/SetupExperienceContentContainer";
import { ISetupExperienceCardProps } from "../../SetupExperienceNavItems";

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
  return (
    !!idp.entity_id && !!idp.idp_name && (!!idp.metadata_url || !!idp.metadata)
  );
};

const EndUserAuthentication = ({
  currentTeamId,
  router,
}: ISetupExperienceCardProps) => {
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
    router.push(PATHS.ADMIN_INTEGRATIONS_IDENTITY_PROVIDER);
  };

  const renderContent = () => {
    if (!globalConfig || isLoadingGlobalConfig || isLoadingTeamConfig) {
      return <Spinner />;
    }
    const mdmConfig = globalConfig.mdm;
    if (
      !(
        mdmConfig.enabled_and_configured ||
        mdmConfig.android_enabled_and_configured
      )
    ) {
      return (
        <TurnOnMdmMessage
          header="Additional configuration required"
          info="Supported on macOS, iOS, iPadOS, and Android. To customize, first turn on MDM."
          buttonText="Turn on"
          router={router}
        />
      );
    }
    return (
      <SetupExperienceContentContainer>
        {!isIdPConfigured(mdmConfig) ? (
          <RequireEndUserAuth onClickConnect={onClickConnect} />
        ) : (
          <EndUserAuthForm
            currentTeamId={currentTeamId}
            defaultIsEndUserAuthEnabled={defaultIsEndUserAuthEnabled}
          />
        )}
      </SetupExperienceContentContainer>
    );
  };

  return (
    <section className={baseClass}>
      <SectionHeader
        title="End user authentication"
        details={
          <CustomLink
            newTab
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/setup-experience/end-user-authentication`}
            text="Preview end user experience"
          />
        }
      />
      {renderContent()}
    </section>
  );
};

export default EndUserAuthentication;
