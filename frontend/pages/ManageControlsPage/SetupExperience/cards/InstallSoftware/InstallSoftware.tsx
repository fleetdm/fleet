import React, { useCallback } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import mdmAPI, {
  IGetSetupExperienceSoftwareResponse,
} from "services/entities/mdm";
import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { ISoftwareTitle } from "interfaces/software";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";
import { IConfig } from "interfaces/config";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import {
  isSetupExperiencePlatform,
  SetupExperiencePlatform,
} from "interfaces/platform";

import SectionHeader from "components/SectionHeader";
import PageDescription from "components/PageDescription";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import GenericMsgWithNavButton from "components/GenericMsgWithNavButton";
import CustomLink from "components/CustomLink";

import InstallSoftwareForm from "./components/InstallSoftwareForm";
import SetupExperienceContentContainer from "../../components/SetupExperienceContentContainer";
import { ISetupExperienceCardProps } from "../../SetupExperienceNavItems";
import getManualAgentInstallSetting from "../../helpers";

const baseClass = "install-software";

// This is so large because we want to get all the software titles that are
// available for install so we can correctly display the selected count.
const PER_PAGE_SIZE = 3000;

export const PLATFORM_BY_INDEX: SetupExperiencePlatform[] = [
  "macos",
  "windows",
  "linux",
  "ios",
  "ipados",
  "android",
];
export interface InstallSoftwareLocation {
  pathname: string;
  query: {
    fleet_id?: string;
  };
}

const InstallSoftware = ({
  currentTeamId,
  router,
  urlPlatformParam,
}: ISetupExperienceCardProps) => {
  const isValidPlatform = isSetupExperiencePlatform(urlPlatformParam);

  // all uses of selectedPlatform are gated by above boolean
  const selectedPlatform = urlPlatformParam as SetupExperiencePlatform;

  const {
    data: softwareTitles,
    isLoading: isLoadingSoftwareTitles,
    isError,
    refetch: refetchSoftwareTitles,
  } = useQuery<
    IGetSetupExperienceSoftwareResponse,
    AxiosError,
    ISoftwareTitle[] | null
  >(
    ["install-software", currentTeamId, selectedPlatform],
    () =>
      mdmAPI.getSetupExperienceSoftware({
        platform: selectedPlatform,
        team_id: currentTeamId,
        per_page: PER_PAGE_SIZE,
      }),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (res) => res.software_titles,
      enabled: isValidPlatform,
    }
  );

  const { data: globalConfig, isLoading: isLoadingGlobalConfig } = useQuery<
    IConfig,
    Error
  >(["config", currentTeamId], () => configAPI.loadAll(), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: isValidPlatform,
  });

  const { data: teamConfig, isLoading: isLoadingTeamConfig } = useQuery<
    ILoadTeamResponse,
    Error,
    ITeamConfig
  >(["team", currentTeamId], () => teamsAPI.load(currentTeamId), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: isValidPlatform && currentTeamId !== API_NO_TEAM_ID,
    select: (res) => res.team,
  });

  const handleTabChange = useCallback(
    (index: number) => {
      const newPlatform = PLATFORM_BY_INDEX[index];
      router.push(
        getPathWithQueryParams(PATHS.CONTROLS_INSTALL_SOFTWARE(newPlatform), {
          fleet_id: currentTeamId,
        })
      );
    },
    [router]
  );

  if (!isValidPlatform) {
    router.replace(
      getPathWithQueryParams(PATHS.CONTROLS_INSTALL_SOFTWARE("macos"), {
        fleet_id: currentTeamId,
      })
    );
  }

  const hasManualAgentInstall = getManualAgentInstallSetting(
    currentTeamId,
    globalConfig,
    teamConfig
  );

  const isAndroidMdmEnabled = globalConfig?.mdm.android_enabled_and_configured;

  const isLoadingConfig = isLoadingGlobalConfig || isLoadingTeamConfig;

  const renderTabContent = (platform: SetupExperiencePlatform) => {
    if (isLoadingSoftwareTitles) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError />;
    }

    if (softwareTitles || softwareTitles === null) {
      const appleMdmAndAbmEnabled =
        globalConfig?.mdm.enabled_and_configured &&
        globalConfig?.mdm.apple_bm_enabled_and_configured;
      const turnOnAppleMdm =
        (platform === "macos" || platform === "ios" || platform === "ipados") &&
        !appleMdmAndAbmEnabled;

      const turnOnAndroidMdm = platform === "android" && !isAndroidMdmEnabled;

      // Only Apple and Android setup experience require MDM
      const turnOnMdm = turnOnAppleMdm || turnOnAndroidMdm;

      return (
        <SetupExperienceContentContainer>
          <PageDescription content="Install software on hosts that automatically enroll to Fleet." />
          {turnOnMdm ? (
            <GenericMsgWithNavButton
              header={
                platform === "android"
                  ? "Turn on Android MDM"
                  : "Additional configuration required"
              }
              info={
                platform === "android"
                  ? "Turn on MDM to install software during setup experience."
                  : "Turn on MDM and automatic enrollment to install software during setup experience."
              }
              buttonText="Turn on"
              path={PATHS.ADMIN_INTEGRATIONS_MDM}
              router={router}
            />
          ) : (
            <InstallSoftwareForm
              currentTeamId={currentTeamId}
              hasManualAgentInstall={hasManualAgentInstall}
              softwareTitles={softwareTitles}
              platform={platform}
              savedRequireAllSoftwareMacOS={
                currentTeamId
                  ? teamConfig?.mdm?.macos_setup?.require_all_software_macos
                  : globalConfig?.mdm?.macos_setup?.require_all_software_macos
              }
              router={router}
              refetchSoftwareTitles={refetchSoftwareTitles}
            />
          )}
        </SetupExperienceContentContainer>
      );
    }

    return null;
  };

  return (
    <section className={baseClass}>
      <SectionHeader
        title="Install software"
        details={
          <CustomLink
            newTab
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/setup-experience/install-software`}
            text="Preview end user experience"
          />
        }
      />
      {isLoadingConfig ? (
        <Spinner />
      ) : (
        <TabNav secondary>
          <Tabs
            selectedIndex={PLATFORM_BY_INDEX.indexOf(selectedPlatform)}
            onSelect={handleTabChange}
          >
            <TabList>
              <Tab>
                <TabText>macOS</TabText>
              </Tab>
              <Tab>
                <TabText>Windows</TabText>
              </Tab>
              <Tab>
                <TabText>Linux</TabText>
              </Tab>
              <Tab>
                <TabText>iOS</TabText>
              </Tab>
              <Tab>
                <TabText>iPadOS</TabText>
              </Tab>
              <Tab>
                <TabText>Android</TabText>
              </Tab>
            </TabList>
            {PLATFORM_BY_INDEX.map((platform) => {
              return (
                <TabPanel key={platform}>{renderTabContent(platform)}</TabPanel>
              );
            })}
          </Tabs>
        </TabNav>
      )}
    </section>
  );
};

export default InstallSoftware;
