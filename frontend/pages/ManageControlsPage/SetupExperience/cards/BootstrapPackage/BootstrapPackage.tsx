import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { AxiosResponse } from "axios";

import { IBootstrapPackageMetadata } from "interfaces/mdm";
import { IApiError } from "interfaces/errors";
import { IConfig } from "interfaces/config";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import mdmAPI from "services/entities/mdm";
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
import SetupExperienceContentContainer from "../../components/SetupExperienceContentContainer";
import BootstrapAdvancedOptions from "./components/BootstrapAdvancedOptions";

const baseClass = "bootstrap-package";

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
    showDeleteBootstrapPackageModal,
    setShowDeleteBootstrapPackageModal,
  ] = useState(false);

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

  const {
    data: bootstrapMetadata,
    isLoading,
    error,
    refetch: refretchBootstrapMetadata,
  } = useQuery<
    IBootstrapPackageMetadata,
    AxiosResponse<IApiError>,
    IBootstrapPackageMetadata
  >(
    ["bootstrap-metadata", currentTeamId],
    () => mdmAPI.getBootstrapPackageMetadata(currentTeamId),
    {
      retry: false,
      refetchOnWindowFocus: false,
      cacheTime: 0,
    }
  );

  const onUpload = () => {
    refretchBootstrapMetadata();
  };

  const onDelete = async () => {
    try {
      await mdmAPI.deleteBootstrapPackage(currentTeamId);
      renderFlash("success", "Successfully deleted!");
    } catch {
      renderFlash("error", "Couldn't delete. Please try again.");
    } finally {
      setShowDeleteBootstrapPackageModal(false);
      refretchBootstrapMetadata();
    }
  };

  const defaultManualInstallSetting = getManualAgentInstallSetting(
    currentTeamId,
    globalConfig,
    teamConfig
  );

  // we are relying on the API to tell us this resource does not exist to
  // determine if the user has uploaded a bootstrap package.
  const noPackageUploaded =
    (error && error.status === 404) || !bootstrapMetadata;

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
            enableInstallManually={!noPackageUploaded}
            defaultManualInstall={defaultManualInstallSetting}
          />
        </div>
        <div className={`${baseClass}__preview-container`}>
          <BootstrapPackagePreview />
        </div>
      </SetupExperienceContentContainer>
    );
  };

  return (
    <section className={baseClass}>
      <SectionHeader title="Bootstrap package" />
      {isLoading || isLoadingGlobalConfig || isLoadingTeamConfig ? (
        <Spinner />
      ) : (
        renderBootstrapView()
      )}
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
