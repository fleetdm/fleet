import React, { useCallback, useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import mdmAPI, {
  IGetSetupExperienceSoftwareResponse,
} from "services/entities/mdm";
import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { ISoftwareTitle } from "interfaces/software";
import { DEFAULT_USE_QUERY_OPTIONS, SUPPORT_LINK } from "utilities/constants";
import { IConfig } from "interfaces/config";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import { SetupExperiencePlatform } from "interfaces/platform";

import SectionHeader from "components/SectionHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import CustomLink from "components/CustomLink";

import InstallSoftwarePreview from "./components/InstallSoftwarePreview";
import AddInstallSoftware from "./components/AddInstallSoftware";
import SelectSoftwareModal from "./components/SelectSoftwareModal";
import SetupExperienceContentContainer from "../../components/SetupExperienceContentContainer";
import { getManualAgentInstallSetting } from "../BootstrapPackage/BootstrapPackage";

const baseClass = "install-software";

// This is so large because we want to get all the software titles that are
// available for install so we can correctly display the selected count.
const PER_PAGE_SIZE = 3000;

const DEFAULT_PLATFORM: SetupExperiencePlatform = "macos";

export const PLATFORM_BY_INDEX: SetupExperiencePlatform[] = [
  "macos",
  "windows",
  "linux",
];

interface IInstallSoftwareProps {
  currentTeamId: number;
}

const InstallSoftware = ({ currentTeamId }: IInstallSoftwareProps) => {
  const [showSelectSoftwareModal, setShowSelectSoftwareModal] = useState(false);
  const [
    selectedPlatform,
    setSelectedPlatform,
  ] = useState<SetupExperiencePlatform>(DEFAULT_PLATFORM);

  const {
    data: softwareTitles,
    isLoading,
    isError,
    refetch: refetchSoftwareTitles,
  } = useQuery<
    IGetSetupExperienceSoftwareResponse,
    AxiosError,
    ISoftwareTitle[] | null
  >(
    ["install-software", currentTeamId, selectedPlatform],
    () =>
      mdmAPI.getSetupExperienceSoftware(selectedPlatform, {
        team_id: currentTeamId,
        per_page: PER_PAGE_SIZE,
      }),
    {
      enabled: selectedPlatform !== "windows", // remove next iteration
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (res) => res.software_titles,
    }
  );

  const { data: globalConfig, isLoading: isLoadingGlobalConfig } = useQuery<
    IConfig,
    Error
  >(["config", currentTeamId], () => configAPI.loadAll(), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: currentTeamId === API_NO_TEAM_ID,
  });

  const { data: teamConfig, isLoading: isLoadingTeamConfig } = useQuery<
    ILoadTeamResponse,
    Error,
    ITeamConfig
  >(["team", currentTeamId], () => teamsAPI.load(currentTeamId), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: currentTeamId !== API_NO_TEAM_ID,
    select: (res) => res.team,
  });

  const onSave = async () => {
    setShowSelectSoftwareModal(false);
    refetchSoftwareTitles();
  };

  const handleTabChange = useCallback((index: number) => {
    setSelectedPlatform(PLATFORM_BY_INDEX[index]);
  }, []);

  const hasManualAgentInstall = getManualAgentInstallSetting(
    currentTeamId,
    globalConfig,
    teamConfig
  );

  const renderTabContent = (platform: SetupExperiencePlatform) => {
    if (platform === "windows") {
      return (
        <div className={`${baseClass}__windows`}>
          <b>Windows setup experience is coming soon.</b>
          <p>
            Need to customize setup for Windows users?{" "}
            <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
          </p>
        </div>
      );
    }
    if (isLoading || isLoadingGlobalConfig || isLoadingTeamConfig) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError />;
    }

    if (softwareTitles || softwareTitles === null) {
      return (
        <SetupExperienceContentContainer>
          <AddInstallSoftware
            currentTeamId={currentTeamId}
            hasManualAgentInstall={hasManualAgentInstall}
            softwareTitles={softwareTitles}
            onAddSoftware={() => setShowSelectSoftwareModal(true)}
            platform={platform}
          />
          <InstallSoftwarePreview />
        </SetupExperienceContentContainer>
      );
    }

    return null;
  };

  return (
    <section className={baseClass}>
      <SectionHeader title="Install software" />
      <TabNav>
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
