import React from "react";
import { useQuery } from "react-query";

import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { IConfig } from "interfaces/config";
import { APP_CONTEXT_NO_TEAM_ID, ITeamConfig } from "interfaces/team";

import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";
import CustomLink from "components/CustomLink";
import { isEndUserIdPConfigured } from "utilities/permissions/permissions";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import PageDescription from "components/PageDescription";
import { EndUserLocalAccountType } from "interfaces/mdm";

import UsersForm from "./components/UsersForm/UsersForm";
import { ISetupExperienceCardProps } from "../../SetupExperienceNavItems";
import SetupExperienceContentContainer from "../../components/SetupExperienceContentContainer";

const baseClass = "setup-experience-users";

type EnabledManagedLocalAccountConfig = {
  managed_local_account: boolean;
  local_account_type?: EndUserLocalAccountType;
};

const getEnabledManagedLocalAccount = (
  currentTeamId: number,
  globalConfig?: IConfig,
  teamConfig?: ITeamConfig
): EnabledManagedLocalAccountConfig => {
  if (globalConfig === undefined && teamConfig === undefined) {
    return { managed_local_account: false };
  }

  if (currentTeamId === 0) {
    return {
      managed_local_account:
        globalConfig?.mdm?.setup_experience
          ?.enable_create_local_admin_account ?? false,
      local_account_type:
        globalConfig?.mdm?.setup_experience?.end_user_local_account_type,
    };
  }

  return {
    managed_local_account:
      teamConfig?.mdm?.setup_experience?.enable_create_local_admin_account ??
      false,
    local_account_type:
      teamConfig?.mdm?.setup_experience?.end_user_local_account_type,
  };
};

const getEnabledManagedLocalAccountWindows = (
  currentTeamId: number,
  globalConfig?: IConfig,
  teamConfig?: ITeamConfig
): boolean => {
  if (currentTeamId === APP_CONTEXT_NO_TEAM_ID) {
    return (
      globalConfig?.mdm?.windows_settings?.enable_managed_local_account ?? false
    );
  }
  return (
    teamConfig?.mdm?.windows_settings?.enable_managed_local_account ?? false
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

const Users = ({ currentTeamId }: ISetupExperienceCardProps) => {
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

  const managedLocalAccountConfig = getEnabledManagedLocalAccount(
    currentTeamId,
    globalConfig,
    teamConfig
  );

  const enableManagedLocalAccountWindows = getEnabledManagedLocalAccountWindows(
    currentTeamId,
    globalConfig,
    teamConfig
  );

  const renderContent = () => {
    if (!globalConfig || isLoadingGlobalConfig || isLoadingTeamConfig) {
      return <Spinner />;
    }
    return (
      <UsersForm
        currentTeamId={currentTeamId}
        defaultIsEndUserAuthEnabled={defaultIsEndUserAuthEnabled}
        defaultLockEndUserInfo={defaultLockEndUserInfo}
        defaultEnableManagedLocalAccount={
          managedLocalAccountConfig.managed_local_account
        }
        defaultLocalAccountType={managedLocalAccountConfig.local_account_type}
        defaultEnableManagedLocalAccountWindows={
          enableManagedLocalAccountWindows
        }
        isIdPConfigured={isEndUserIdPConfigured(globalConfig)}
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
        className={`${baseClass}__page-description`}
        content={
          <>
            Customize local user accounts. For advanced account configuration,
            like creating local accounts with IdP credentials via Platform
            Single Sign-On (PSSO), use a custom setup.{" "}
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
