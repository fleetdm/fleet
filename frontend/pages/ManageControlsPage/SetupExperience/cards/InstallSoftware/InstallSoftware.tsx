import React, { useCallback, useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import PATHS from "router/paths";

import mdmAPI, {
  IGetSetupExperienceSoftwareResponse,
} from "services/entities/mdm";
import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { ISoftwareTitle } from "interfaces/software";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { IConfig } from "interfaces/config";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import {
  isSetupExperiencePlatform,
  SetupExperiencePlatform,
} from "interfaces/platform";

import SectionHeader from "components/SectionHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import TurnOnMdmMessage from "components/TurnOnMdmMessage";

import InstallSoftwarePreview from "./components/InstallSoftwarePreview";
import AddInstallSoftware from "./components/AddInstallSoftware";
import SelectSoftwareModal from "./components/SelectSoftwareModal";
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
];
export interface InstallSoftwareLocation {
  search: string;
  pathname: string;
  query: {
    team_id?: string;
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

  const [showSelectSoftwareModal, setShowSelectSoftwareModal] = useState(false);

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
        PATHS.CONTROLS_INSTALL_SOFTWARE(newPlatform).concat(
          location?.search ?? ""
        )
      );
    },
    [router]
  );

  if (!isValidPlatform) {
    router.replace(
      PATHS.CONTROLS_INSTALL_SOFTWARE("macos").concat(location?.search ?? "")
    );
  }

  const onSave = async () => {
    setShowSelectSoftwareModal(false);
    refetchSoftwareTitles();
  };

  const hasManualAgentInstall = getManualAgentInstallSetting(
    currentTeamId,
    globalConfig,
    teamConfig
  );

  const renderTabContent = (platform: SetupExperiencePlatform) => {
    if (
      isLoadingSoftwareTitles ||
      isLoadingGlobalConfig ||
      isLoadingTeamConfig
    ) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError />;
    }

    if (softwareTitles || softwareTitles === null) {
      const appleMdmAndAbmEnabled =
        globalConfig?.mdm.enabled_and_configured &&
        globalConfig?.mdm.apple_bm_enabled_and_configured;
      const turnOnAppleMdm = platform === "macos" && !appleMdmAndAbmEnabled;

      const turnOnWindowsMdm =
        platform === "windows" &&
        !globalConfig?.mdm.windows_enabled_and_configured;

      const turnOnMdm = turnOnAppleMdm || turnOnWindowsMdm;

      if (turnOnMdm) {
        return (
          <TurnOnMdmMessage
            header="Additional configuration required"
            info="To customize, first turn on automatic enrollment."
            buttonText="Turn on"
            router={router}
          />
        );
      }
      return (
        <SetupExperienceContentContainer>
          <AddInstallSoftware
            currentTeamId={currentTeamId}
            hasManualAgentInstall={hasManualAgentInstall}
            softwareTitles={softwareTitles}
            onAddSoftware={() => setShowSelectSoftwareModal(true)}
            platform={platform}
            savedRequireAllSoftwareMacOS={
              teamConfig?.mdm?.macos_setup?.require_all_software_macos
            }
          />
          <InstallSoftwarePreview platform={platform} />
        </SetupExperienceContentContainer>
      );
    }

    return null;
  };

  return (
    <section className={baseClass}>
      <SectionHeader title="Install software" />
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
          </TabList>
          <TabPanel>{renderTabContent(PLATFORM_BY_INDEX[0])}</TabPanel>
          <TabPanel>{renderTabContent(PLATFORM_BY_INDEX[1])}</TabPanel>
          <TabPanel>{renderTabContent(PLATFORM_BY_INDEX[2])}</TabPanel>
        </Tabs>
      </TabNav>
      {showSelectSoftwareModal && softwareTitles && (
        <SelectSoftwareModal
          currentTeamId={currentTeamId}
          softwareTitles={softwareTitles}
          platform={selectedPlatform}
          onSave={onSave}
          onExit={() => setShowSelectSoftwareModal(false)}
        />
      )}
    </section>
  );
};

export default InstallSoftware;
