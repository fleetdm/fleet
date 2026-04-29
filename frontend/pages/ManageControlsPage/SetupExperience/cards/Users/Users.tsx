import React from "react";
import PATHS from "router/paths";
import { useQuery } from "react-query";

import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import mdmAppleBmAPI, {
  IGetAbmTokensResponse,
} from "services/entities/mdm_apple_bm";
import { IConfig, IMdmConfig } from "interfaces/config";
import { IMdmAbmToken } from "interfaces/mdm";
import { ITeamConfig } from "interfaces/team";

import SectionHeader from "components/SectionHeader/SectionHeader";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import UsersForm from "./components/UsersForm/UsersForm";
import SetupExperienceContentContainer from "../../components/SetupExperienceContentContainer";
import { ISetupExperienceCardProps } from "../../SetupExperienceNavItems";

const baseClass = "setup-experience-users";

const getEnabledManagedLocalAccount = (
  currentTeamId: number,
  globalConfig?: IConfig,
  teamConfig?: ITeamConfig
) => {
  if (globalConfig === undefined && teamConfig === undefined) {
    return false;
  }

  if (currentTeamId === 0) {
    return (
      globalConfig?.mdm?.setup_experience?.enable_create_local_admin_account ??
      false
    );
  }

  return (
    teamConfig?.mdm?.setup_experience?.enable_create_local_admin_account ??
    false
  );
};

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
      globalConfig?.mdm?.setup_experience.enable_end_user_authentication ??
      false
    );
  }

  return (
    teamConfig?.mdm?.setup_experience.enable_end_user_authentication ?? false
  );
};

const getLockEndUserInfo = (
  currentTeamId: number,
  globalConfig?: IConfig,
  teamConfig?: ITeamConfig
) => {
  if (globalConfig === undefined && teamConfig === undefined) {
    return false;
  }

  // team is "No team" when currentTeamId === 0
  if (currentTeamId === 0) {
    return globalConfig?.mdm?.setup_experience.lock_end_user_info ?? false;
  }

  return teamConfig?.mdm?.setup_experience.lock_end_user_info ?? false;
};

const isIdPConfigured = ({
  end_user_authentication: idp,
}: Pick<IMdmConfig, "end_user_authentication">) => {
  return (
    !!idp.entity_id && !!idp.idp_name && (!!idp.metadata_url || !!idp.metadata)
  );
};

/** Returns true if any ABM token assigns the given team for macOS, iOS, or iPadOS.
 *  For "No team" (id 0), checks for team_id 0 which represents unassigned hosts.
 *
 *  Note: This is a per-team ADE check. Other Setup Experience cards (SetupAssistant,
 *  BootstrapPackage, RunScript, InstallSoftware) gate on the global
 *  apple_bm_enabled_and_configured flag instead. Aligning those cards to use
 *  per-team ADE checking would be a potential improvement for consistency. */
export const isAdeConfiguredForTeam = (
  teamId: number,
  abmTokens?: IMdmAbmToken[]
) => {
  if (!abmTokens?.length) return false;
  return abmTokens.some(
    (token) =>
      token.macos_team.team_id === teamId ||
      token.ios_team.team_id === teamId ||
      token.ipados_team.team_id === teamId
  );
};

const Users = ({ currentTeamId, router }: ISetupExperienceCardProps) => {
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
    select: (res) => res.fleet,
  });

  const { data: abmTokensData } = useQuery<IGetAbmTokensResponse>(
    ["abm_tokens"],
    () => mdmAppleBmAPI.getTokens(),
    {
      refetchOnWindowFocus: false,
      retry: false,
      enabled: !!globalConfig?.mdm.apple_bm_enabled_and_configured,
    }
  );

  const teamHasAde = isAdeConfiguredForTeam(
    currentTeamId,
    abmTokensData?.abm_tokens
  );

  const defaultIsEndUserAuthEnabled = getEnabledEndUserAuth(
    currentTeamId,
    globalConfig,
    teamConfig
  );

  const defaultLockEndUserInfo = getLockEndUserInfo(
    currentTeamId,
    globalConfig,
    teamConfig
  );

  const defaultEnableManagedLocalAccount = getEnabledManagedLocalAccount(
    currentTeamId,
    globalConfig,
    teamConfig
  );

  const renderContent = () => {
    if (!globalConfig || isLoadingGlobalConfig || isLoadingTeamConfig) {
      return <Spinner />;
    }
    const mdmConfig = globalConfig.mdm;
    return (
      <SetupExperienceContentContainer>
        <UsersForm
          currentTeamId={currentTeamId}
          defaultIsEndUserAuthEnabled={defaultIsEndUserAuthEnabled}
          defaultLockEndUserInfo={defaultLockEndUserInfo}
          defaultEnableManagedLocalAccount={defaultEnableManagedLocalAccount}
          isIdPConfigured={isIdPConfigured(mdmConfig)}
          teamHasAde={teamHasAde}
        />
      </SetupExperienceContentContainer>
    );
  };

  return (
    <section className={baseClass}>
      <SectionHeader
        title="Users"
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

export default Users;
