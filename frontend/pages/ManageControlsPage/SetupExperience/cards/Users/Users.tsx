import React from "react";
import { useQuery } from "react-query";

import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { IConfig, IMdmConfig } from "interfaces/config";
import { ITeamConfig } from "interfaces/team";

import SectionHeader from "components/SectionHeader/SectionHeader";
import PageDescription from "components/PageDescription";
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
      <UsersForm
        currentTeamId={currentTeamId}
        defaultIsEndUserAuthEnabled={defaultIsEndUserAuthEnabled}
        defaultLockEndUserInfo={defaultLockEndUserInfo}
        defaultEnableManagedLocalAccount={defaultEnableManagedLocalAccount}
        isIdPConfigured={isIdPConfigured(mdmConfig)}
      />
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
      <PageDescription
        variant="right-panel"
        content={
          <>
            Customize local user accounts. You can automatically create local
            user accounts using IdP credentials (PSSO).{" "}
            <CustomLink
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/psso-local-account`}
              text="Learn how"
              newTab
            />
          </>
        }
      />
      <SetupExperienceContentContainer>
        {renderContent()}
      </SetupExperienceContentContainer>
    </section>
  );
};

export default Users;
