import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { AxiosError, AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import { IConfig } from "interfaces/config";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import { ISoftwareTitle } from "interfaces/software";
import mdmAPI, {
  IGetBootstrapPackageMetadataResponse,
  IGetSetupExperienceScriptResponse,
  IGetSetupExperienceSoftwareResponse,
} from "services/entities/mdm";
import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { NotificationContext } from "context/notification";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";

import BootstrapPackagePreview from "./components/BootstrapPackagePreview";
import PackageUploader from "./components/BootstrapPackageUploader";
import UploadedPackageView from "./components/UploadedPackageView";
import DeleteBootstrapPackageModal from "./components/DeleteBootstrapPackageModal";
import BootstrapAdvancedOptions from "./components/BootstrapAdvancedOptions";
import SetupExperienceContentContainer from "../../components/SetupExperienceContentContainer";
import { getInstallSoftwareDuringSetupCount } from "../InstallSoftware/components/AddInstallSoftware/helpers";

const baseClass = "bootstrap-package";

// This is so large because we want to get all the software titles that are
// available for install so we can correctly display the selected count.
const PER_PAGE_SIZE = 3000;

export const getManualAgentInstallSetting = (
  currentTeamId: number,
  globalConfig?: IConfig,
  teamConfig?: ITeamConfig
) => {
  if (currentTeamId === API_NO_TEAM_ID) {
    return globalConfig?.mdm.macos_setup.manual_agent_install || false;
  }
  return teamConfig?.mdm?.macos_setup.manual_agent_install || false;
};

interface IBootstrapPackageProps {
  currentTeamId: number;
}

const BootstrapPackage = ({ currentTeamId }: IBootstrapPackageProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [
    selectedManualAgentInstall,
    setSelectedManualAgentInstall,
  ] = useState<boolean>(false);
  const [
    showDeleteBootstrapPackageModal,
    setShowDeleteBootstrapPackageModal,
  ] = useState(false);

  const { data: softwareTitles, isLoading: isLoadingSoftware } = useQuery<
    IGetSetupExperienceSoftwareResponse,
    AxiosError,
    ISoftwareTitle[] | null
  >(
    ["install-software", currentTeamId],
    () =>
      mdmAPI.getSetupExperienceSoftware({
        team_id: currentTeamId,
        per_page: PER_PAGE_SIZE,
      }),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (res) => res.software_titles,
    }
  );

  const { data: script, isLoading: isLoadingScript } = useQuery<
    IGetSetupExperienceScriptResponse,
    AxiosError
  >(
    ["setup-experience-script", currentTeamId],
    () => mdmAPI.getSetupExperienceScript(currentTeamId),
    { ...DEFAULT_USE_QUERY_OPTIONS }
  );

  const {
    isLoading: isLoadingGlobalConfig,
    refetch: refetchGlobalConfig,
  } = useQuery<IConfig, Error>(
    ["config", currentTeamId],
    () => configAPI.loadAll(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: currentTeamId === API_NO_TEAM_ID,
      onSuccess: (data) => {
        setSelectedManualAgentInstall(
          getManualAgentInstallSetting(currentTeamId, data)
        );
      },
    }
  );

  const {
    isLoading: isLoadingTeamConfig,
    refetch: refetchTeamConfig,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["team", currentTeamId],
    () => teamsAPI.load(currentTeamId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: currentTeamId !== API_NO_TEAM_ID,
      select: (res) => res.team,
      onSuccess: (data) => {
        setSelectedManualAgentInstall(
          getManualAgentInstallSetting(currentTeamId, undefined, data)
        );
      },
    }
  );

  const {
    data: bootstrapMetadata,
    isLoading: isloadingBootstrapMetadata,
    error: errorBootstrapMetadata,
    refetch: refretchBootstrapMetadata,
  } = useQuery<IGetBootstrapPackageMetadataResponse, AxiosResponse<IApiError>>(
    ["bootstrap-metadata", currentTeamId],
    () => mdmAPI.getBootstrapPackageMetadata(currentTeamId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      cacheTime: 0,
    }
  );

  const onUpload = () => {
    refretchBootstrapMetadata();
  };

  const onDelete = async () => {
    try {
      await mdmAPI.deleteBootstrapPackage(currentTeamId);
      await mdmAPI.updateSetupExperienceSettings({
        team_id: currentTeamId,
        manual_agent_install: false,
      });
      renderFlash("success", "Successfully deleted!");
    } catch {
      renderFlash("error", "Couldn't delete. Please try again.");
    } finally {
      setShowDeleteBootstrapPackageModal(false);
      refretchBootstrapMetadata();
      if (currentTeamId !== API_NO_TEAM_ID) {
        refetchTeamConfig();
      } else {
        refetchGlobalConfig();
      }
    }
  };

  // we are relying on the API to tell us this resource does not exist to
  // determine if the user has uploaded a bootstrap package.
  const noPackageUploaded =
    (errorBootstrapMetadata && errorBootstrapMetadata.status === 404) ||
    !bootstrapMetadata;
  const hasSetupExperienceInstallSoftware =
    getInstallSoftwareDuringSetupCount(softwareTitles) !== 0;
  const hasSetupExperienceScript = !!script;

  const renderBootstrapView = () => {
    const bootstrapPackageView = noPackageUploaded ? (
      <PackageUploader currentTeamId={currentTeamId} onUpload={onUpload} />
    ) : (
      <UploadedPackageView
        bootstrapPackage={bootstrapMetadata}
        currentTeamId={currentTeamId}
        onDelete={() => setShowDeleteBootstrapPackageModal(true)}
      />
    );

    return (
      <SetupExperienceContentContainer className={`${baseClass}__content`}>
        <div className={`${baseClass}__uploader-container`}>
          {bootstrapPackageView}
          <BootstrapAdvancedOptions
            currentTeamId={currentTeamId}
            disableInstallManually={
              noPackageUploaded ||
              hasSetupExperienceInstallSoftware ||
              hasSetupExperienceScript
            }
            selectManualAgentInstall={selectedManualAgentInstall}
            onChange={(manualAgentInstall) => {
              setSelectedManualAgentInstall(manualAgentInstall);
            }}
          />
        </div>
        <div className={`${baseClass}__preview-container`}>
          <BootstrapPackagePreview />
        </div>
      </SetupExperienceContentContainer>
    );
  };

  const isLoading =
    isloadingBootstrapMetadata ||
    isLoadingGlobalConfig ||
    isLoadingTeamConfig ||
    isLoadingScript ||
    isLoadingSoftware;

  return (
    <section className={baseClass}>
      <SectionHeader title="Bootstrap package" />
      {isLoading ? <Spinner /> : renderBootstrapView()}
      {showDeleteBootstrapPackageModal && (
        <DeleteBootstrapPackageModal
          onDelete={onDelete}
          onCancel={() => setShowDeleteBootstrapPackageModal(false)}
        />
      )}
    </section>
  );
};

export default BootstrapPackage;
